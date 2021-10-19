package hyperv

import (
	"fmt"
	"github.com/mysteriumnetwork/hyperv-node/common"
)

func (h *HyperV) SetVMMaxRam(mbMax int) error {
	return common.OutWithIt(
		h.shell.Execute(
			"Set-VMMemory",
			common.WrapInQuotes(h.vmName),
			"-DynamicMemoryEnabled",
			"$false",
			"-StartupBytes",
			mbToString(mbMax),
		))
}

func mbToString(mb int) string {
	return fmt.Sprintf("%dMB", mb)
}
