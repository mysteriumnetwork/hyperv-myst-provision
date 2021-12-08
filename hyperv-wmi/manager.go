package hyperv_wmi

import (
	"fmt"
	"github.com/gabriel-samfira/go-wmi/utils"
	"github.com/gabriel-samfira/go-wmi/virt/vm"
	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/go-ole/go-ole"
	"github.com/pkg/errors"
	"time"
)

const (
	switchName = "Myst Bridge Switch"
)

type Manager struct {
	vmName           string
	con              *wmi.WMI
	switchMgr        *wmi.Result
	vsMgr            *wmi.Result
	imageMgr         *wmi.Result
	guestFileService *wmi.Result // file copy service

	// guest KV map
	Kvp map[string]string
}

// NewVMManager returns a new Manager type
func NewVMManager(vmName string) (*Manager, error) {
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
		vmName: vmName,
		con:    w,

		switchMgr: switchMgr,
		vsMgr:     vsMgr,
		imageMgr:  imageMgr,
		Kvp:       nil,
	}
	return sw, nil
}

func (m *Manager) GetVM() (*wmi.Result, error) {
	qParams := []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "ElementName", Value: m.vmName, Type: wmi.Equals}},
	}
	return m.con.GetOne(ComputerSystemClass, []string{}, qParams)
}

func (m *Manager) CreateVM(vhdFilePath string) error {

	// create switch settings in xml representation
	data, err := m.con.Get(VMSystemSettingData)
	if err != nil {
		return errors.Wrap(err, "Get")
	}
	systemInstance, err := data.Get("SpawnInstance_")
	if err != nil {
		return errors.Wrap(err, "SpawnInstance_")
	}
	systemInstance.Set("ElementName", m.vmName)
	systemInstance.Set("Notes", []string{"VM for mysterium node"})
	systemInstance.Set("VirtualSystemSubType", "Microsoft:Hyper-V:SubType:2")
	systemInstance.Set("SecureBootEnabled", false)
	systemInstance.Set("UserSnapshotType", 2)

	systemText, err := systemInstance.GetText(1)
	if err != nil {
		return errors.Wrap(err, "GetText")
	}

	// memory
	data, err = m.con.Get(MemorySettingData)
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

	// create vm
	jobPath1 := ole.VARIANT{}
	resultingSystem1 := ole.VARIANT{}
	jobState1, err := m.vsMgr.Get("DefineSystem", systemText, []string{sysMemoryDataStr}, nil, &resultingSystem1, &jobPath1)
	if err != nil {
		return errors.Wrap(err, "DefineSystem")
	}
	err = m.waitForJob(jobState1, jobPath1)
	if err != nil {
		return err
	}

	vmLocationURI := getPathFromResultingSystem(resultingSystem1)
	fmt.Println("vmLocationURI:", vmLocationURI)

	loc, err := wmi.NewLocation(vmLocationURI)
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
	fmt.Println("vm id", id.Value())

	// add SCSI controller
	scsiControllerRes, err := m.getDefaultClassValue(ResourceAllocationSettingData, scsiType)
	if err != nil {
		return err
	}
	newID, err := utils.UUID4()
	if err != nil {
		return errors.Wrap(err, "UUID4")
	}
	if scsiControllerRes.Set("VirtualSystemIdentifiers", []string{fmt.Sprintf("{%s}", newID)}); err != nil {
		return errors.Wrap(err, "VirtualSystemIdentifiers")
	}
	scsiCtrlStr, err := scsiControllerRes.GetText(1)
	if err != nil {
		return errors.Wrap(err, "GetText")
	}

	jobPath2 := ole.VARIANT{}
	resultingSystem2 := ole.VARIANT{}
	jobState2, err := m.vsMgr.Get("AddResourceSettings", vmLocationURI, []string{scsiCtrlStr}, &resultingSystem2, &jobPath2)
	if err != nil {
		return errors.Wrap(err, "AddResourceSettings")
	}
	err = m.waitForJob(jobState2, jobPath2)
	if err != nil {
		return err
	}
	scsiControllerURI := getPathFromResultingResourceSettings(resultingSystem2)
	fmt.Println("scsiControllerURI:", scsiControllerURI)

	// add disk drive
	diskRes, err := m.getDefaultClassValue(ResourceAllocationSettingData, diskType)
	if err != nil {
		return errors.Wrap(err, "getDefaultClassValue")
	}
	diskRes.Set("Address", 0)
	diskRes.Set("AddressOnParent", 0)
	diskRes.Set("Parent", scsiControllerURI)
	diskResStr, err := diskRes.GetText(1)
	if err != nil {
		return errors.Wrap(err, "GetText")
	}

	// add disk to VM config
	jobPath3 := ole.VARIANT{}
	resultingSystem3 := ole.VARIANT{}
	jobState3, err := m.vsMgr.Get("AddResourceSettings", vmLocationURI, []string{diskResStr}, &resultingSystem3, &jobPath3)
	if err != nil {
		return errors.Wrap(err, "AddResourceSettings")
	}
	err = m.waitForJob(jobState3, jobPath3)
	if err != nil {
		return err
	}
	diskLocationURI := getPathFromResultingResourceSettings(resultingSystem3)
	fmt.Println("diskLocationURI:", diskLocationURI)

	// add vhdx disk
	vhdRes, err := m.getDefaultClassValue(StorageAllocSettingDataClass, vhdType)
	if err != nil {
		return errors.Wrap(err, "getDefaultClassValue")
	}
	if err := vhdRes.Set("Parent", diskLocationURI); err != nil {
		return errors.Wrap(err, "Parent")
	}
	if err := vhdRes.Set("HostResource", []string{vhdFilePath}); err != nil {
		return errors.Wrap(err, "HostResource")
	}
	vhdResStr, err := vhdRes.GetText(1)
	if err != nil {
		return errors.Wrap(err, "GetText")
	}

	// add disk to VM config
	jobPath4 := ole.VARIANT{}
	resultingSystem4 := ole.VARIANT{}
	jobState4, err := m.vsMgr.Get("AddResourceSettings", vmLocationURI, []string{vhdResStr}, &resultingSystem4, &jobPath4)
	if err != nil {
		return errors.Wrap(err, "AddResourceSettings")
	}
	if err := m.waitForJob(jobState4, jobPath4); err != nil {
		return err
	}

	// hyperv-wmi adapter
	networkRes, err := m.getDefaultClassValue(vm.SyntheticEthernetPortSettingDataClass, "")
	if err != nil {
		return err
	}
	newID2, err := utils.UUID4()
	if err != nil {
		return errors.Wrap(err, "UUID4")
	}
	if networkRes.Set("VirtualSystemIdentifiers", []string{fmt.Sprintf("{%s}", newID2)}); err != nil {
		return errors.Wrap(err, "VirtualSystemIdentifiers")
	}
	if err := networkRes.Set("ElementName", "Myst Network VM Adapter"); err != nil {
		return errors.Wrap(err, "set ElementName")
	}
	networkStr, err := networkRes.GetText(1)
	if err != nil {
		return errors.Wrap(err, "GetText")
	}

	jobPath5 := ole.VARIANT{}
	resultingSystem5 := ole.VARIANT{}
	jobState5, err := m.vsMgr.Get("AddResourceSettings", vmLocationURI, []string{networkStr}, &resultingSystem5, &jobPath5)
	if err != nil {
		return errors.Wrap(err, "AddResourceSettings")
	}
	err = m.waitForJob(jobState5, jobPath5)
	if err != nil {
		return err
	}
	networkURI := getPathFromResultingResourceSettings(resultingSystem5)
	fmt.Println("networkURI", networkURI)

	// connect adapter to switch
	sw, _ := m.GetVirtSwitchByName(switchName)
	swPath, _ := sw.Path()

	portAllocRes, err := m.getDefaultClassValue(vm.EthernetPortAllocationSettingDataClass, "")
	if err != nil {
		return err
	}
	if err := portAllocRes.Set("Parent", networkURI); err != nil {
		return errors.Wrap(err, "Set")
	}
	if err := portAllocRes.Set("HostResource", []string{swPath}); err != nil {
		return errors.Wrap(err, "Set")
	}
	portAllocStr, err := portAllocRes.GetText(1)
	if err != nil {
		return errors.Wrap(err, "GetText")
	}

	jobPath6 := ole.VARIANT{}
	resultingSystem6 := ole.VARIANT{}
	jobState6, err := m.vsMgr.Get("AddResourceSettings", vmLocationURI, []string{portAllocStr}, &resultingSystem6, &jobPath6)
	if err != nil {
		return errors.Wrap(err, "AddResourceSettings")
	}
	err = m.waitForJob(jobState6, jobPath6)
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) StartVM() error {
	fmt.Println("StartVM")
	vm, err := m.GetVM()
	if err != nil {
		return errors.Wrap(err, "GetOne")
	}

	jobPath := ole.VARIANT{}
	jobState, err := vm.Get("RequestStateChange", StateEnabled, &jobPath, nil)
	if err != nil {
		return errors.Wrap(err, "RequestStateChange")
	}

	return m.waitForJob(jobState, jobPath)
}

func (m *Manager) StopVM() error {
	fmt.Println("StopVM")
	vm, err := m.GetVM()
	if err != nil {
		return errors.Wrap(err, "GetOne")
	}

	jobPath := ole.VARIANT{}
	jobState, err := vm.Get("RequestStateChange", StateDisabled, &jobPath, nil)
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
	sw, err := m.con.GetOne(VMSwitchClass, []string{}, qParams)
	if err != nil && !errors.Is(err, wmi.ErrNotFound) {
		return errors.Wrap(err, "GetOne")
	}
	if err == nil && sw != nil {
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

func (m *Manager) GetVirtSwitchByName(switchName string) (*wmi.Result, error) {
	qParams := []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "ElementName", Value: switchName, Type: wmi.Equals}},
	}
	return m.con.GetOne(VMSwitchClass, []string{}, qParams)
}

func (m *Manager) GetGuestKVP() error {
	vm, err := m.GetVM()
	if err != nil {
		return err
	}
	vmID, err := vm.GetProperty("Name")
	if err != nil {
		return errors.Wrap(err, "GetProperty")
	}

	qParams := []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "SystemName", Value: vmID.Value(), Type: wmi.Equals}},
	}
	res, err := m.con.GetOne("Msvm_KvpExchangeComponent", []string{}, qParams)
	if err != nil {
		return errors.Wrap(err, "GetOne")
	}
	p, err := res.GetProperty("GuestIntrinsicExchangeItems")
	if err != nil {
		return errors.Wrap(err, "GetProperty")
	}

	m.Kvp = decodeXMLArray(p.ToArray().ToValueArray())
	return nil
}

func (m *Manager) StartGuestFileService() error {
	vm, err := m.GetVM()
	if err != nil {
		return err
	}
	vmPath, _ := vm.Path()
	fmt.Println("VM:", vmPath)

	// get instance of guestFileService for a _running_ vm
	assoc, err := vm.Get("associators_", "Msvm_SystemDevice", "Msvm_GuestServiceInterfaceComponent", nil, nil, false)
	if err != nil {
		return errors.Wrap(err, "Get")
	}
	guestServiceInterfaceComponent, err := assoc.ItemAtIndex(0)
	if err != nil {
		return errors.Wrap(err, "ItemAtIndex")
	}

	assoc, err = guestServiceInterfaceComponent.Get("associators_", "Msvm_RegisteredGuestService", "Msvm_GuestFileService", nil, nil, false)
	if err != nil {
		return errors.Wrap(err, "Get")
	}
	m.guestFileService, err = assoc.ItemAtIndex(0)
	if err != nil {
		return errors.Wrap(err, "Get")
	}
	return nil
}

func (m *Manager) EnableGuestServices() error {
	vm, err := m.GetVM()
	if err != nil {
		return err
	}
	vmPath, _ := vm.Path()
	fmt.Println("VM:", vmPath)

	// enable Guest Service Interface Component
	assoc, err := vm.Get("associators_", nil, "Msvm_VirtualSystemSettingData")
	if err != nil {
		return errors.Wrap(err, "Get")
	}
	virtualSystemSettingData, _ := assoc.ItemAtIndex(0)
	assoc, err = virtualSystemSettingData.Get("associators_", nil, "Msvm_GuestServiceInterfaceComponentSettingData")
	if err != nil {
		return errors.Wrap(err, "Get")
	}
	guestSvcSettingData, err := assoc.ItemAtIndex(0)
	if err != nil {
		return errors.Wrap(err, "ItemAtIndex")
	}
	guestSvcSettingData.Set("EnabledState", StateEnabled)
	guestSvcSettingDataStr, _ := guestSvcSettingData.GetText(2)

	jobPath := ole.VARIANT{}
	resultingSystem := ole.VARIANT{}
	jobState, err := m.vsMgr.Get("ModifyGuestServiceSettings", []string{guestSvcSettingDataStr}, &resultingSystem, &jobPath)
	if err != nil {
		return errors.Wrap(err, "DefineSystem")
	}
	err = m.waitForJob(jobState, jobPath)
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) CopyFile(src, dst string) error {

	// create switch settings in xml representation
	copyFileToGuestSettingDataClass, err := m.con.Get("Msvm_CopyFileToGuestSettingData")
	if err != nil {
		return errors.Wrap(err, "Get")
	}
	wmiCopyFileToGuestSettingData, err := copyFileToGuestSettingDataClass.Get("SpawnInstance_")
	if err != nil {
		return errors.Wrap(err, "SpawnInstance_")
	}
	wmiCopyFileToGuestSettingData.Set("SourcePath", src)
	wmiCopyFileToGuestSettingData.Set("DestinationPath", dst)
	wmiCopyFileToGuestSettingData.Set("OverwriteExisting", true)
	wmiCopyFileToGuestSettingData.Set("CreateFullPath", true)
	copyCmdTxt, _ := wmiCopyFileToGuestSettingData.GetText(2)

	jobPath := ole.VARIANT{}
	jobState, err := m.guestFileService.Get("CopyFilesToGuest", []string{copyCmdTxt}, &jobPath)
	if err != nil {
		return errors.Wrap(err, "CopyFilesToGuest")
	}

	_ = jobState
	//fmt.Println("jobState", jobState, jobPath.Value())
	//fmt.Println("jobState", jobState.Value(), jobState.Value().(int32))
	//j, err := NewJobState(jobPath.Value().(string))
	//fmt.Println(j, err)

	return nil
}

func (m *Manager) WaitUntilBooted(pollEvery, timeout time.Duration) error {
	fmt.Printf("waiting for VM `%s` to boot\n", m.vmName)
	for {
		select {
		case <-time.After(pollEvery):
			err := m.GetGuestKVP()
			if err != nil {
				fmt.Printf("unexpected error while waiting for VM `%s` to boot, %s\n", m.vmName, err)
				return err
			}
			ip := m.Kvp["NetworkAddressIPv4"]
			if ip != "" {
				fmt.Println("VM IP:", ip)
				return nil
			}

		case <-time.After(timeout):
			fmt.Printf("time out while waiting for VM `%s` to boot\n", m.vmName)
			return errors.New("Timeout")
		}
	}
}

func (m *Manager) RemoveVM() error {
	vm, err := m.GetVM()
	if errors.Is(err, wmi.ErrNotFound) {
		return nil
	}
	if err != nil {
		return errors.Wrap(err, "GetVM")
	}

	vmPath, err := vm.Path()
	if err != nil {
		return errors.Wrap(err, "GetVM")
	}

	// stop
	jobPath1 := ole.VARIANT{}
	jobState1, err := vm.Get("RequestStateChange", StateDisabled, &jobPath1, nil)
	if err != nil {
		return errors.Wrap(err, "RequestStateChange")
	}
	m.waitForJob(jobState1, jobPath1)
	if err != nil {
		return err
	}

	// remove
	jobPath2 := ole.VARIANT{}
	jobState2, err := m.vsMgr.Get("DestroySystem", vmPath, &jobPath2)
	if err != nil {
		return errors.Wrap(err, "DestroySystem")
	}
	return m.waitForJob(jobState2, jobPath2)
}