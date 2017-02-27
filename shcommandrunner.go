package main

import (
	sh "github.com/codeskyblue/go-sh"
	"os"
	"syscall"
)

// ShCommandRunner runs a command everytime a message is received on runChan and
// the command is not already running.
type ShCommandRunner struct {
	args     []string
	runChan  chan bool
	killChan chan bool

	cmd    *sh.Session
	killed bool
}

// NewShCommandRunner creates a new ShCommandRunner, with args as the command to
// execute
func NewShCommandRunner(args []string) *ShCommandRunner {
	cr := &ShCommandRunner{}

	cr.args = os.Args[1:]
	cr.runChan = make(chan bool)
	cr.killChan = make(chan bool)

	return cr
}

// Run starts executing the ShCommandRunner
func (cr *ShCommandRunner) Run() {
	doneChan := make(chan bool)
	for {
		if cr.cmd == nil && cr.killed {
			return
		} else if cr.cmd == nil {
			// subprocess is not running
			select {
			case <-cr.runChan:
				log.Debug("Received on runChan, starting subprocess")
				cr.createSubprocess()
				cr.wait(doneChan)
			case <-cr.killChan:
				return
			}
		} else {
			// subprocess is running
			select {
			case <-cr.runChan:
				continue
			case <-doneChan:
				continue
			case <-cr.killChan:
				if !cr.killed {
					log.Debug("Sending SIGTERM")
					cr.cmd.Kill(syscall.SIGTERM)
					cr.killed = true
				} else {
					log.Debug("Sending SIGKILL")
					cr.cmd.Kill(syscall.SIGKILL)
				}
			}
		}
	}
}

func (cr *ShCommandRunner) createSubprocess() {
	cr.cmd = sh.Command(cr.args[0], cr.args[1:])
	cr.cmd.Stdout = os.Stdout

	if err := cr.cmd.Start(); err != nil {
		log.Fatal(err.Error())
		os.Exit(1)
	}
}

func (cr *ShCommandRunner) wait(done chan bool) {
	go func() {
		cr.cmd.Wait()
		cr.cmd = nil
		done <- true
	}()
}
