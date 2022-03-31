package vbox

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/pkg/errors"
	"github.com/terra-farm/go-virtualbox"

	"github.com/mysteriumnetwork/hyperv-node/model"
	"github.com/mysteriumnetwork/hyperv-node/service2/util/winutil"
)

const (
	macAddress = "00155D21A42C"
	VM         = "Myst VM (alpine)"

	KeyIPv4   = "/VirtualBox/GuestInfo/Net/0/V4/IP"
	KeyOSProd = "/VirtualBox/GuestInfo/OS/Product"
)

type Manager struct {
	cfg    *model.Config
	vmName string

	cimv2 *wmi.WMI

	/* hyperv */
	// con              *wmi.WMI
	// wmi              *wmi.WMI
	// switchMgr        *wmi.Result
	// vsMgr            *wmi.Result
	// imageMgr         *wmi.Result
	// guestFileService *wmi.Result // file copy service

	// guest KV map
	Kvp map[string]interface{}

	// ethernet notifier
	notifier winutil.Notifier
}

// NewVMManager returns a new Manager type
func NewVMManager(vmName string, cfg *model.Config) (*Manager, error) {
	cimv2, err := wmi.NewConnection(".", `root\cimv2`)
	if err != nil {
		return nil, err
	}

	// w, err := wmi.NewConnection(".", `root\virtualization\v2`)
	// if err != nil {
	// 	return nil, err
	// }

	// wmi_, err := wmi.NewConnection(".", `root\wmi`)
	// if err != nil {
	// 	return nil, err
	// }

	// // Get virtual switch management service
	// switchMgr, err := w.GetOne(VMSwitchManagementService, []string{}, []wmi.Query{})
	// if err != nil {
	// 	return nil, err
	// }
	// vsMgr, err := w.GetOne(VMSystemManagementService, []string{}, []wmi.Query{})
	// if err != nil {
	// 	return nil, err
	// }
	// imageMgr, err := w.GetOne(VMImageManagementService, []string{}, []wmi.Query{})
	// if err != nil {
	// 	return nil, err
	// }

	n := winutil.NewNotifier()

	sw := &Manager{
		vmName: vmName,
		cfg:    cfg,
		cimv2:  cimv2,

		// con:       w,
		// wmi:       wmi_,
		// switchMgr: switchMgr,
		// vsMgr:     vsMgr,
		// imageMgr:  imageMgr,
		Kvp:      nil,
		notifier: n,
	}
	return sw, nil
}

func (mn *Manager) GetVM() (*virtualbox.Machine, error) {
	m, err := virtualbox.GetMachine(VM)
	return m, err
}

func (mn *Manager) CreateVM(vhdFilePath string, opt ImportOptions) error {
	log.Println("CreateVM >>", opt)
	log.Println(os.Getwd())
	cwd, _ := os.Getwd()

	m, err := virtualbox.CreateMachine(VM, cwd)
	log.Println("CreateVM >>", m, err)
	if errors.Is(err, virtualbox.ErrMachineExist) {
		m, err = virtualbox.GetMachine(VM)
		log.Println("m>>", m, err)
	}
	log.Println("CreateVM >>", m)

	m.Flag |= virtualbox.IOAPIC
	m.Flag |= virtualbox.RTCUSEUTC
	m.Flag |= virtualbox.ACPI
	m.Firmware = "EFI"
	m.OSType = "Linux_64"
	m.BootOrder = []string{"disk", "none", "none", "none"}
	m.Memory = 256
	m.Modify()

	//bridgeadapter1 := "Intel(R) Dual Band Wireless-AC 8260"

	err = m.SetNIC(1, virtualbox.NIC{
		Network:       virtualbox.NICNetBridged,
		Hardware:      virtualbox.IntelPro1000MTDesktop,
		HostInterface: opt.AdapterName,
		MacAddr:       macAddress,
	})
	log.Println("SetNIC", err)

	storageCtl := VM + "_IDE_1"
	err = m.AddStorageCtl(storageCtl, virtualbox.StorageController{
		SysBus:      virtualbox.SysBusIDE,
		Ports:       2,
		Chipset:     "PIIX4",
		HostIOCache: true,
		Bootable:    true,
	})
	log.Println("AddStorageCtl", err)

	// img := `C:\Users\user\src\hyperv-myst-provision\alpine-vm-disk.vdi`
	img := `C:\ProgramData\MystVmHelper\vhdx\alpine-vm-disk.vdi`

	err = m.AttachStorage(storageCtl, virtualbox.StorageMedium{Port: 0, Device: 0, DriveType: virtualbox.DriveHDD, Medium: img})
	log.Println("AttachStorage", err)

	return nil
}

func (mn *Manager) StartVM() error {
	fmt.Println("StartVM")
	vm, err := mn.GetVM()
	if err != nil {
		return errors.Wrap(err, "GetOne")
	}
	err = vm.Start()

	virtualbox.SetGuestProperty(VM, KeyIPv4, "")
	return err
}

func (m *Manager) StopVM() error {
	fmt.Println("StopVM")
	vm, err := m.GetVM()
	if err != nil {
		return errors.Wrap(err, "GetOne")
	}
	err = vm.Stop()
	return err
}

func (m *Manager) GetGuestKVP() error {

	m.Kvp = make(KVMap)

	getKey := func(keySpec, key string) error {
		val, err := virtualbox.GetGuestProperty(VM, keySpec)
		if err != nil && err.Error() != "No match with get guestproperty output" {
			log.Println("GetGuestProperty", err)
			return err
		} else {
			log.Println("GetGuestProperty >", keySpec, val)

			m.Kvp[key] = val
			return nil
		}
	}
	getKey(KeyIPv4, "IP")
	getKey(KeyOSProd, "OS")

	return nil
}

func (m *Manager) StartGuestFileService() error {
	return nil
}

func (m *Manager) EnableGuestServices() error {
	return nil
}

func (m *Manager) CopyFile(src, dst string) error {
	log.Println("CopyFile>", src, dst)
	return nil
}

func (m *Manager) WaitUntilGotIP(pollEvery, timeout time.Duration) error {
	log.Printf("waiting for VM `%s` to boot\n", m.vmName)
	deadline := time.Now().Add(timeout)

	for {
		select {
		case <-time.After(pollEvery):
			err := m.GetGuestKVP()
			if err != nil {
				return errors.Wrap(err, "GetGuestKVP")
			}

			ip := m.Kvp["IP"]
			if ip != nil && ip != "" {
				log.Println("VM IP:", ip)
				return nil
			}

			if time.Now().After(deadline) {
				log.Printf("time out while waiting for VM `%s` to get an IP\n", m.vmName)
				return errors.New("Timeout")
			}
		}
	}
}

func (m *Manager) WaitUntilBoot(pollEvery, timeout time.Duration) error {
	log.Printf("waiting for VM `%s` to boot\n", m.vmName)
	deadline := time.Now().Add(timeout)

	for {
		select {
		case <-time.After(pollEvery):

			err := m.GetGuestKVP()
			if err != nil {
				return errors.Wrap(err, "GetGuestKVP")
			}

			osName := m.Kvp["OS"]
			if osName != nil && osName != "" {
				log.Println("VM OSName:", osName)
				return nil
			}

			if time.Now().After(deadline) {
				log.Printf("time out while waiting for VM `%s` to boot\n", m.vmName)
				return errors.New("Timeout")
			}
		}
	}
}

func (m *Manager) RemoveVM() error {
	vm, err := m.GetVM()
	if errors.Is(err, virtualbox.ErrMachineNotExist) {
		return nil
	}
	if err != nil {
		log.Println("GetVM", vm, err)
		return errors.Wrap(err, "GetVM")
	}

	err = vm.Delete()
	return err
}