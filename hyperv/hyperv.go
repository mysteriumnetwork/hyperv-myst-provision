package hyperv

import (
	"github.com/mysteriumnetwork/hyperv-node/powershell"
)

type HyperV struct {
	shell          *powershell.PowerShell
	vmName         string
	vmPath         string
	vmSnapshotPath string
	exportPath     string
}

func New(
	vmName, vmPath, exportPath string,
	powershell *powershell.PowerShell,
) *HyperV {
	return &HyperV{
		shell:      powershell,
		vmName:     vmName,
		vmPath:     vmPath,
		exportPath: exportPath,
	}
}

func (h *HyperV) fullPath(and ...string) string {
	fp := h.vmPath + `\` + h.vmName

	for _, a := range and {
		fp += `\` + a
	}

	return fp
}
