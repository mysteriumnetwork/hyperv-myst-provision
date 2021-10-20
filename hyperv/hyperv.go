package hyperv

import (
	"github.com/mysteriumnetwork/hyperv-node/powershell"
)

type HyperV struct {
	shell          *powershell.PowerShell
	vmName         string
	workDir        string
	vmSnapshotPath string
	exportPath     string
}

func New(
	vmName, workDir, exportPath string,
	powershell *powershell.PowerShell,
) *HyperV {
	return &HyperV{
		shell:      powershell,
		vmName:     vmName,
		workDir:    workDir,
		exportPath: exportPath,
	}
}

func (h *HyperV) fullPath(and ...string) string {
	fp := h.workDir + `\` + h.vmName

	for _, a := range and {
		fp += `\` + a
	}

	return fp
}
