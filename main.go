package main

import (
	"fmt"
	"github.com/op/go-logging"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var log = logging.MustGetLogger("Watch")

var debugFormat = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.0000} %{shortfunc:-15s} ▶ %{level:-5s} %{id:03d}%{color:reset} %{message}`,
)

var normalFormat = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.0000} ▶%{color:reset} %{message}`,
)

func initLogs() {
	loggingBackend := logging.NewLogBackend(os.Stderr, "", 0)
	// formatter := logging.NewBackendFormatter(loggingBackend, debugFormat)
	formatter := logging.NewBackendFormatter(loggingBackend, normalFormat)

	logging.SetBackend(formatter)
	logging.SetLevel(logging.INFO, "Watch")
}

func main() {
	initLogs()

	log.Info("Starting")

	filter := func(path string) bool {
		if len(path) >= 4 && path[0:4] == ".git" {
			return false
		}
		if len(path) >= 6 && path[0:6] == "_build" {
			return false
		}
		return true
	}

	wg := sync.WaitGroup{}
	cancelChan := make(chan bool, 1)
	w, err := NewWatcher(".", cancelChan, filter)
	if err != nil {
		return
	}

	r := NewShCommandRunner(os.Args[1:])

	wg.Add(1)
	go func() {
		defer log.Debug("Exiting filtering goroutine")
		defer wg.Done()
		for range w.Updates {
			r.runChan <- true
		}
	}()

	wg.Add(1)
	go func() {
		defer log.Debug("Exiting watcher goroutine")
		w.Run()
		wg.Done()
	}()

	doneChan := make(chan bool)
	wg.Add(1)
	go func() {
		defer log.Debug("Exiting command runner goroutine")
		r.Run()
		doneChan <- true
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		defer log.Debug("Exiting signal goroutine")
		defer wg.Done()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(sigChan)

		for {
			select {
			case signal := <-sigChan:
				// output a newline to not mess up logging columns
				fmt.Fprintf(os.Stderr, "\n")

				log.Debug("Received signal", signal.String())
				cancelChan <- true
				r.killChan <- true
			case <-doneChan:
				return
			}
		}
	}()

	wg.Wait()

	log.Info("Done")
}
