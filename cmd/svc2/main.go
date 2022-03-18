package main

import (
	"errors"
	"fmt"
	"time"

	virtualbox "github.com/terra-farm/go-virtualbox"

	"github.com/mysteriumnetwork/hyperv-node/service/daemon/flags"
	"github.com/mysteriumnetwork/hyperv-node/service/util"
)

func main() {
	defer util.PanicHandler("main")
	flags.Parse()


	VM := "Myst VM (alpine)"
	m, err := virtualbox.CreateMachine(VM, `C:\Users\user\src\VM\`)
	fmt.Println("create", m, err)
	if errors.Is(err, virtualbox.ErrMachineExist) {
		m, err = virtualbox.GetMachine(VM)
		fmt.Println("m>>", m, err)
	}
	fmt.Println("m", m)

	m.Flag |= virtualbox.IOAPIC
	m.Flag |= virtualbox.RTCUSEUTC
	m.Flag |= virtualbox.ACPI
	m.Firmware = "EFI"
	m.OSType = "Linux_64"
	m.BootOrder = []string{"disk", "none", "none", "none"}
	m.Memory = 256
	m.Modify()

	bridgeadapter1 := "Intel(R) Dual Band Wireless-AC 8260"
	macaddress1 := "080027996C69"
	err = m.SetNIC(1, virtualbox.NIC{Network: virtualbox.NICNetBridged, Hardware: virtualbox.IntelPro1000MTDesktop, HostInterface: bridgeadapter1, MacAddr: macaddress1})
	fmt.Println("SetNIC", err)

	storageCtl := VM + "_IDE_0"
	err = m.AddStorageCtl(storageCtl, virtualbox.StorageController{SysBus: virtualbox.SysBusIDE, Ports: 2, Chipset: "PIIX4", HostIOCache: true, Bootable: true})
	fmt.Println("AddStorageCtl", err)

	img := `C:\Users\user\src\hyperv-myst-provision\alpine-vm-disk_copy.vdi`
	err = m.AttachStorage(storageCtl, virtualbox.StorageMedium{Port: 0, Device: 0, DriveType: virtualbox.DriveHDD, Medium: img})
	fmt.Println("AttachStorage", err)

	err = m.Start()
	if err != nil {
		fmt.Println(err)
		return
	}

	key := "/VirtualBox/GuestInfo/Net/0/V4/IP"
	virtualbox.SetGuestProperty(VM, key, "")
	for i := 0; i <= 20; i++ {
		val, err := virtualbox.GetGuestProperty(VM, key)
		if err != nil {
			fmt.Println("GetGuestProperty", err)
		} else {
			fmt.Println("IP:", val)
			return
		}
		time.Sleep(2 * time.Second)
	}

}
