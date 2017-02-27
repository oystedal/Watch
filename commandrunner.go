package main

import (
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"syscall"
	"time"
)

// CommandRunner runs a command everytime a message is received on runChan and
// the command is not already running.
type CommandRunner struct {
	args     []string
	runChan  chan bool
	killChan chan bool

	cmd    *exec.Cmd
	ticker *time.Ticker
	killed bool
}

// NewCommandRunner creates a new CommandRunner, with args as the command to
// execute
func NewCommandRunner(args []string) *CommandRunner {
	cr := &CommandRunner{}

	cr.args = os.Args[1:]
	cr.runChan = make(chan bool)
	cr.killChan = make(chan bool)

	return cr
}

// Run starts executing the CommandRunner
func (cr *CommandRunner) Run() {
	for {
		if cr.cmd == nil {
			if cr.killed {
				return
			}
			// subprocess is not running
			select {
			case <-cr.runChan:
				log.Debug("Received on runChan, starting subprocess")
				cr.createSubprocess()
			case <-cr.killChan:
				return
			}
		} else {
			// subprocess is running
			select {
			case <-cr.runChan:
				continue
			case <-cr.killChan:
				// using -pid here due to setpgid
				if !cr.killed {
					log.Debug("Sending SIGTERM")
					syscall.Kill(-cr.cmd.Process.Pid, syscall.SIGTERM)
					cr.killed = true
				} else {
					log.Debug("Sending SIGKILL")
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
	reflect.ValueOf(&attr).Elem().FieldByName("Setpgid").SetBool(true)

	cr.cmd = exec.Command(cr.args[0], cr.args[1:]...)
	cr.cmd.SysProcAttr = &attr
	cr.cmd.Stdout = os.Stdout

	if err := cr.cmd.Start(); err != nil {
		io.WriteString(os.Stdout, "fatal: "+err.Error()+"\n")
		os.Exit(1)
	}

	log.Infof("[%s]\n", strings.Join(cr.args, ", "))

	cr.ticker = time.NewTicker(5 * time.Millisecond)
}

func (cr *CommandRunner) wait() {
	var status syscall.WaitStatus
	q, err := syscall.Wait4(cr.cmd.Process.Pid, &status, syscall.WNOHANG, nil)

	if err != nil {
		log.Critical(err)
		os.Exit(1)
	} else if q > 0 {
		cr.cmd.Wait()
		cr.ticker.Stop()
		cr.ticker = nil
		cr.cmd = nil
	}
}
