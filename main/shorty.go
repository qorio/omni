package main

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
	omni_http "github.com/qorio/omni/http"
	"github.com/qorio/omni/runtime"
	"github.com/qorio/omni/shorty"
	"github.com/qorio/omni/tally"
	"io"
	"net/http"
	"os"
)

var (
	redisHost      = flag.String("redis_host", "", "Redis host (leave empty for localhost)")
	redisPort      = flag.Int("redis_port", 6379, "Redis port")
	redisPrefix    = flag.String("redis_prefix", "shorty:", "Redis prefix to use")
	restrictDomain = flag.String("restrict_domain", "", "Restrict destination URLs to a single domain")
	redirect404    = flag.String("redirect_404", "", "Global redirect when no record found")
	urlLength      = flag.Int("url_length", 8, "How many characters should the short code have")
	port           = flag.Int("port", 8080, "Port where server is listening on")
	geoDbFilePath  = flag.String("geo_db", "./GeoLiteCity.dat", "Location to the MaxMind GeoIP city database file")

	currentWorkingDir, _ = os.Getwd()

	tallyRedisHost    = flag.String("tally_redis_host", "", "Redis host (leave empty for localhost)")
	tallyRedisPort    = flag.Int("tally_redis_port", 6379, "Redis port")
	tallyRedisChannel = flag.String("tally_redis_chanel", "shorty", "Redis publish chanel for Events")

	logstashInputQueue          = flag.String("logstash_input_queue", "logstash-input", "Logstash input queue name")
	maxLogstashInputQueueLength = flag.Int("logstash_input_queue_max_length", 1000, "Logstash input queue max length")
)

type fileSystemWrapper int

// Implements the http.FileSystem interface and try to open a local file.  If not found,
// defer to embedded
func (f *fileSystemWrapper) Open(path string) (file http.File, err error) {
	if file, err = http.Dir(currentWorkingDir + "/webapp").Open(path); err == nil {
		return
	}
	return //webapp.Dir(".").Open(path)
}

// Starts a separate server for the web ui.
func startWebUi(port int) {
	http.Handle("/", http.FileServer(new(fileSystemWrapper)))
	webappListen := fmt.Sprintf(":%d", port)
	go func() {
		err := http.ListenAndServe(webappListen, nil)
		if err != nil {
			panic(err)
		}
	}()
}

func translate(r *omni_http.RequestOrigin) (event *tally.Event) {
	event = tally.NewEvent()

	appKey := "shorty"
	eventType := "decode"
	source := "shorty"
	lat := float64(r.Location.Latitude)
	lon := float64(r.Location.Longitude)
	event.AppKey = &appKey
	event.Type = &eventType
	event.Source = &source
	event.Context = &r.HttpRequest.URL.Host
	event.Location = &tally.Location{
		Lat: &lat,
		Lon: &lon,
	}
	event.SetAttribute("ip", r.Ip)
	event.SetAttribute("referrer", r.Referrer)
	event.SetAttributeBool("bot", r.UserAgent.Bot)
	event.SetAttributeBool("mobile", r.UserAgent.Mobile)
	event.SetAttribute("platform", r.UserAgent.Platform)
	event.SetAttribute("os", r.UserAgent.OS)
	event.SetAttribute("make", r.UserAgent.Make)
	event.SetAttribute("browser", r.UserAgent.Browser)
	event.SetAttribute("browserVersion", r.UserAgent.BrowserVersion)
	event.SetAttribute("header", r.UserAgent.Header)
	event.SetAttributeBool("cookied", r.Cookied)
	event.SetAttributeInt("visits", r.Visits)
	return
}

func main() {

	flag.Parse()

	buildInfo := runtime.BuildInfo()
	glog.Infoln("Build", buildInfo.Number, "Commit", buildInfo.Commit, "When", buildInfo.Timestamp)

	// Set up a subscriber service that will subscribe to the channel and
	// push the message to a work queue for indexing
	subscriber, err := tally.InitSubscriber(tally.SubscriberSettings{
		RedisUrl:       fmt.Sprintf("%s:%d", *tallyRedisHost, *tallyRedisPort),
		RedisChannel:   *tallyRedisChannel,
		MaxQueueLength: *maxLogstashInputQueueLength,
	})
	if err != nil {
		glog.Infoln("cannot-start-subscriber", err)
	} else {
		subscriber.Start()
		// Route to a work queue
		subscriber.Queue(*logstashInputQueue, subscriber.Channel())
	}

	// Tally service which publishes the decode events
	tallyService := tally.Init(tally.Settings{
		RedisUrl:     fmt.Sprintf("%s:%d", *tallyRedisHost, *tallyRedisPort),
		RedisChannel: *tallyRedisChannel,
	})
	tallyService.Start()

	shortyService := shorty.Init(shorty.Settings{
		RedisUrl:       fmt.Sprintf("%s:%d", *redisHost, *redisPort),
		RedisPrefix:    *redisPrefix,
		RestrictDomain: *restrictDomain,
		UrlLength:      *urlLength,
	})

	// Wire the service's together ==> this allows the shorty http requests
	// be published to the redis channel
	fromShorty := shortyService.Channel()
	toTally := tallyService.Channel()
	go func() {
		for {
			toTally <- translate(<-fromShorty)
		}
	}()

	// HTTP endpoints
	httpSettings := shorty.ShortyEndPointSettings{
		Redirect404:     *redirect404,
		GeoIpDbFilePath: *geoDbFilePath,
	}

	shutdownc := make(chan io.Closer, 1)
	go runtime.HandleSignals(shutdownc)

	// Run the http server in a separate go routine
	// When stopping, send a true to the httpDone channel.
	// The channel done is used for getting notification on clean server shutdown.

	// *** The main redirector ***
	glog.Infoln("Starting redirector")
	redirectorDone := make(chan bool)
	var redirectorStopped chan bool
	if redirector, err := shorty.NewRedirector(httpSettings, shortyService); err == nil {
		redirectorHttpServer := &http.Server{
			Handler: redirector,
			Addr:    fmt.Sprintf(":%d", *port),
		}
		redirectorStopped = runtime.RunServer(redirectorHttpServer, redirectorDone)
	} else {
		panic(err)
	}

	// *** The API endpoint ***
	glog.Infoln("Starting api endpoint")
	apiDone := make(chan bool)
	var apiStopped chan bool
	if endpoint, err := shorty.NewApiEndPoint(httpSettings, shortyService); err == nil {
		apiHttpServer := &http.Server{
			Handler: endpoint,
			Addr:    fmt.Sprintf(":%d", *port+1),
		}
		apiStopped = runtime.RunServer(apiHttpServer, apiDone)
	} else {
		panic(err)
	}

	// Here is a list of shutdown hooks to execute when receiving the OS signal
	shutdownc <- runtime.ShutdownSequence{
		runtime.ShutdownHook(func() error {
			// Clean up database connections
			glog.Infoln("Stopping database connections")
			shortyService.Close()
			tallyService.Close()
			return nil
		}),
		runtime.ShutdownHook(func() error {
			glog.Infoln("Stopping api endpoint")
			apiDone <- true
			return nil
		}),
		runtime.ShutdownHook(func() error {
			glog.Infoln("Stopping redirector")
			redirectorDone <- true
			return nil
		}),
	}

	count := 0
	select {
	case <-apiStopped:
		glog.Infoln("Api endpoint stopped.")
		count++
		if count == 2 {
			break
		}
	case <-redirectorStopped:
		glog.Infoln("Redirector stopped.")
		count++
		if count == 2 {
			break
		}
	}
}
