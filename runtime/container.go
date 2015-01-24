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

func MinimalContainer(port int, endpoint func() http.Handler, shutdown func() error) {
	start_container(port, endpoint, shutdown, false)
}

func StandardContainer(port int, endpoint func() http.Handler, shutdown func() error) {
	start_container(port, endpoint, shutdown, true)
}

func start_container(port int, endpoint func() http.Handler, shutdown func() error, runManager bool) {
	buildInfo := version.BuildInfo()

	var wg sync.WaitGroup
	shutdownc := make(chan io.Closer, 1)
	go HandleSignals(shutdownc)

	// *** The API endpoint ***
	glog.Infoln("Starting api endpoint")
	apiDone := make(chan bool)
	RunServer(&http.Server{
		Handler: endpoint(),
		Addr:    fmt.Sprintf(":%d", port),
	}, apiDone)

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
	}

	if runManager {
		// *** The Manager endpoint ***
		glog.Infoln("Starting manager endpoint")
		managerDone := make(chan bool)
		RunServer(&http.Server{
			Handler: NewManagerEndPoint(Config{
				BuildInfo: buildInfo,
			}),
			Addr: fmt.Sprintf(":%d", port+1),
		}, managerDone)

		shutdown_tasks = append(shutdown_tasks,
			ShutdownHook(func() error {
				if runManager {
					managerDone <- true
					glog.Infoln("Stopped manager endpoint")
					wg.Done()
				}
				return nil
			}))
	}

	// Pid file
	pid, pidErr := SavePidFile(fmt.Sprintf("%d", port))
	shutdown_tasks = append(shutdown_tasks,
		ShutdownHook(func() error {
			if pidErr == nil {
				os.Remove(pid)
				glog.Infoln("Removed pid file:", pid)
			}
			wg.Done()
			return nil
		}))

	shutdownc <- shutdown_tasks
	wg.Add(len(shutdown_tasks))
	wg.Wait()
}
