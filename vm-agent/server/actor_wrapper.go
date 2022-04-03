package server

import (
	"context"
	"fmt"
	"time"

	"github.com/mysteriumnetwork/hyperv-node/vm-agent/utils"
)

type ActorWrapper struct {
	pr     *utils.ProcessRunner
	cancel context.CancelFunc
}

func (a *ActorWrapper) Start() error {
	// wait for keystotre

	if checkKeystore() {
		return a.pr.Start()
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	//defer cancel()

	for {
		select {
		case <-time.After(2 * time.Second):
			if checkKeystore() {
				return a.pr.Start()
			}

		case <-ctx.Done():
			a.pr.Shutdown()
			return nil
		}
	}

}

func (a *ActorWrapper) Stop() error {
	fmt.Println("ActorWrapper > Stop")
	if a.cancel != nil {
		a.cancel()
	}
	return nil
}
