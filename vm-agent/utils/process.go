package utils

import (
	"log"
	"os/exec"
)

//
// Runner
// keeps a command running, restarts it if it fails
//

type Runner struct {
	stop chan struct{}
	done chan error
	cmd  *exec.Cmd

	name string
	arg  []string
}

func NewProcessRunner(name string, arg ...string) *Runner {
	r := new(Runner)
	r.name = name
	r.arg = arg
	r.done = make(chan error, 1)
	r.stop = make(chan struct{}, 1)

	return r
}

func (r *Runner) runCmd() error {
	r.cmd = exec.Command(r.name, r.arg...)
	if err := r.cmd.Start(); err != nil {
		log.Println("Start", err)
		return err
	}

	go func() {
		r.done <- r.cmd.Wait()
	}()
	return nil
}

func (r *Runner) Start() error {
	if err := r.runCmd(); err != nil {
		return err
	}

	for {
		select {

		case err := <-r.done:
			if err != nil {
				log.Fatalf("process finished with error = %v", err)
			}
			log.Print("process finished successfully")

			if err := r.runCmd(); err != nil {
				return err
			}

		case <-r.stop:
			err := r.cmd.Process.Kill()
			log.Println("stop", err)
			return nil

		}
	}
}

func (r *Runner) Shutdown() {
	r.stop <- struct{}{}
}
