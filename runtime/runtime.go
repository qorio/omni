package runtime

import (
	"fmt"
	"github.com/golang/glog"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func SavePidFile(args ...string) (string, error) {
	cmd := filepath.Base(os.Args[0])
	pidFile, err := os.Create(fmt.Sprintf("%s-%s.pid", cmd, strings.Join(args, "-")))
	if err != nil {
		return "", err
	}

	defer pidFile.Close()
	fmt.Fprintf(pidFile, "%d", os.Getpid())
	return pidFile.Name(), nil
}

type ShutdownSequence []io.Closer

func ShutdownHook(h func() error) closeWrapper {
	return closeWrapper{run: h}
}

type closeWrapper struct {
	run func() error
}

func (w closeWrapper) Close() error {
	return w.run()
}

// Implements io.Closer
func (s ShutdownSequence) Close() (err error) {
	for _, cl := range s {
		if err1 := cl.Close(); err == nil && err1 != nil {
			err = err1
		}
	}
	return
}

func exitf(pattern string, args ...interface{}) {
	if !strings.HasSuffix(pattern, "\n") {
		pattern = pattern + "\n"
	}
	fmt.Fprintf(os.Stderr, pattern, args...)
	os.Exit(1)
}

func HandleSignals(shutdownc <-chan io.Closer) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	signal.Notify(c, syscall.SIGINT)
	for {
		sig := <-c
		sysSig, ok := sig.(syscall.Signal)
		if !ok {
			glog.Fatal("Not a unix signal")
		}
		switch sysSig {
		case syscall.SIGHUP:
		case syscall.SIGINT:
			glog.Warningln("Got SIGTERM: shutting down")
			donec := make(chan bool)
			go func() {
				cl := <-shutdownc
				if err := cl.Close(); err != nil {
					exitf("Error shutting down: %v", err)
				}
				donec <- true
			}()
			select {
			case <-donec:
				glog.Infoln("Shut down completed.")
				os.Exit(0)
			case <-time.After(5 * time.Second):
				exitf("Timeout shutting down. Exiting uncleanly.")
			}
		default:
			glog.Fatal("Received another signal, should not happen.")
		}
	}
}

// Runs the http server.  This server offers more control than the standard go's default http server
// in that when a 'true' is sent to the stop channel, the listener is closed to force a clean shutdown.
func RunServer(server *http.Server, stop chan bool) (stopped chan bool) {
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		panic(err)
	}
	stopped = make(chan bool)

	glog.Infoln("Starting listener at", server.Addr)

	// This will be set to true if a shutdown signal is received. This allows us to detect
	// if the server stop is intentional or due to some error.
	fromSignal := false

	// The main goroutine where the server listens on the network connection
	go func(fromSignal *bool) {
		// Serve will block until an error (e.g. from shutdown, closed connection) occurs.
		err := server.Serve(listener)
		if !*fromSignal {
			glog.Warningln("Warning: server stops due to error", err)
		}
		stopped <- true
	}(&fromSignal)

	// Another goroutine that listens for signal to close the network connection
	// on shutdown.  This will cause the server.Serve() to return.
	go func(fromSignal *bool) {
		select {
		case <-stop:
			listener.Close()
			*fromSignal = true // Intentially stopped from signal
			return
		}
	}(&fromSignal)
	return
}
