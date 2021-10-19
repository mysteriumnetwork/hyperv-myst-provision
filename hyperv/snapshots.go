package hyperv

import "github.com/mysteriumnetwork/hyperv-node/common"

func (h *HyperV) RemoveVMSnapshots() error {
	return common.OutWithIt(
		h.shell.Execute(
			`Get-Vm`,
			common.WrapInQuotes(h.vmName),
			`| Remove-VMSnapshot -Name *`,
		))
}
