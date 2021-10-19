package hyperv

import "github.com/mysteriumnetwork/hyperv-node/common"

func (h *HyperV) ExportVM() error {
	return common.OutWithIt(
		h.shell.Execute(
			"Export-VM",
			"-Name",
			common.WrapInQuotes(h.vmName),
			"-Path",
			common.WrapInQuotes(h.exportPath),
		))
}
