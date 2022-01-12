package hyperv_wmi

import (
	"fmt"
	"sort"

	"github.com/gabriel-samfira/go-wmi/virt/network"
	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/go-ole/go-ole"
	"github.com/pkg/errors"
)

const (
	//defaultSwitchName = "Myst Bridge Switch"
	defaultSwitchName = "Default Switch"
)

func (m *Manager) GetSwitch(switchName string) (*wmi.Result, error) {
	qParams := []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "ElementName", Value: switchName, Type: wmi.Equals}},
	}
	return m.con.GetOne(VMSwitchClass, []string{}, qParams)
}

// Find a network adapter with minimum metric and use it for virtual network switch
func (m *Manager) FindDefaultNetworkAdapter(preferEthernet bool) (*wmi.Result, error) {
	adapterConfs := make([]adapter, 0)

	qParams := []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "IPEnabled", Value: true, Type: wmi.Equals}},
	}
	confs, err := m.cimv2.Gwmi("Win32_NetworkAdapterConfiguration", []string{}, qParams)
	if err != nil {
		return nil, errors.Wrap(err, "Get")
	}
	el, _ := confs.Elements()
	for _, v := range el {
		c, _ := v.GetProperty("IPConnectionMetric")
		id, _ := v.GetProperty("SettingID")
		description, _ := v.GetProperty("Description")

		a := adapter{
			id:          id.Value().(string),
			description: description.Value().(string),
			metric:      c.Value().(int32),
		}
		adapterConfs = append(adapterConfs, a)
		fmt.Println("a1>", a)
	}
	sort.Sort(metricSorter(adapterConfs))
	for _, ac := range adapterConfs {
		a, err := m.findNetworkAdapterByID(ac.id, preferEthernet)
		if err == nil {
			return a, nil
		}
	}
	// TODO: fallback to wifi

	return nil, nil
}

func (m *Manager) findNetworkAdapterByID(id string, preferEthernet bool) (*wmi.Result, error) {
	fmt.Println("findNetworkAdapterByID>", preferEthernet)

	qParams := []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "EnabledState", Value: StateEnabled, Type: wmi.Equals}},
		&wmi.AndQuery{wmi.QueryFields{Key: "DeviceID", Value: "Microsoft:" + id, Type: wmi.Equals}},
	}

	eep, err := m.con.GetOne(network.ExternalPort, []string{}, qParams)
	if !errors.Is(err, wmi.ErrNotFound) && err != nil {
		return nil, err
	}

	if preferEthernet {
		return eep, err
	}
	eep, err = m.con.GetOne(WifiPort, []string{}, qParams)
	return eep, err
}

func (m *Manager) CreateExternalNetworkSwitchIfNotExistsAndAssign(preferEthernet bool) error {
	// check if the switch exists
	_, err := m.GetSwitch(defaultSwitchName)
	if err == nil {
		return nil
	}
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
	swInstance.Set("ElementName", defaultSwitchName)
	swInstance.Set("Notes", []string{"vSwitch for mysterium node"})
	switchText, err := swInstance.GetText(1)
	if err != nil {
		return errors.Wrap(err, "GetText")
	}

	eep, err := m.FindDefaultNetworkAdapter(preferEthernet)
	if err != nil {
		return errors.Wrap(err, "FindDefaultNetworkAdapter")
	}

	// find external ethernet port and get its device eepPath
	eepPath, err := eep.Path()
	if err != nil {
		return errors.Wrap(err, "Path")
	}

	extPortData, err := m.getDefaultClassValue(PortAllocSetData, network.ETHConnResSubType)
	if err != nil {
		return errors.Wrap(err, "getDefaultClassValue")
	}
	extPortData.Set("ElementName", defaultSwitchName+" external port")
	extPortData.Set("HostResource", []string{eepPath})
	extPortDataStr, err := extPortData.GetText(1)
	if err != nil {
		return errors.Wrap(err, "GetText")
	}

	// SetInternalPort will create an internal port which will allow the OS to manage this switches network settings.
	mac, err := eep.GetProperty("PermanentAddress")
	if err != nil {
		return errors.Wrap(err, "GetProperty")
	}
	hostPath, err := m.getHostPath()
	if err != nil {
		return errors.Wrap(err, "getHostPath")
	}
	intPortData, err := m.getDefaultClassValue(PortAllocSetData, network.ETHConnResSubType)
	if err != nil {
		return errors.Wrap(err, "getDefaultClassValue")
	}
	intPortData.Set("HostResource", []string{hostPath})
	intPortData.Set("Address", mac.Value())
	intPortData.Set("ElementName", defaultSwitchName)
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
