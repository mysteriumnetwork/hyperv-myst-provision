package vbox

import (
	"fmt"
	"strings"

	"github.com/mysteriumnetwork/hyperv-node/service2/util"

	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

func (m *Manager) GetSwitch(switchName string) (*wmi.Result, error) {
	return nil, nil
}

type Adapter struct {
	ID      string
	Name    string
	NetType int
	Metric  int
}

func (a *Adapter) Set(id, name string, netType int, metr int) {
	a.ID = id
	a.Name = name
	a.NetType = netType
	a.Metric = metr
}

func (m *Manager) SelectAdapter() ([]Adapter, error) {
	qParams := []wmi.Query{
		//&wmi.OrQuery{wmi.QueryFields{Key: "NetConnectionID", Value: "Ethernet", Type: wmi.Equals}},
		//&wmi.OrQuery{wmi.QueryFields{Key: "NetConnectionID", Value: "Wi-Fi", Type: wmi.Equals}},

		&wmi.OrQuery{wmi.QueryFields{Key: "PhysicalAdapter", Value: true, Type: wmi.Equals}},
	}
	adapters, err := m.cimv2.Gwmi("Win32_NetworkAdapter", []string{}, qParams)
	if err != nil {
		return nil, errors.Wrap(err, "Get")
	}
	el, err := adapters.Elements()
	if err != nil {
		return nil, errors.Wrap(err, "Elements")
	}

	var list []Adapter
	for _, adp := range el {
		id_, _ := adp.GetProperty("GUID")
		name_, _ := adp.GetProperty("Name")
		netConnectionID_, _ := adp.GetProperty("NetConnectionID")

		id, name, netConnectionID := id_.Value().(string), name_.Value().(string), netConnectionID_.Value().(string)
		fmt.Println(id, name, netConnectionID)
		list = append(list, Adapter{
			ID:   id,
			Name: name,
			//NetType: netConnectionID,
		})
	}
	return list, nil
}

// type NetworkAdapterInfo struct {
// 	externalPort *wmi.Result
// 	adapter      *wmi.Result
// 	adapterID    string
// 	adapterName  string
// }

// Find a network adapter with minimum metric and use it for virtual network switch
// prefer active
// returns: NetworkAdapterInfo, error
// func (m *Manager) FindDefaultNetworkAdapter(preferEthernet bool, adapterID string, nai *NetworkAdapterInfo) error {
// 	log.Info().Msgf("FindDefaultNetworkAdapter> %v %v", preferEthernet, adapterID)

// 	if adapterID != "" {
// 		port, err := m.findNetworkAdapterPortByID(adapterID, false)
// 		if err != nil {
// 			return errors.Wrap(err, "findNetworkAdapterPortByID")
// 		}
// 		nai.externalPort = port
// 		nai.adapterID = adapterID
// 		return nil
// 	}

// 	qParams := []wmi.Query{
// 		&wmi.OrQuery{wmi.QueryFields{Key: "NetConnectionID", Value: "Ethernet", Type: wmi.Equals}},
// 		&wmi.OrQuery{wmi.QueryFields{Key: "NetConnectionID", Value: "Wi-Fi", Type: wmi.Equals}},
// 	}
// 	adapters, err := m.cimv2.Gwmi("Win32_NetworkAdapter", []string{}, qParams)
// 	if err != nil {
// 		return errors.Wrap(err, "Get")
// 	}
// 	el, _ := adapters.Elements()
// 	for _, adp := range el {
// 		id_, _ := adp.GetProperty("GUID")
// 		name_, _ := adp.GetProperty("Name")
// 		netConnectionID_, _ := adp.GetProperty("NetConnectionID")
// 		description_, _ := adp.GetProperty("Description")

// 		id, name, netConnectionID := id_.Value().(string), name_.Value().(string), netConnectionID_.Value().(string)
// 		log.Debug().Msgf("FindDefaultNetworkAdapter> %v %v %v", id, name, netConnectionID)

// 		if (preferEthernet && netConnectionID == "Ethernet") || (!preferEthernet && netConnectionID != "Ethernet") {
// 			ap, err := m.findNetworkAdapterPortByID(id, preferEthernet)
// 			if err == nil {
// 				nai.externalPort = ap
// 				nai.adapter = adp
// 				nai.adapterID = id
// 				nai.adapterName = description_.Value().(string)
// 				return nil
// 			}
// 		}
// 	}

// 	return nil
// }

// func (m *Manager) findNetworkAdapterPortByID(id string, preferEthernet bool) (*wmi.Result, error) {
// 	log.Info().Msgf("findNetworkAdapterPortByID> %v %v", id, preferEthernet)

// 	qParams := []wmi.Query{
// 		//&wmi.AndQuery{wmi.QueryFields{Key: "EnabledState", Value: StateEnabled, Type: wmi.Equals}},
// 		&wmi.AndQuery{wmi.QueryFields{Key: "DeviceID", Value: "Microsoft:" + id, Type: wmi.Equals}},
// 	}
// 	eep, err := m.con.GetOne(network.ExternalPort, []string{}, qParams)

// 	if !errors.Is(err, wmi.ErrNotFound) && err != nil {
// 		return nil, err
// 	}
// 	if err == nil {
// 		return eep, err
// 	}

// 	//// skip if not ethernet and try wifi
// 	//if preferEthernet && !errors.Is(err, wmi.ErrNotFound) {
// 	//	return eep, err
// 	//}

// 	eep, err = m.con.GetOne(WifiPort, []string{}, qParams)
// 	return eep, err
// }

// func (m *Manager) RemoveSwitch() error {
// 	// check if the switch exists
// 	sw, err := m.GetSwitch(defaultSwitchName)
// 	if errors.Is(err, wmi.ErrNotFound) {
// 		return nil
// 	}
// 	if err != nil && !errors.Is(err, wmi.ErrNotFound) {
// 		return errors.Wrap(err, "GetOne")
// 	}
// 	path, err := sw.Path()
// 	if err != nil {
// 		return err
// 	}

// 	jobPath := ole.VARIANT{}
// 	jobState, err := m.switchMgr.Get("DestroySystem", path, &jobPath)
// 	if err != nil {
// 		return errors.Wrap(err, "DestroySystem")
// 	}
// 	return m.waitForJob(jobState, jobPath)
// }

func (m *Manager) AdapterHasIPAddress(adapter *wmi.Result) (int, error) {
	// fmt.Println("AdapterHasIPAddress >")
	assoc, err := adapter.Get("associators_", nil, "Win32_NetworkAdapterConfiguration")
	if err != nil {
		return 0, errors.Wrap(err, "Get")
	}
	cfg, err := assoc.ItemAtIndex(0)
	if err != nil {
		return 0, errors.Wrap(err, "ItemAtIndex")
	}

	met, _ := cfg.GetProperty("IPConnectionMetric")
	if met.Value() == nil {
		// fmt.Println("AdapterHasIPAddress > nil")
		return 0, nil
	} else {
		v := met.Value().(int32)
		// fmt.Println("AdapterHasIPAddress >", v)
		return int(v), nil
	}

	// ips_, err := cfg.GetProperty("IPAddress")
	// if err != nil {
	// 	return false, errors.Wrap(err, "GetProperty")
	// }
	// arr := ips_.ToArray()
	// if arr != nil {
	// 	vals := arr.ToValueArray()
	// 	log.Print("ip ", vals)

	// 	for _, ip := range vals {
	// 		if net.ParseIP(ip.(string)) != nil {
	// 			log.Print("ip> ", ip)
	// 			fmt.Println("AdapterHasIPAddress >", ip)

	// 			return true, nil
	// 		}
	// 	}
	// }
	//return false, nil
}

// func (m *Manager) WaitForExternalNetworkIsReady_(extAdapter *wmi.Result) error {
// 	log.Info().Msg("Wait for external network is ready")
// 	deadline := time.Now().Add(1 * time.Minute)
// 	for {
// 		ok, err := m.AdapterHasIPAddress(extAdapter)
// 		if err != nil {
// 			return errors.Wrap(err, "AdapterHasIPAddress")
// 		}
// 		if ok {
// 			break
// 		}
// 		if time.Now().After(deadline) {
// 			return errors.New("Network wait timeout")
// 		}

// 		m.notifier.WaitForIPChange()
// 	}
// 	return nil
// }

func (m *Manager) GetAdapterType(adapterName string) (int, error) {
	//log.Info().Msgf("GetAdapterType 1> %v", adapterName)

	qParams := []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "InstanceName", Value: adapterName, Type: wmi.Equals}},
	}
	mediumType, err := m.rootWmi.GetOne("MSNdis_PhysicalMediumType", []string{}, qParams)
	if err != nil {
		return 0, errors.Wrap(err, "GetOne")
	}

	id_, _ := mediumType.GetProperty("NdisPhysicalMediumType")
	id := id_.Value().(int32)
	//name_, _ := mediumType.GetProperty("InstanceName")
	//log.Info().Msgf("GetAdapterType 4> %v %v", id_.Value(), name_.Value())

	return int(id), nil
}

func (m *Manager) GetAdapter(ID string) (*wmi.Result, error) {
	qParams := []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "GUID", Value: ID, Type: wmi.Equals}},
	}
	res, err := m.cimv2.GetOne("Win32_NetworkAdapter", []string{}, qParams)
	if err != nil {
		return nil, errors.Wrap(err, "GetOne")
	}

	return res, nil
}

// If old chosen adapter goes offline, find a new one with minimal metric
func (m *Manager) MonitorNetwork(networkChangeEv chan bool) error {
	defer util.PanicHandler("monitor_")
	log.Info().Msg("Monitor network")

	list := func() error {
		// Find adapter by foll. criteria: Physical, PNPDeviceID ! like ^ROOT/%, with minimal metric, type={0,9}

		qParams := []wmi.Query{
			//&wmi.OrQuery{wmi.QueryFields{Key: "NetConnectionID", Value: "Ethernet", Type: wmi.Equals}},
			//&wmi.OrQuery{wmi.QueryFields{Key: "NetConnectionID", Value: "Wi-Fi", Type: wmi.Equals}},
			&wmi.OrQuery{wmi.QueryFields{Key: "PhysicalAdapter", Value: true, Type: wmi.Equals}},
		}
		adapters, err := m.cimv2.Gwmi("Win32_NetworkAdapter", []string{}, qParams)
		if err != nil {
			return errors.Wrap(err, "Get")
		}
		el, err := adapters.Elements()
		if err != nil {
			return errors.Wrap(err, "Elements")
		}

		var minAdapter Adapter
		for _, adp := range el {
			id_, _ := adp.GetProperty("GUID")
			name_, _ := adp.GetProperty("Name")
			pnpDeviceID_, _ := adp.GetProperty("PNPDeviceID") // exclude virtual adapters, bluetooth

			id := id_.Value().(string)
			name := name_.Value().(string)
			pnpDeviceID := pnpDeviceID_.Value().(string)
			if strings.HasPrefix(pnpDeviceID, `ROOT\`) || strings.HasPrefix(pnpDeviceID, `BTH\`) {
				continue
			}

			metr, _ := m.AdapterHasIPAddress(adp)
			adapterType, err := m.GetAdapterType(name_.Value().(string))
			if err != nil {
				log.Info().Msgf("!list > %v", err)
				continue
			}
			//log.Info().Msgf("Monitor network !list: %v %v", id, metr, name_.Value(), adapterType, pnpDeviceID)

			if minAdapter.ID == "" && metr != 0 {
				minAdapter.Set(id, name, adapterType, metr)
			}
			if metr < minAdapter.Metric && metr != 0 {
				//findNewAdapter = true
				minAdapter.Set(id, name, adapterType, metr)
			}
		}

		log.Info().Msgf("Monitor network !list > minAdapter : %v", minAdapter)
		// If old adapter lost its metric -- then change adapter

		if m.MinAdapter.ID != minAdapter.ID {
			old, _ := m.GetAdapter(m.MinAdapter.ID)
			if err != nil {
				log.Info().Msgf("!GetAdapter > %v", err)
			}
			log.Info().Msgf("Monitor network !list > old : %v", old)

			oldMetric := 0
			if old != nil {
				oldMetric, _ = m.AdapterHasIPAddress(old)
				log.Info().Msgf("Monitor network !list > minAdapter > oldMetric : %v", oldMetric)
			}

			if oldMetric == 0 && minAdapter.Metric > 0 { // exclude single adapter switch off case
				m.MinAdapter = minAdapter

				//log.Print("New minAdapter >>>", minAdapter)
				//log.Info().Msgf("New minAdapter >>> %v", minAdapter)
				networkChangeEv <- true
			}
		}

		//findNewAdapter := false
		//for _, adp := range el {
		//	id_, _ := adp.GetProperty("GUID")
		//	id := id_.Value().(string)
		//
		//	metr, _ := m.AdapterHasIPAddress(adp)
		//	if id == m.MinAdapter.ID && metr == 0 {
		//		findNewAdapter = true
		//	}
		//}
		//if m.MinAdapter.ID == "" {
		//	findNewAdapter = true
		//}
		//log.Info().Msgf("Monitor network !list: %v", findNewAdapter)
		//
		//if findNewAdapter {
		//	var minAdapter Adapter
		//	for _, adp := range el {
		//		log.Info().Msgf("!list for >>>>")
		//
		//		id_, _ := adp.GetProperty("GUID")
		//		name_, _ := adp.GetProperty("Name")
		//		netConnectionID_, _ := adp.GetProperty("NetConnectionID")
		//
		//		id, name, netConnectionID := id_.Value().(string), name_.Value().(string), netConnectionID_.Value().(string)
		//
		//		log.Info().Msgf("!list > GetAdapterType: %v", name)
		//		adapterType, err := m.GetAdapterType(name)
		//		if err != nil {
		//			log.Info().Msgf("!list > %v", err)
		//			continue
		//		}
		//		log.Info().Msgf("!list > adapterType %v", adapterType)
		//
		//		metr, _ := m.AdapterHasIPAddress(adp)
		//
		//		if metr > 0 && (adapterType == 0 || adapterType == 9) { // wifi or ethernet
		//			if minAdapter.Metric == 0 {
		//				minAdapter.Set(id, name, netConnectionID, metr)
		//			} else if minAdapter.Metric > metr {
		//				minAdapter.Set(id, name, netConnectionID, metr)
		//			}
		//		}
		//	}
		//
		//	if minAdapter.ID != "" && minAdapter.ID != m.MinAdapter.ID {
		//		m.MinAdapter = minAdapter
		//		log.Print("New minAdapter >>>", minAdapter)
		//		log.Info().Msgf("New minAdapter >>> %v", minAdapter)
		//
		//		networkChangeEv <- true
		//	}
		//}

		return nil
	}
	//log.Info().Msg("!list >")
	list()
	//log.Info().Msg("!list >")

	// deadline := time.Now().Add(1 * time.Minute)
	for {
		// ok, err := m.AdapterHasIPAddress(extAdapter)
		// if err != nil {
		// 	return errors.Wrap(err, "AdapterHasIPAddress")
		// }
		// if ok {
		// 	break
		// }
		// if time.Now().After(deadline) {
		// 	return errors.New("Network wait timeout")
		// }

		m.notifier.WaitForIPChange()
		//log.Info().Msg("!list >")
		list()
		//log.Info().Msg("!list >")
	}

	//return nil
}
