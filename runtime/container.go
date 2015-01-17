package runtime

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/qorio/omni/version"
	"io"
	"net/http"
	"os"
	"sync"
)

func StandardContainer(port int, endpoint func() http.Handler, shutdown func() error) {
	buildInfo := version.BuildInfo()
	shutdownc := make(chan io.Closer, 1)
	go HandleSignals(shutdownc)

	// *** The API endpoint ***
	glog.Infoln("Starting api endpoint")
	apiDone := make(chan bool)
	RunServer(&http.Server{
		Handler: endpoint(),
		Addr:    fmt.Sprintf(":%d", port),
	}, apiDone)

	// *** The Manager endpoint ***
	glog.Infoln("Starting manager endpoint")
	managerDone := make(chan bool)
	RunServer(&http.Server{
		Handler: NewManagerEndPoint(Config{
			BuildInfo: buildInfo,
		}),
		Addr: fmt.Sprintf(":%d", port+1),
	}, managerDone)

	// Save pid
	pid, pidErr := SavePidFile(fmt.Sprintf("%d", port))

	var wg sync.WaitGroup

	// Here is a list of shutdown hooks to execute when receiving the OS signal
	shutdown_tasks := ShutdownSequence{
		ShutdownHook(func() error {
			if shutdown != nil {
				return shutdown()
			}
			return nil
		}),
		ShutdownHook(func() error {
			apiDone <- true
			glog.Infoln("Stopped api endpoint")
			wg.Done()
			return nil
		}),
		ShutdownHook(func() error {
			managerDone <- true
			glog.Infoln("Stopped manager endpoint")
			wg.Done()
			return nil
		}),
		ShutdownHook(func() error {
			if pidErr == nil {
				os.Remove(pid)
				glog.Infoln("Removed pid file:", pid)
			}
			wg.Done()
			return nil
		}),
	}

	shutdownc <- shutdown_tasks
	wg.Add(len(shutdown_tasks))
	wg.Wait()
}
