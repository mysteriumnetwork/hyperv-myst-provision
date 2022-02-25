package utils

import (
	"log"
	"os/exec"
	"time"
)

//
// ProcessRunner
// keeps a command running, restarts it if it fails
//

type ProcessRunner struct {
	action chan string
	done   chan error
	cmd    *exec.Cmd

	isStopped bool
	name      string
	arg       []string
}

func NewProcessRunner() *ProcessRunner {
	log.Println("new runner")

	r := new(ProcessRunner)
	r.done = make(chan error, 1)
	r.action = make(chan string, 1)

	return r
}

func (r *ProcessRunner) SetArgs(name string, arg ...string) {
	r.name = name
	r.arg = arg
}

func (r *ProcessRunner) runCmd() error {
	log.Println("run command:", r.name, r.arg)

	r.cmd = exec.Command(r.name, r.arg...)
	if err := r.cmd.Start(); err != nil {
		log.Println("start", err)
		return err
	}

	go func() {
		r.done <- r.cmd.Wait()
	}()
	return nil
}

func (r *ProcessRunner) Start() error {
	if err := r.runCmd(); err != nil {
		log.Println("runCmd", err)
		return err
	}
	r.isStopped = false

	for {
		select {

		case err := <-r.done:
			if err != nil {
				log.Printf("process finished with error = %v", err)
			} else {
				log.Print("process finished successfully")
			}

			time.Sleep(2 * time.Second)
			if !r.isStopped {
				if err := r.runCmd(); err != nil {
					return err
				}
			}

		case act := <-r.action:
			switch act {
			case "shutdown":
				err := r.cmd.Process.Kill()
				log.Println("shutdown > kill cmd", err)
				return nil

			case "stop":
				r.isStopped = true
				err := r.cmd.Process.Kill()
				log.Println("stop >", err)

				// wait process to finish
				err = <-r.done
				if err != nil {
					log.Printf(">>> process finished with error = %v", err)
				} else {
					log.Print(">>> process finished successfully")
				}

			case "start":
				if r.isStopped == true {
					r.isStopped = false
					if err := r.runCmd(); err != nil {
						log.Print("run cmd error:", err)
					}
				}
			}
		}
	}
}

func (r *ProcessRunner) Shutdown() {
	r.action <- "shutdown"
}

func (r *ProcessRunner) StopCommand() {
	r.action <- "stop"
}

func (r *ProcessRunner) StartCommand() {
	r.action <- "start"
}
