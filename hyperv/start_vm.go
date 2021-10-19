package hyperv

import "github.com/mysteriumnetwork/hyperv-node/common"

func (h *HyperV) StartVM() error {
	return common.OutWithIt(h.shell.Execute(
		"Start-VM",
		"-Name",
		common.WrapInQuotes(h.vmName),
	))
}
