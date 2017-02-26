package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"reflect"
	"syscall"
	"time"
)

type CommandRunner struct {
	args     []string
	runChan  chan bool
	killChan chan bool

	cmd    *exec.Cmd
	ticker *time.Ticker
	killed bool
}

func NewCommandRunner(args []string) *CommandRunner {
	cr := &CommandRunner{}

	cr.args = os.Args[1:]
	cr.runChan = make(chan bool)
	cr.killChan = make(chan bool)

	return cr
}

func (cr *CommandRunner) Run() {
	for {
		if cr.cmd == nil {
			if cr.killed {
				return
			}
			// subprocess is running
			select {
			case <-cr.runChan:
				log.Println("Received on runChan, starting subprocess")
				cr.createSubprocess()
			case <-cr.killChan:
				return
			}
		} else {
			// subprocess is not running
			select {
			case _, ok := <-cr.runChan:
				if !ok {
					// log.Println("here")
					syscall.Kill(-cr.cmd.Process.Pid, syscall.SIGKILL)
				} else {
					// log.Println("Received on runChan, but already executing")
				}
			case <-cr.killChan:
				// using -pid here due to setpgid
				if !cr.killed {
					log.Println("Sending SIGTERM")
					syscall.Kill(-cr.cmd.Process.Pid, syscall.SIGTERM)
					cr.killed = true
				} else {
					log.Println("Sending SIGKILL")
					syscall.Kill(-cr.cmd.Process.Pid, syscall.SIGKILL)
				}
			case <-cr.ticker.C:
				cr.wait()
			}
		}
	}
}

func (cr *CommandRunner) createSubprocess() {
	var attr syscall.SysProcAttr
	// TODO: Panic if this field is not present
	reflect.ValueOf(&attr).Elem().FieldByName("Setpgid").SetBool(true)

	cr.cmd = exec.Command(cr.args[0], cr.args[1:]...)
	cr.cmd.SysProcAttr = &attr
	cr.cmd.Stdout = os.Stdout

	if err := cr.cmd.Start(); err != nil {
		io.WriteString(os.Stdout, "fatal: "+err.Error()+"\n")
		os.Exit(1)
	}

	cr.ticker = time.NewTicker(5 * time.Millisecond)
}

func (cr *CommandRunner) wait() {
	var status syscall.WaitStatus
	q, err := syscall.Wait4(cr.cmd.Process.Pid, &status, syscall.WNOHANG, nil)

	if err != nil {
		log.Println(err)
		os.Exit(1)
	} else if q > 0 {
		// log.Println("Command has finished executing")
		cr.cmd.Wait() // Clean up any goroutines created by cmd.Start.
		cr.ticker.Stop()
		cr.ticker = nil
		cr.cmd = nil
	}
}
