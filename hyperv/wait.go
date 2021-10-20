package hyperv

import (
	"errors"
	"fmt"
	"time"
)

func (h *HyperV) WaitUntilBooted(pollEvery, timeout time.Duration) error {
	fmt.Printf("waiting for VM `%s` to boot\n", h.vmName)
	for {
		select {
		case <-time.After(pollEvery):
			_, err := h.VMIP4Address()
			if !errors.Is(err, errEmptyIP) {
				fmt.Printf("unexpected error while waiting for VM `%s` to boot, %s\n", h.vmName, err)
				return err
			}
		case <-time.After(timeout):
			fmt.Printf("time out while waiting for VM `%s` to boot\n", h.vmName)
		}
	}
}
