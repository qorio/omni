package main

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
	omni_auth "github.com/qorio/omni/auth"
	"github.com/qorio/omni/passport"
	"github.com/qorio/omni/runtime"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	instanceId = flag.String("id", "", "Instance id")

	mongoHosts = flag.String("mongo_hosts", "localhost", "Mongo hosts, comma-separated")
	mongoDb    = flag.String("mongo_db", "accounts", "Mongo Db name")

	currentWorkingDir, _ = os.Getwd()

	port      = flag.Int("port", 6060, "Port where server is listening on")
	apiSocket = flag.String("api_socket", "", "File name for domain socket instead of port")
	adminPort = flag.Int("admin_port", 7070, "Port where management server is listening on")

	authKeyFile       = flag.String("auth_public_key_file", "test/authKey.pub", "Auth public key file")
	authTokenTTLHours = flag.Int64("auth_token_ttl_hours", 24, "TTL hours for auth token")
)

func main() {

	flag.Parse()

	buildInfo := runtime.BuildInfo()
	glog.Infoln("Build", buildInfo.Number, "Commit", buildInfo.Commit, "When", buildInfo.Timestamp)

	// the auth service
	key, err := omni_auth.ReadPublicKey(*authKeyFile)
	if err != nil {
		glog.Warningln("Cannot read public key file", *authKeyFile)
		panic(err)
	}
	auth := omni_auth.Init(omni_auth.Settings{
		SignKey:  key,
		TTLHours: time.Duration(*authTokenTTLHours),
	})

	// HTTP endpoints
	shutdownc := make(chan io.Closer, 1)
	go runtime.HandleSignals(shutdownc)

	// Run the http server in a separate go routine
	// When stopping, send a true to the httpDone channel.
	// The channel done is used for getting notification on clean server shutdown.

	// *** The API endpoint ***
	addr := fmt.Sprintf(":%d", *port)
	if *apiSocket != "" {
		addr = *apiSocket
	}
	glog.Infoln("Starting api endpoint")
	apiDone := make(chan bool)
	var apiStopped chan bool

	passportSettings := passport.Settings{
		Mongo: passport.DbSettings{
			Hosts: strings.Split(*mongoHosts, ","),
			Db:    *mongoDb,
		},
	}

	passportService, sErr := passport.NewService(passportSettings)
	if sErr != nil {
		panic(sErr)
	}
	if endpoint, err := passport.NewApiEndPoint(passportSettings, auth, passportService); err == nil {
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
			// Clean up database connections
			glog.Infoln("Stopping database connections")
			passportService.Close()
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
		if count == 2 {
			break
		}
	case <-apiStopped:
		glog.Infoln("Api endpoint stopped.")
		count++
		if count == 2 {
			break
		}
	}
}
