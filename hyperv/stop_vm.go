package hyperv

import "github.com/mysteriumnetwork/hyperv-node/common"

func (h *HyperV) StopVM() error {
	return common.OutWithIt(
		h.shell.Execute(
			"Stop-VM",
			"-Name",
			common.WrapInQuotes(h.vmName),
			"-Force",
		))
}
