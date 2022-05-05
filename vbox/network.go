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
	IPs     string
}

func (a *Adapter) Set(id, name string, netType int, metr int, ips string) {
	a.ID = id
	a.Name = name
	a.NetType = netType
	a.Metric = metr
	a.IPs = ips
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

func ArrayToStr(a []interface{}) string {
	v := ""
	for _, k := range a {
		v += k.(string) + ","
	}
	return v
}

func (m *Manager) GetAdapterMetrics(adapter *wmi.Result) (int, string, error) {
	assoc, err := adapter.Get("associators_", nil, "Win32_NetworkAdapterConfiguration")
	if err != nil {
		return 0, "", errors.Wrap(err, "Get")
	}
	cfg, err := assoc.ItemAtIndex(0)
	if err != nil {
		return 0, "", errors.Wrap(err, "ItemAtIndex")
	}

	met, _ := cfg.GetProperty("IPConnectionMetric")
	v, _ := met.Value().(int32)

	adr, _ := cfg.GetProperty("IPAddress")
	aa := ""
	arr := adr.ToArray()
	if arr != nil {
		aa = ArrayToStr(arr.ToValueArray())
	}
	return int(v), aa, nil
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

///////////////
type NetworkEv int

const (
	NetworkChangeIP    = NetworkEv(1)
	NetworkChangeMedia = NetworkEv(2)
)

// If old chosen adapter goes offline, find a new one with minimal metric
func (m *Manager) MonitorNetwork(networkChangeEv chan NetworkEv) error {
	defer util.PanicHandler("monitor_")
	log.Info().Msg("Monitor network")

	list := func() error {
		// Find adapter by foll. criteria: Physical, PNPDeviceID ! like ^ROOT/%, with minimal metric, type={0,9}

		qParams := []wmi.Query{
			&wmi.OrQuery{wmi.QueryFields{Key: "PhysicalAdapter", Value: true, Type: wmi.Equals}},
			&wmi.AndQuery{wmi.QueryFields{Key: "MACAddress", Value: "NOT NULL", Type: wmi.Is}},
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
				log.Info().Msgf("Monitor network >", id, name, pnpDeviceID_.Value())
				//continue
			}
			if strings.HasPrefix(pnpDeviceID, `ROOT\`) || strings.HasPrefix(pnpDeviceID, `BTH\`) {
				continue
			}

			metric, ips, _ := m.GetAdapterMetrics(adp)
			adapterType, err := m.GetAdapterType(name_.Value().(string))
			if err != nil {
				log.Info().Msgf("!list > %v", err)
				continue
			}
			//log.Info().Msgf("Monitor network !list: %v %v", id, metric, name_.Value(), adapterType, pnpDeviceID)

			if metric != 0 {
				// first assignment
				if minAdapter.ID == "" {
					minAdapter.Set(id, name, adapterType, metric, ips)
				}

				if metric < minAdapter.Metric {
					minAdapter.Set(id, name, adapterType, metric, ips)
				}
			}
		}
		//log.Info().Msgf("Monitor network !list > minAdapter : %v", minAdapter)

		// If old adapter lost its metric -- then change adapter
		// IP has changed

		//log.Info().Msgf("Monitor network !list >", m.MinAdapter.ID, minAdapter.ID, m.MinAdapter.IPs)
		if m.MinAdapter.ID == minAdapter.ID && m.MinAdapter.IPs != minAdapter.IPs {
			log.Info().Msgf("Monitor network > ! IP change detected")
			m.MinAdapter = minAdapter

			networkChangeEv <- NetworkChangeIP
		}

		if m.MinAdapter.ID != minAdapter.ID {
			old, _ := m.GetAdapter(m.MinAdapter.ID)
			if err != nil {
				log.Info().Msgf("!GetAdapter > %v", err)
			}
			//log.Info().Msgf("Monitor network !list > old : %v", old)

			oldMetric := 0
			if old != nil {
				oldMetric, _, _ = m.GetAdapterMetrics(old)
				log.Info().Msgf("Monitor network !list > minAdapter > oldMetric : %v", oldMetric)
			}

			if oldMetric == 0 && minAdapter.Metric > 0 { // exclude single adapter switch off case
				m.MinAdapter = minAdapter

				//log.Info().Msgf("New minAdapter > %v", minAdapter)
				networkChangeEv <- NetworkChangeMedia
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
