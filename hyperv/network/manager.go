package network

import (
	"fmt"

	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/go-ole/go-ole"
	"github.com/pkg/errors"
)

type Manager struct {
	con *wmi.WMI
	svc *wmi.Result
}

// NewVMSwitchManager returns a new Manager type
func NewVMSwitchManager() (*Manager, error) {
	w, err := wmi.NewConnection(".", `root\virtualization\v2`)
	if err != nil {
		return nil, err
	}

	// Get virtual switch management service
	svc, err := w.GetOne(VMSwitchManagementService, []string{}, []wmi.Query{})
	if err != nil {
		return nil, err
	}

	sw := &Manager{
		con: w,
		svc: svc,
	}
	return sw, nil
}

func (m *Manager) CreateExternalNetworkSwitchIfNotExistsAndAssign() error {
	w, err := wmi.NewConnection(".", `root\virtualization\v2`)
	if err != nil {
		return err
	}

	data, err := w.Get("Msvm_VirtualEthernetSwitchSettingData")
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
	fmt.Println("vesData >", switchText)

	qParams := []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "EnabledState", Value: 2, Type: wmi.Equals}},
	}
	eep, err := w.GetOne("Msvm_ExternalEthernetPort", []string{}, qParams)
	if err != nil {
		return errors.Wrap(err, "GetOne")
	}
	path, err := eep.Path()
	if err != nil {
		return errors.Wrap(err, "Path")
	}

	data, err = w.Get("Msvm_EthernetPortAllocationSettingData")
	if err != nil {
		return errors.Wrap(err, "Get")
	}
	extPortData, err := data.Get("SpawnInstance_")
	if err != nil {
		return errors.Wrap(err, "SpawnInstance_")
	}
	extPortData.Set("ElementName", switchName+"__extPort")
	extPortData.Set("HostResource", []string{path})
	extPortDataStr, err := extPortData.GetText(1)
	if err != nil {
		return errors.Wrap(err, "GetText")
	}
	fmt.Println("extPortDataStr", extPortDataStr)

	jobPath := ole.VARIANT{}
	resultingSystem := ole.VARIANT{}
	jobState, err := m.svc.Get("DefineSystem", switchText, []string{extPortDataStr}, nil, &resultingSystem, &jobPath)
	fmt.Println("result", jobState, jobPath.Value(), resultingSystem.Value(), err)
	if err != nil {
		return errors.Wrap(err, "DefineSystem")
	}
	if jobState.Value().(int32) == wmi.JobStatusStarted {
		fmt.Println("started")
		err := wmi.WaitForJob(jobPath.Value().(string))
		fmt.Println("started", err)
		if err != nil {
			return errors.Wrap(err, "WaitForJob")
		}
	}

	return nil
}
