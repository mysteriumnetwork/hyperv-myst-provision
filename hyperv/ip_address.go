package hyperv

import (
	"errors"
	"fmt"
	"github.com/mysteriumnetwork/hyperv-node/common"
)

var errEmptyIP = errors.New("could not resolve IP address")

func (h *HyperV) VMIP4Address() (string, error) {
	out, err := h.shell.Execute(
		`Get-VM -Name`,
		common.WrapInQuotes(h.vmName),
		`| Where State -eq "Running"`,
		`| select -ExpandProperty networkadapters`,
		`| select -ExpandProperty IPAddresses`,
		`| Select-Object -First 1`,
	)

	if err := common.OutWithIt(out, err); err != nil {
		return "", err
	}

	if out.IsEmpty() {
		return "", fmt.Errorf("VM Name:"+h.vmName+". %w", errEmptyIP)
	}

	return out.OutTrimNewLineString(), nil
}
