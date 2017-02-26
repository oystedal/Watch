package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
)

func main() {
	wg := sync.WaitGroup{}
	cancelChan := make(chan bool, 1)
	w, err := NewWatcher(".", cancelChan)
	if err != nil {
		log.Println(err)
		return
	}
	r := NewCommandRunner(os.Args[1:])

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case _, ok := <-w.Updates:
				if !ok {
					return
				}
				r.runChan <- true
			}
		}
	}()

	wg.Add(1)
	go func() {
		w.Run()
		wg.Done()
	}()

	doneChan := make(chan bool)
	wg.Add(1)
	go func() {
		r.Run()
		doneChan <- true
		wg.Done()
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, os.Kill)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-sigChan:
				cancelChan <- true
				r.killChan <- true
			case <-doneChan:
				return
			}
		}
	}()

	wg.Wait()

	signal.Stop(sigChan)

	log.Println("Done")
}
