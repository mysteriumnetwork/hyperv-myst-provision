package hyperv

import "github.com/mysteriumnetwork/hyperv-node/common"

func (h *HyperV) CopyVMFile(src, dst string) error {
	return common.OutWithIt(h.shell.Execute(
		`Copy-VMFile -Name`,
		common.WrapInQuotes(h.vmName),
		`-SourcePath`,
		common.WrapInQuotes(src),
		`-DestinationPath`,
		common.WrapInQuotes(dst),
		`-CreateFullPath`,
		`-FileSource Host`,
	))
}
