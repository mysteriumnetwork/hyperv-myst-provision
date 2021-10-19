package hyperv

import (
	"github.com/mysteriumnetwork/hyperv-node/common"
)

func (h *HyperV) ImportVM() error {
	out, err := h.shell.Execute(
		"gci",
		common.WrapInQuotes(h.fullPath("Virtual Machines")),
		"*.vmcx",
		"-Recurse",
		"| Select Name -ExpandProperty Name -First 1",
	)
	if err != nil {
		return err
	}
	if out.IsErr() {
		return out.GetError()
	}

	fullVMCXPath := h.fullPath("Virtual Machines", out.OutTrimNewLineString())
	out, err = h.shell.Execute(
		"Import-VM",
		"-Path",
		common.WrapInQuotes(fullVMCXPath),
	)
	if err != nil {
		return err
	}
	if out.IsErr() {
		return out.GetError()
	}
	return nil
}
