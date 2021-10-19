package hyperv

import (
	"errors"
	"fmt"
	"time"
)

func (h *HyperV) WaitUntilBooted() error {
	fmt.Printf("waiting for VM `%s` to boot\n", h.vmName)
	for {
		select {
		case <-time.After(5 * time.Second):
			_, err := h.VMIP4Address()
			if !errors.Is(err, errEmptyIP) {
				fmt.Printf("unexpected error while waiting for VM `%s` to boot, %s\n", h.vmName, err)
				return err
			}
		case <-time.After(5 * time.Minute):
			fmt.Printf("time out while waiting for VM `%s` to boot\n", h.vmName)
		}
	}
}
