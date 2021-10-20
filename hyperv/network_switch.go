package hyperv

import (
	"errors"
	"github.com/mysteriumnetwork/hyperv-node/common"
	"github.com/mysteriumnetwork/hyperv-node/powershell"
)

const networkSwitchName = "Myst Bridge Switch"

func (h *HyperV) CreateExternalNetworkSwitchIfNotExistsAndAssign() error {
	out, err := h.shell.Execute(
		`Get-VMSwitch `,
		`| Where Name -eq`,
		common.WrapInQuotes(networkSwitchName),
		`| Select -ExpandProperty Name -First 1`,
	)

	if err := common.OutWithIt(out, err); err != nil {
		return err
	}

	exist := !out.IsEmpty()

	if !exist {
		ifName, err := findInterfaceUsedAsInternetGateway(h.shell)
		if err != nil {
			return err
		}

		err = common.OutWithIt(h.shell.Execute(
			"New-VMSwitch",
			common.WrapInQuotes(networkSwitchName),
			"-NetAdapterName",
			common.WrapInQuotes(ifName),
		))

		if err != nil {
			return err
		}
	}

	return common.OutWithIt(h.shell.Execute(
		"Connect-VMNetworkAdapter",
		"-VMName",
		common.WrapInQuotes(h.vmName),
		"-SwitchName",
		common.WrapInQuotes(networkSwitchName),
	))
}

func (h *HyperV) DisconnectVMNetworkSwitch() error {
	return common.OutWithIt(h.shell.Execute(
		`Disconnect-VMNetworkAdapter -VMName`,
		common.WrapInQuotes(h.vmName),
	))
}

func (h *HyperV) RemoveNetworkSwitch() error {
	return common.OutWithIt(h.shell.Execute(
		"Remove-VMSwitch",
		common.WrapInQuotes(networkSwitchName),
	))
}

// TODO this is a poor man's solution for this
func findInterfaceUsedAsInternetGateway(shell *powershell.PowerShell) (string, error) {
	out, err := shell.Execute(
		"Get-NetAdapter -Physical",
		`| Where Status -eq "Up"`,
		"| Sort-Object ifIndex",
		"| Select -ExpandProperty Name -first 1",
	)

	if err := common.OutWithIt(out, err); err != nil {
		return "", err
	}

	interfaceName := out.OutTrimNewLineString()
	if interfaceName == "" {
		return "", errors.New("could not find gateway ethernet adapter")
	}

	return interfaceName, nil
}
