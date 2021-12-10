package hyperv_wmi

import (
	"github.com/gabriel-samfira/go-wmi/virt/network"
	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/go-ole/go-ole"
	"github.com/pkg/errors"
)

const (
	switchName = "Myst Bridge Switch"
)

func (m *Manager) GetSwitch() (*wmi.Result, error) {
	qParams := []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "ElementName", Value: switchName, Type: wmi.Equals}},
	}
	return m.con.GetOne(VMSwitchClass, []string{}, qParams)
}

func (m *Manager) CreateExternalNetworkSwitchIfNotExistsAndAssign() error {
	// check if the switch exists
	_, err := m.GetSwitch()
	if err != nil && !errors.Is(err, wmi.ErrNotFound) {
		return errors.Wrap(err, "GetOne")
	}

	// create switch settings in xml representation
	data, err := m.con.Get(VMSwitchSettings)
	if err != nil {
		return errors.Wrap(err, "Get")
	}
	swInstance, err := data.Get("SpawnInstance_")
	if err != nil {
		return errors.Wrap(err, "SpawnInstance_")
	}
	swInstance.Set("ElementName", switchName)
	swInstance.Set("Notes", []string{"vSwitch for mysterium node"})
	switchText, err := swInstance.GetText(1)
	if err != nil {
		return errors.Wrap(err, "GetText")
	}

	// find external ethernet port and get its device eepPath
	qParams := []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "EnabledState", Value: StateEnabled, Type: wmi.Equals}},
	}
	eep, err := m.con.GetOne(ExternalPort, []string{}, qParams)
	if err != nil {
		return errors.Wrap(err, "GetOne")
	}
	eepPath, err := eep.Path()
	if err != nil {
		return errors.Wrap(err, "Path")
	}

	extPortData, err := m.getDefaultClassValue(PortAllocSetData, network.ETHConnResSubType)
	if err != nil {
		return errors.Wrap(err, "getDefaultClassValue")
	}
	extPortData.Set("ElementName", switchName+" external port")
	extPortData.Set("HostResource", []string{eepPath})
	extPortDataStr, err := extPortData.GetText(1)
	if err != nil {
		return errors.Wrap(err, "GetText")
	}

	// SetInternalPort will create an internal port which will allow the OS to manage this switches network settings.
	hostPath, err := m.getHostPath()
	if err != nil {
		return errors.Wrap(err, "getHostPath")
	}
	intPortData, err := m.getDefaultClassValue(PortAllocSetData, network.ETHConnResSubType)
	if err != nil {
		return errors.Wrap(err, "getDefaultClassValue")
	}
	intPortData.Set("HostResource", []string{hostPath})
	intPortData.Set("Address", "54E1AD8F293B")
	intPortData.Set("ElementName", switchName)
	intPortDataStr, err := intPortData.GetText(1)
	if err != nil {
		return errors.Wrap(err, "GetText")
	}

	// create switch
	jobPath := ole.VARIANT{}
	resultingSystem := ole.VARIANT{}
	jobState, err := m.switchMgr.Get("DefineSystem", switchText, []string{extPortDataStr, intPortDataStr}, nil, &resultingSystem, &jobPath)
	if err != nil {
		return errors.Wrap(err, "DefineSystem")
	}

	return m.waitForJob(jobState, jobPath)
}

func (m *Manager) GetVirtSwitchByName(switchName string) (*wmi.Result, error) {
	qParams := []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "ElementName", Value: switchName, Type: wmi.Equals}},
	}
	return m.con.GetOne(VMSwitchClass, []string{}, qParams)
}
