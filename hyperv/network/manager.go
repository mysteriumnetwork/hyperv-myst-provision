package network

import (
	"fmt"
	"github.com/gabriel-samfira/go-wmi/utils"
	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/go-ole/go-ole"
	"github.com/pkg/errors"
)

type Manager struct {
	con       *wmi.WMI
	switchMgr *wmi.Result
	vsMgr     *wmi.Result
	imageMgr  *wmi.Result
}

// NewVMManager returns a new Manager type
func NewVMManager() (*Manager, error) {
	w, err := wmi.NewConnection(".", `root\virtualization\v2`)
	if err != nil {
		return nil, err
	}

	// Get virtual switch management service
	switchMgr, err := w.GetOne(VMSwitchManagementService, []string{}, []wmi.Query{})
	if err != nil {
		return nil, err
	}
	vsMgr, err := w.GetOne(VMSystemManagementService, []string{}, []wmi.Query{})
	if err != nil {
		return nil, err
	}
	imageMgr, err := w.GetOne(VMImageManagementService, []string{}, []wmi.Query{})
	if err != nil {
		return nil, err
	}

	sw := &Manager{
		con: w,

		switchMgr: switchMgr,
		vsMgr:     vsMgr,
		imageMgr:  imageMgr,
	}
	return sw, nil
}

// check if VM exists
func (m *Manager) GetVMByName(vmName string) (*wmi.Result, error) {
	qParams := []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "ElementName", Value: vmName, Type: wmi.Equals}},
	}
	swColl, err := m.con.Gwmi(ComputerSystem, []string{}, qParams)
	if err != nil {
		return nil, errors.Wrap(err, "Gwmi")
	}
	count, err := swColl.Count()
	if err != nil {
		return nil, errors.Wrap(err, "Count")
	}

	if count > 0 {
		el, _ := swColl.Elements()
		return el[0], nil
	}
	return nil, nil
}

func (m *Manager) CreateVM(vmName string) error {

	// create switch settings in xml representation
	data, err := m.con.Get(VMSystemSettingData)
	if err != nil {
		return errors.Wrap(err, "Get")
	}

	systemInstance, err := data.Get("SpawnInstance_")
	if err != nil {
		return errors.Wrap(err, "SpawnInstance_")
	}
	systemInstance.Set("ElementName", vmName)
	systemInstance.Set("Notes", []string{"VM for mysterium node"})
	systemInstance.Set("VirtualSystemSubType", "Microsoft:Hyper-V:SubType:2")
	systemText, err := systemInstance.GetText(1)
	if err != nil {
		return errors.Wrap(err, "GetText")
	}

	// resources
	data, err = m.con.Get(MsvmMemorySettingData)
	if err != nil {
		return errors.Wrap(err, "Get")
	}
	sysMemoryData, err := data.Get("SpawnInstance_")
	if err != nil {
		return errors.Wrap(err, "SpawnInstance_")
	}
	sysMemoryData.Set("VirtualQuantity", 512)
	sysMemoryData.Set("DynamicMemoryEnabled", false)
	sysMemoryDataStr, err := sysMemoryData.GetText(1)
	if err != nil {
		return errors.Wrap(err, "GetText")
	}

	//data, err = m.con.Get(MsvmVirtualHardDiskSettingData)
	//if err != nil {
	//	return errors.Wrap(err, "Get")
	//}
	//vhdData, err := data.Get("SpawnInstance_")
	//if err != nil {
	//	return errors.Wrap(err, "SpawnInstance_")
	//}
	//vhdData.Set("BlockSize", 0)
	//vhdDataStr, err := vhdData.GetText(1)
	//if err != nil {
	//	return errors.Wrap(err, "GetText")
	//}
	//fmt.Println(vhdDataStr)

	el, err := m.getDefaultClassValue(scsi)
	if err != nil {
		return err
	}
	newID, err := utils.UUID4()
	if err != nil {
		return errors.Wrap(err, "UUID4")
	}
	if el.Set("VirtualSystemIdentifiers", []string{fmt.Sprintf("{%s}", newID)}); err != nil {
		return errors.Wrap(err, "VirtualSystemIdentifiers")
	}
	scsiCtrlStr, err := el.GetText(1)
	if err != nil {
		return errors.Wrap(err, "GetText")
	}

	// create vm
	jobPath := ole.VARIANT{}
	resultingSystem := ole.VARIANT{}
	jobState, err := m.vsMgr.Get("DefineSystem", systemText, []string{sysMemoryDataStr}, nil, &resultingSystem, &jobPath)
	if err != nil {
		return errors.Wrap(err, "DefineSystem")
	}
	err = m.waitForJob(jobState, jobPath)
	if err != nil {
		return err
	}

	locationURI := resultingSystem.Value().(string)
	fmt.Println(locationURI)
	loc, err := wmi.NewLocation(locationURI)
	if err != nil {
		return errors.Wrap(err, "getting location")
	}
	result, err := loc.GetResult()
	if err != nil {
		return errors.Wrap(err, "getting result")
	}
	id, err := result.GetProperty("Name")
	if err != nil {
		return errors.Wrap(err, "fetching VM ID")
	}
	fmt.Println(id.Value())

	// add scsiCtrl
	jobPath2 := ole.VARIANT{}
	resultingSystem2 := ole.VARIANT{}
	jobState2, err := m.vsMgr.Get("AddResourceSettings", locationURI, []string{scsiCtrlStr}, &resultingSystem2, &jobPath)
	if err != nil {
		return errors.Wrap(err, "AddResourceSettings")
	}
	err = m.waitForJob(jobState2, jobPath2)
	if err != nil {
		return err
	}
	locationURI2 := resultingSystem2.ToArray().ToValueArray()
	fmt.Println(locationURI2[0].(string))

	// add drive
	diskRes, err := m.getDefaultClassValue(diskType)
	if err != nil {
		return errors.Wrap(err, "getDefaultClassValue")
	}
	diskRes.Set("Address", 0)
	diskRes.Set("AddressOnParent", 0)
	diskRes.Set("Parent", locationURI2[0].(string))
	diskResStr, err := diskRes.GetText(1)
	if err != nil {
		return errors.Wrap(err, "GetText")
	}
	//fmt.Println(diskResStr)

	jobPath3 := ole.VARIANT{}
	resultingSystem3 := ole.VARIANT{}
	jobState3, err := m.vsMgr.Get("AddResourceSettings", locationURI, []string{diskResStr}, &resultingSystem3, &jobPath3)
	if err != nil {
		return errors.Wrap(err, "AddResourceSettings")
	}
	err = m.waitForJob(jobState3, jobPath3)
	if err != nil {
		return err
	}
	//locationURI3 := resultingSystem3.ToArray().ToValueArray()
	//fmt.Println(locationURI3)

	//if err := storageRes.Set("Parent", drivePath); err != nil {
	//	return errors.Wrap(err, "Parent")
	//}
	//if err := storageRes.Set("HostResource", []string{path}); err != nil {
	//	return errors.Wrap(err, "HostResource")
	//}

	//r, err := result.Get("associators_", nil, ResourceAllocationSettingData)
	//fmt.Println("r>", r, err)
	//fmt.Println(r.Count())
	//r_, err := r.ItemAtIndex(0)
	//fmt.Println("r>", r_, err)
	//fmt.Println(resultingSystem.ToIDispatch().GetProperty("InstanceID"))
	//fmt.Println(resultingSystem.ToIDispatch().GetProperty("Name"))
	return err
}

func (m *Manager) StartVM(vmName string) error {
	qParams := []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "ElementName", Value: vmName, Type: wmi.Equals}},
	}
	swColl, err := m.con.Gwmi(ComputerSystem, []string{}, qParams)
	if err != nil {
		return errors.Wrap(err, "Gwmi")
	}
	count, err := swColl.Count()
	if err != nil {
		return errors.Wrap(err, "Count")
	}
	if count == 0 {
		return errors.New("VM not found")
	}

	el, err := swColl.Elements()
	if err != nil {
		return errors.Wrap(err, "Elements")
	}
	vm := el[0]
	fmt.Println(vm.Path())

	jobPath := ole.VARIANT{}
	jobState, err := vm.Get("RequestStateChange", 2, nil, &jobPath)
	if err != nil {
		return errors.Wrap(err, "RequestStateChange")
	}

	return m.waitForJob(jobState, jobPath)
}

func (m *Manager) CreateExternalNetworkSwitchIfNotExistsAndAssign() error {
	// check if the switch exists
	qParams := []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "ElementName", Value: switchName, Type: wmi.Equals}},
	}
	swColl, err := m.con.Gwmi(VMSwitchClass, []string{}, qParams)
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

	// find external ethernet port and get its device path
	qParams = []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "EnabledState", Value: 2, Type: wmi.Equals}},
	}
	eep, err := m.con.GetOne(ExternalPort, []string{}, qParams)
	if err != nil {
		return errors.Wrap(err, "GetOne")
	}
	path, err := eep.Path()
	if err != nil {
		return errors.Wrap(err, "Path")
	}

	// get xml prepresentation of external ethernet port
	data, err = m.con.Get(PortAllocSetData)
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
	jobState, err := m.switchMgr.Get("DefineSystem", switchText, []string{extPortDataStr}, nil, &resultingSystem, &jobPath)
	if err != nil {
		return errors.Wrap(err, "DefineSystem")
	}

	return m.waitForJob(jobState, jobPath)
}
