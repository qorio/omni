package main

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/qorio/omni/runtime"
	"io"
	"net/http"
	"os"
)

var (
	port                 = flag.Int("port", 8888, "Port where dashboard server is listening on")
	currentWorkingDir, _ = os.Getwd()
)

type fileSystemWrapper int

// Implements the http.FileSystem interface and try to open a local file.  If not found,
// defer to embedded
func (f *fileSystemWrapper) Open(path string) (file http.File, err error) {
	if file, err = http.Dir(currentWorkingDir + "/www").Open(path); err == nil {
		return
	}
	return //webapp.Dir(".").Open(path)
}

func main() {

	flag.Parse()

	buildInfo := runtime.BuildInfo()
	glog.Infoln("Build", buildInfo.Number, "Commit", buildInfo.Commit, "When", buildInfo.Timestamp)

	shutdownc := make(chan io.Closer, 1)
	go runtime.HandleSignals(shutdownc)

	// Run the http server in a separate go routine
	// When stopping, send a true to the httpDone channel.
	// The channel done is used for getting notification on clean server shutdown.

	// Dashboard web app
	glog.Infoln("Starting dashboard")
	dasherDone := make(chan bool)
	var dasherStopped chan bool
	dasherHttpServer := &http.Server{
		Handler: http.FileServer(new(fileSystemWrapper)),
		Addr:    fmt.Sprintf(":%d", *port),
	}
	dasherStopped = runtime.RunServer(dasherHttpServer, dasherDone)

	// Here is a list of shutdown hooks to execute when receiving the OS signal
	shutdownc <- runtime.ShutdownSequence{
		runtime.ShutdownHook(func() error {
			glog.Infoln("Stopping dashboard")
			dasherDone <- true
			return nil
		}),
	}

	count := 0
	select {
	case <-dasherStopped:
		glog.Infoln("Redirector stopped.")
		count++
		if count == 2 {
			break
		}
	}
}
