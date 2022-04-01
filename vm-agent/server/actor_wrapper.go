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
	defer cancel()

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

	return a.pr.Start()
}

func (a *ActorWrapper) Stop() error {
	fmt.Println("ActorWrapper > Stop")
	a.cancel()
	return nil
}
