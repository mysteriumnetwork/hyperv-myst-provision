package hyperv_wmi

import "errors"

// Hyper-V networking constants
const (
	ExternalPort                  = "Msvm_ExternalEthernetPort"
	ComputerSystemClass           = "Msvm_ComputerSystem"
	VMSwitchClass                 = "Msvm_VirtualEthernetSwitch"
	VMSwitchSettings              = "Msvm_VirtualEthernetSwitchSettingData"
	VMSystemSettingData           = "Msvm_VirtualSystemSettingData"
	ResourceAllocationSettingData = "Msvm_ResourceAllocationSettingData"
	StorageAllocSettingDataClass  = "Msvm_StorageAllocationSettingData"
	VMSwitchManagementService     = "Msvm_VirtualEthernetSwitchManagementService"
	VMSystemManagementService     = "Msvm_VirtualSystemManagementService"
	VMImageManagementService      = "Msvm_ImageManagementService"
	PortAllocSetData              = "Msvm_EthernetPortAllocationSettingData"
	MemorySettingData             = "Msvm_MemorySettingData"

	StateDisabled = 3
	StateEnabled  = 2
)

var (
	errEmptyIP = errors.New("could not resolve IP address")
)
