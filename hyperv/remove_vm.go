package hyperv

import "github.com/mysteriumnetwork/hyperv-node/common"

func (h *HyperV) RemoveVM() error {
	err := common.OutWithIt(
		h.shell.Execute(
			"Remove-VM",
			"-Name",
			common.WrapInQuotes(h.vmName),
			"-Force",
		))

	if err != nil {
		return err
	}

	return nil
}
