package vbox

import (
	"fmt"
	"net"
	"time"

	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
// defaultSwitchName = "Myst Bridge Switch"
// defaultSwitchName = "Default Switch"
)

func (m *Manager) GetSwitch(switchName string) (*wmi.Result, error) {
	return nil, nil
}

type Adapter struct {
	ID      string
	Name    string
	NetType string
}

func (m *Manager) SelectAdapter() ([]Adapter, error) {
	qParams := []wmi.Query{
		&wmi.OrQuery{wmi.QueryFields{Key: "NetConnectionID", Value: "Ethernet", Type: wmi.Equals}},
		&wmi.OrQuery{wmi.QueryFields{Key: "NetConnectionID", Value: "Wi-Fi", Type: wmi.Equals}},
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
			ID:      id,
			Name:    name,
			NetType: netConnectionID,
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

		m.notifier.WaitForIPChange()
	}
	return nil
}
