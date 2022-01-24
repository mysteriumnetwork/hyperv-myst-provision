package hyperv_wmi

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/gabriel-samfira/go-wmi/virt/network"
	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/go-ole/go-ole"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
	defaultSwitchName = "Myst Bridge Switch"
	//defaultSwitchName = "Default Switch"
)

func (m *Manager) GetSwitch(switchName string) (*wmi.Result, error) {
	qParams := []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "ElementName", Value: switchName, Type: wmi.Equals}},
	}
	return m.con.GetOne(VMSwitchClass, []string{}, qParams)
}

// Find a network adapter with minimum metric and use it for virtual network switch
// prefer active
// returns: port, adapter, id, error
func (m *Manager) FindDefaultNetworkAdapter(preferEthernet bool) (*wmi.Result, *wmi.Result, string, error) {

	// 9    - wifi
	// 0    - ethernet
	// abs. - other type
	nameToType := make(map[string]int32)
	mediaTypes, err := m.wmi.Gwmi("MSNdis_PhysicalMediumType", []string{}, nil)
	if err != nil {
		return nil, nil, "", errors.Wrap(err, "Get")
	}
	el, _ := mediaTypes.Elements()
	for _, v := range el {
		mt, _ := v.GetProperty("NdisPhysicalMediumType")
		in, _ := v.GetProperty("InstanceName")
		fmt.Println(mt.Value(), in.Value())
		nameToType[in.Value().(string)] = mt.Value().(int32)
	}

	qParams := []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "PhysicalAdapter", Value: true, Type: wmi.Equals}},
	}
	adapters, err := m.cimv2.Gwmi("Win32_NetworkAdapter", []string{}, qParams)
	if err != nil {
		return nil, nil, "", errors.Wrap(err, "Get")
	}
	el, _ = adapters.Elements()
	for _, adp := range el {
		id_, _ := adp.GetProperty("GUID")
		name_, _ := adp.GetProperty("Name")
		servName_, _ := adp.GetProperty("ServiceName")

		id, name, servName := id_.Value().(string), name_.Value().(string), servName_.Value().(string)
		if strings.HasPrefix(strings.ToLower(servName), "vm") {
			continue
		}

		connType, ok := nameToType[name]
		if !ok || connType == 10 {
			// skip bluetooth
			continue
		}
		log.Print("dbg >", id, name, connType, ok, servName)
		if (preferEthernet && connType == 0) || (!preferEthernet && connType != 0) {
			ap, err := m.findNetworkAdapterPortByID(id, preferEthernet)
			log.Print("ap>", ap, id, err)
			if err == nil {
				return ap, adp, id, nil
			}
		}
	}

	// TODO: fallback to wifi
	return nil, nil, "", nil
}

func (m *Manager) findNetworkAdapterPortByID(id string, preferEthernet bool) (*wmi.Result, error) {
	log.Print("findNetworkAdapterPortByID>", id, preferEthernet)

	qParams := []wmi.Query{
		//&wmi.AndQuery{wmi.QueryFields{Key: "EnabledState", Value: StateEnabled, Type: wmi.Equals}},
		&wmi.AndQuery{wmi.QueryFields{Key: "DeviceID", Value: "Microsoft:" + id, Type: wmi.Equals}},
	}

	eep, err := m.con.GetOne(network.ExternalPort, []string{}, qParams)
	if !errors.Is(err, wmi.ErrNotFound) && err != nil {
		return nil, err
	}

	// skip if not ethernet and try wifi
	if preferEthernet && !errors.Is(err, wmi.ErrNotFound) {
		return eep, err
	}
	eep, err = m.con.GetOne(WifiPort, []string{}, qParams)
	return eep, err
}

func (m *Manager) RemoveSwitch() error {
	// check if the switch exists
	sw, err := m.GetSwitch(defaultSwitchName)
	if errors.Is(err, wmi.ErrNotFound) {
		return nil
	}
	if err != nil && !errors.Is(err, wmi.ErrNotFound) {
		return errors.Wrap(err, "GetOne")
	}
	path, err := sw.Path()
	if err != nil {
		return err
	}

	jobPath := ole.VARIANT{}
	jobState, err := m.switchMgr.Get("DestroySystem", path, &jobPath)
	if err != nil {
		return errors.Wrap(err, "DestroySystem")
	}
	return m.waitForJob(jobState, jobPath)
}

func (m *Manager) AdapterHasIPAddress(adapter *wmi.Result) (bool, error) {
	assoc, err := adapter.Get("associators_", nil, "Win32_NetworkAdapterConfiguration")
	if err != nil {
		return false, errors.Wrap(err, "Get")
	}
	cfg, err := assoc.ItemAtIndex(0)
	if err != nil {
		return false, errors.Wrap(err, "ItemAtIndex")
	}

	ips_, err := cfg.GetProperty("IPAddress")
	if err != nil {
		return false, errors.Wrap(err, "GetProperty")
	}
	arr := ips_.ToArray()
	if arr != nil {
		vals := arr.ToValueArray()
		log.Print("ip ", vals)

		for _, ip := range vals {
			if net.ParseIP(ip.(string)) != nil {
				log.Print("ip> ", ip)
				return true, nil
			}
		}
	}
	return false, nil
}

func (m *Manager) WaitForExternalNetworkIsReady(extAdapter *wmi.Result) error {
	log.Info().Msg("Wait for external network is ready")
	deadline := time.Now().Add(1 * time.Minute)
	for {
		ok, err := m.AdapterHasIPAddress(extAdapter)
		if err != nil {
			return errors.Wrap(err, "AdapterHasIPAddress")
		}
		if ok {
			break
		}
		if time.Now().After(deadline) {
			return errors.New("Network wait timeout")
		}

		m.n.WaitForIPChange()
	}
	return nil
}

// Modify switch (set external connection) or create a new one
func (m *Manager) ModifySwitchSettings(preferEthernet bool) error {
	log.Info().Msgf("ModifySwitchSettings %v", preferEthernet)

	sw, err := m.GetSwitch(defaultSwitchName)
	if errors.Is(err, wmi.ErrNotFound) {
		err := m.CreateExternalNetworkSwitchIfNotExistsAndAssign(preferEthernet)
		return err
	}
	if err != nil && !errors.Is(err, wmi.ErrNotFound) {
		return errors.Wrap(err, "GetOne")
	}
	swPath, err := sw.Path()
	if err != nil {
		return err
	}
	fmt.Println(swPath)

	// find external ethernet port and get its device eepPath
	eep, extAdapter, devID, err := m.FindDefaultNetworkAdapter(preferEthernet)
	if err != nil {
		return errors.Wrap(err, "FindDefaultNetworkAdapter")
	}
	eepPath, err := eep.Path()
	if err != nil {
		return errors.Wrap(err, "Path")
	}

	_ = extAdapter
	//err = m.WaitForExternalNetworkIsReady(extAdapter)
	//if err != nil {
	//	return err
	//}

	switchSettingsResult, err := sw.Get("associators_", nil, VMSwitchSettings)
	if err != nil {
		return err
	}
	switchSettings, err := switchSettingsResult.ItemAtIndex(0)

	// iterate through switch ports
	portsData, err := switchSettings.Get("associators_", nil, PortAllocSetData)
	if err != nil {
		return errors.Wrap(err, "associators_")
	}
	count, _ := portsData.Count()
	var ports []string
	for i := 0; i < count; i++ {
		var err error
		port, err := portsData.ItemAtIndex(i)
		if err != nil {
			return errors.Wrap(err, "ItemAtIndex")
		}

		hr, err := port.GetProperty("HostResource")
		if err != nil {
			return errors.Wrap(err, "GetProperty")
		}
		vals := hr.ToArray().ToValueArray()[0].(string)
		for _, pair := range strings.Split(vals, ",") {
			//log.Print(">", pair)
			log.Info().Msgf("> %v", pair)

			kv := strings.Split(pair, "=")
			if len(kv) < 2 {
				continue
			}
			if kv[0] == "DeviceID" {
				hrDeviceID := strings.TrimRight(strings.TrimLeft(kv[1], `"`), `"`)
				sameDevice := hrDeviceID == "Microsoft:"+devID
				log.Info().Msgf("DeviceID> %v %v skipping %v", hrDeviceID, devID, sameDevice)
				if sameDevice {
					// don't change switch settings
					return nil
				}
			}
		}

		p, err := port.Path()
		_ = err
		ports = append(ports, p)
	}
	ssPath, err := switchSettings.Path()
	if err != nil {
		return errors.Wrap(err, "Path")
	}
	fmt.Println("ssPath", ssPath)

	// remove ports
	jobPath := ole.VARIANT{}
	jobState, err := m.switchMgr.Get("RemoveResourceSettings", ports, &jobPath)
	if err != nil {
		return errors.Wrap(err, "RemoveResourceSettings")
	}
	m.waitForJob(jobState, jobPath)

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

	// modify switch
	jobPath = ole.VARIANT{}
	jobState, err = m.switchMgr.Get("AddResourceSettings", ssPath, []string{extPortDataStr, intPortDataStr}, nil, &jobPath)
	if err != nil {
		return errors.Wrap(err, "AddResourceSettings")
	}
	return m.waitForJob(jobState, jobPath)
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

	eep, _, _, err := m.FindDefaultNetworkAdapter(preferEthernet)
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
