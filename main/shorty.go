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
	"math"
	"net/http"
	"os"
)

var (
	instanceId = flag.String("id", "", "Instance id")

	redisHost      = flag.String("redis_host", "", "Redis host (leave empty for localhost)")
	redisPort      = flag.Int("redis_port", 6379, "Redis port")
	redisPrefix    = flag.String("redis_prefix", "shorty:", "Redis prefix to use")
	restrictDomain = flag.String("restrict_domain", "", "Restrict destination URLs to a single domain")
	redirect404    = flag.String("redirect_404", "", "Global redirect when no record found")
	urlLength      = flag.Int("url_length", 8, "How many characters should the short code have")
	geoDbFilePath  = flag.String("geo_db", "./GeoLiteCity.dat", "Location to the MaxMind GeoIP city database file")

	currentWorkingDir, _ = os.Getwd()

	tallyRedisHost    = flag.String("tally_redis_host", "", "Redis host (leave empty for localhost)")
	tallyRedisPort    = flag.Int("tally_redis_port", 6379, "Redis port")
	tallyRedisChannel = flag.String("tally_redis_chanel", "shorty", "Redis publish chanel for Events")

	startSubscriber = flag.Bool("start_subscriber", false, "Starts the subscriber to route events to logstash queue")

	logstashInputQueue          = flag.String("logstash_input_queue", "logstash-input", "Logstash input queue name")
	maxLogstashInputQueueLength = flag.Int("logstash_input_queue_max_length", int(math.Pow10(6)), "Qqueue max length")

	port           = flag.Int("port", 8080, "Port where server is listening on")
	apiSocket      = flag.String("api_socket", "", "File name for domain socket instead of port")
	directorSocket = flag.String("redirect_socket", "", "File name for domain socket instead of port")
	adminPort      = flag.Int("admin_port", 7070, "Port where management server is listening on")
)

func translate(r *omni_http.RequestOrigin) (event *tally.Event) {
	event = tally.NewEvent()

	appKey := "shorty"

	requestUrl := r.HttpRequest.URL.String()
	lat := float64(r.Location.Latitude)
	lon := float64(r.Location.Longitude)
	event.AppKey = &appKey
	event.Source = &requestUrl
	event.Context = &requestUrl
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
	event.SetAttribute("country", r.Location.CountryName)
	event.SetAttribute("country_code", r.Location.CountryCode)
	event.SetAttribute("region", r.Location.Region)
	event.SetAttribute("city", r.Location.City)
	event.SetAttribute("postal", r.Location.PostalCode)
	event.SetAttribute("shortcode", r.ShortCode)
	event.SetAttribute("last_visit", r.LastVisit)
	return
}

func translateDecode(decodeEvent *shorty.DecodeEvent) (event *tally.Event) {
	event = translate(decodeEvent.RequestOrigin)
	eventType := "decode"
	event.Type = &eventType
	event.SetAttribute("destination", decodeEvent.Destination)
	event.SetAttribute("uuid", string(decodeEvent.Context))
	event.SetAttribute("origin", decodeEvent.Origin)
	event.SetAttribute("app_key", string(decodeEvent.AppKey))
	event.SetAttribute("campaign_key", string(decodeEvent.CampaignKey))
	event.SetAttributeInt("matched_rule", decodeEvent.MatchedRuleIndex)

	return
}

func translateInstall(installEvent *shorty.InstallEvent) (event *tally.Event) {
	event = translate(installEvent.RequestOrigin)
	eventType := "install"
	event.Type = &eventType
	event.SetAttribute("destination", installEvent.Destination)
	event.SetAttribute("app_url_scheme", string(installEvent.App))
	event.SetAttribute("app_uuid", string(installEvent.AppContext))
	event.SetAttribute("uuid", string(installEvent.SourceContext))
	event.SetAttribute("source_application", installEvent.SourceApplication)
	event.SetAttribute("origin", installEvent.Origin)
	event.SetAttribute("app_key", string(installEvent.AppKey))
	event.SetAttribute("campaign_key", string(installEvent.CampaignKey))
	event.SetAttribute("reporting_method", installEvent.ReportingMethod)
	return
}

func translateAppOpen(appOpenEvent *shorty.AppOpenEvent) (event *tally.Event) {
	event = translate(appOpenEvent.RequestOrigin)
	eventType := "app-open"
	event.Type = &eventType
	event.SetAttribute("app_url_scheme", string(appOpenEvent.App))
	event.SetAttribute("app_uuid", string(appOpenEvent.AppContext))
	event.SetAttribute("uuid", string(appOpenEvent.SourceContext))
	event.SetAttribute("source_application", appOpenEvent.SourceApplication)
	event.SetAttribute("origin", appOpenEvent.Origin)
	event.SetAttribute("app_key", string(appOpenEvent.AppKey))
	event.SetAttribute("campaign_key", string(appOpenEvent.CampaignKey))
	return
}

func translateLink(linkEvent *shorty.LinkEvent) (event *tally.Event) {
	event = translate(linkEvent.RequestOrigin)
	eventType := "link-uuid"
	event.Type = &eventType
	event.SetAttribute("app_url_scheme", string(linkEvent.App))
	event.SetAttribute("uuid1", string(linkEvent.Context1))
	event.SetAttribute("uuid2", string(linkEvent.Context2))
	event.SetAttribute("origin", linkEvent.Origin)
	event.SetAttribute("app_key", string(linkEvent.AppKey))
	event.SetAttribute("campaign_key", string(linkEvent.CampaignKey))
	return
}

func main() {

	flag.Parse()

	buildInfo := runtime.BuildInfo()
	glog.Infoln("Build", buildInfo.Number, "Commit", buildInfo.Commit, "When", buildInfo.Timestamp)

	if *startSubscriber {
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
			glog.Infoln("Subscriber started for channel", *tallyRedisChannel, "sending to queue", *logstashInputQueue)
		}
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
	fromShortyDecode := shortyService.DecodeEventChannel()
	fromShortyInstall := shortyService.InstallEventChannel()
	fromShortyAppOpen := shortyService.AppOpenEventChannel()
	fromShortyLink := shortyService.LinkEventChannel()

	toTally := tallyService.Channel()
	go func() {
		for {
			select {
			case decode := <-fromShortyDecode:
				toTally <- translateDecode(decode)
			case appOpen := <-fromShortyAppOpen:
				toTally <- translateAppOpen(appOpen)
			case install := <-fromShortyInstall:
				toTally <- translateInstall(install)
			case link := <-fromShortyLink:
				toTally <- translateLink(link)
			}

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
	addr := fmt.Sprintf(":%d", *port)
	if *directorSocket != "" {
		addr = *directorSocket
	}
	glog.Infoln("Starting redirector")
	redirectorDone := make(chan bool)
	var redirectorStopped chan bool
	if redirector, err := shorty.NewRedirector(httpSettings, shortyService); err == nil {
		redirectorHttpServer := &http.Server{
			Handler: redirector,
			Addr:    addr,
		}
		redirectorStopped = runtime.RunServer(redirectorHttpServer, redirectorDone)
	} else {
		panic(err)
	}

	// *** The API endpoint ***
	addr = fmt.Sprintf(":%d", *port+1)
	if *apiSocket != "" {
		addr = *apiSocket
	}
	glog.Infoln("Starting api endpoint")
	apiDone := make(chan bool)
	var apiStopped chan bool
	if endpoint, err := shorty.NewApiEndPoint(httpSettings, shortyService); err == nil {
		apiHttpServer := &http.Server{
			Handler: endpoint,
			Addr:    addr,
		}
		apiStopped = runtime.RunServer(apiHttpServer, apiDone)
	} else {
		panic(err)
	}

	// *** The Manager endpoint ***
	glog.Infoln("Starting manager endpoint")
	managerDone := make(chan bool)
	var managerStopped chan bool
	managerHttpServer := &http.Server{
		Handler: runtime.NewManagerEndPoint(),
		Addr:    fmt.Sprintf(":%d", *adminPort),
	}
	managerStopped = runtime.RunServer(managerHttpServer, managerDone)

	// Save pid
	label := fmt.Sprintf("%d", *port)
	if *instanceId != "" {
		label = *instanceId
	}
	pid, pidErr := runtime.SavePidFile(label)

	// Here is a list of shutdown hooks to execute when receiving the OS signal
	shutdownc <- runtime.ShutdownSequence{
		runtime.ShutdownHook(func() error {
			glog.Infoln("Stopping api endpoint")
			apiDone <- true
			return nil
		}),
		runtime.ShutdownHook(func() error {
			glog.Infoln("Stopping manager endpoint")
			managerDone <- true
			return nil
		}),
		runtime.ShutdownHook(func() error {
			glog.Infoln("Stopping redirector")
			redirectorDone <- true
			return nil
		}),
		runtime.ShutdownHook(func() error {
			// Clean up database connections
			glog.Infoln("Stopping database connections")
			shortyService.Close()
			tallyService.Close()
			return nil
		}),
		runtime.ShutdownHook(func() error {
			if pidErr == nil {
				glog.Infoln("Remove pid file:", pid)
				os.Remove(pid)
			}
			return nil
		}),
	}

	count := 0
	select {
	case <-managerStopped:
		glog.Infoln("Manager endpoint stopped.")
		count++
		if count == 3 {
			break
		}
	case <-apiStopped:
		glog.Infoln("Api endpoint stopped.")
		count++
		if count == 3 {
			break
		}
	case <-redirectorStopped:
		glog.Infoln("Redirector stopped.")
		count++
		if count == 3 {
			break
		}
	}
}
