package hyperv

import (
	"errors"
	"strings"

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
		ifName, err := findInterfaceUsedAsIPv4InternetGateway(h.shell)
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

func findInterfaceUsedAsIPv4InternetGateway(shell *powershell.PowerShell) (string, error) {
	out, err := shell.Execute(
		"Get-NetAdapter -Physical",
		`| Where Status -eq "Up"`,
		"| Select -ExpandProperty Name",
	)

	if err := common.OutWithIt(out, err); err != nil {
		return "", err
	}

	interfaceNameList := strings.Split(out.OutTrimNewLineString(), "\r\n")
	if len(interfaceNameList) < 1 {
		return "", errors.New("could not find gateway ethernet adapter")
	}

	whereQuery := `$_.InterfaceAlias -eq "` + interfaceNameList[0] + `"`
	for i := 1; i < len(interfaceNameList); i++ {
		whereQuery += ` -or $_.InterfaceAlias -eq "` + interfaceNameList[i] + `"`
	}

	out, err = shell.Execute(
		" Get-NetIpInterface",
		"| Where {"+whereQuery+"}",
		`| Where AddressFamily -eq "IPv4"`,
		"| Sort-Object InterfaceMetric",
		"| Select -ExpandProperty InterfaceAlias -first 1",
	)

	if err := common.OutWithIt(out, err); err != nil {
		return "", err
	}

	usedInterface := out.OutTrimNewLineString()

	return usedInterface, nil
}
