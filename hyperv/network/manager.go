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

	// check if the switch exists
	qParams := []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "ElementName", Value: switchName, Type: wmi.Equals}},
	}
	swColl, err := w.Gwmi(VMSwitchClass, []string{}, qParams)
	if err != nil {
		return errors.Wrap(err, "Gwmi")
	}
	count, err := swColl.Count()
	if err != nil {
		return errors.Wrap(err, "Count")
	}
	if count > 0 {
		return nil
	}	

	// create switch settings in xml representation
	data, err := w.Get(VMSwitchSettings)
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

	// find external ethernet port and get its device path
	qParams = []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "EnabledState", Value: 2, Type: wmi.Equals}},
	}
	eep, err := w.GetOne(ExternalPort, []string{}, qParams)
	if err != nil {
		return errors.Wrap(err, "GetOne")
	}
	path, err := eep.Path()
	if err != nil {
		return errors.Wrap(err, "Path")
	}

	// get xml prepresentation of external ethernet port
	data, err = w.Get(PortAllocSetData)
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
	
	// create switch
	jobPath := ole.VARIANT{}
	resultingSystem := ole.VARIANT{}
	jobState, err := m.svc.Get("DefineSystem", switchText, []string{extPortDataStr}, nil, &resultingSystem, &jobPath)
	if err != nil {
		return errors.Wrap(err, "DefineSystem")
	}

	// wait for completion
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
