package network

// Hyper-V networking constants
const (
	switchName = "Myst Bridge Switch"

	ExternalPort                  = "Msvm_ExternalEthernetPort"
	ComputerSystem                = "Msvm_ComputerSystem"
	VMSwitchClass                 = "Msvm_VirtualEthernetSwitch"
	VMSwitchSettings              = "Msvm_VirtualEthernetSwitchSettingData"
	VMSystemSettingData           = "Msvm_VirtualSystemSettingData"
	ResourceAllocationSettingData = "Msvm_ResourceAllocationSettingData"
	StorageAllocSettingDataClass  = "Msvm_StorageAllocationSettingData"

	VMSwitchManagementService = "Msvm_VirtualEthernetSwitchManagementService"
	VMSystemManagementService = "Msvm_VirtualSystemManagementService"
	VMImageManagementService  = "Msvm_ImageManagementService"

	WIFIPort                       = "Msvm_WiFiPort"
	EthernetSwitchPort             = "Msvm_EthernetSwitchPort"
	PortAllocSetData               = "Msvm_EthernetPortAllocationSettingData"
	PortVLANSetData                = "Msvm_EthernetSwitchPortVlanSettingData"
	PortSecuritySetData            = "Msvm_EthernetSwitchPortSecuritySettingData"
	PortAllocACLSetData            = "Msvm_EthernetSwitchPortAclSettingData"
	PortExtACLSetData              = PortAllocACLSetData
	MsvmMemorySettingData          = "Msvm_MemorySettingData"
	MsvmVirtualHardDiskSettingData = "Msvm_VirtualHardDiskSettingData"

	LANEndpoint                 = "Msvm_LANEndpoint"
	CIMResAllocSettingDataClass = "CIM_ResourceAllocationSettingData"
	StateDisabled               = 3
	OperationModeAccess         = 1
	OperationModeTrunk          = 2
	ETHConnResSubType           = "Microsoft:Hyper-V:Ethernet Connection"
	NetAdapterClass             = "MSFT_NetAdapter"
)
