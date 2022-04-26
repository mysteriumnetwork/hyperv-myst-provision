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

func (m *Manager) GetAdapters() ([]Adapter, error) {
	qParams := []wmi.Query{
		&wmi.OrQuery{wmi.QueryFields{Key: "PhysicalAdapter", Value: true, Type: wmi.Equals}},
	}
	adapters, err := m.cimV2.Gwmi("Win32_NetworkAdapter", []string{}, qParams)
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
		})
	}
	return list, nil
}

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
}

func (m *Manager) GetAdapterType(adapterName string) (int, error) {
	qParams := []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "InstanceName", Value: adapterName, Type: wmi.Equals}},
	}

	mediumType, err := m.rootWmi.GetOne("MSNdis_PhysicalMediumType", []string{}, qParams)
	if err != nil {
		return 0, errors.Wrap(err, "GetOne")
	}

	id_, _ := mediumType.GetProperty("NdisPhysicalMediumType")
	id := id_.Value().(int32)

	return int(id), nil
}

func (m *Manager) GetAdapter(ID string) (*wmi.Result, error) {
	qParams := []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "GUID", Value: ID, Type: wmi.Equals}},
	}
	res, err := m.cimV2.GetOne("Win32_NetworkAdapter", []string{}, qParams)
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
			&wmi.OrQuery{wmi.QueryFields{Key: "PhysicalAdapter", Value: true, Type: wmi.Equals}},
		}
		adapters, err := m.cimV2.Gwmi("Win32_NetworkAdapter", []string{}, qParams)
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
			pnpDeviceID, ok := pnpDeviceID_.Value().(string)
			if !ok {
				log.Info().Msg("")
				log.Info().Msgf("Monitor network >", id, name, pnpDeviceID)
				//continue
			}
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

		return nil
	}
	list()

	for {
		m.notifier.WaitForIPChange()
		list()
	}
}
