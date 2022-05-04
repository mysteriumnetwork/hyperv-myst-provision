package vbox

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/pkg/errors"

	//"github.com/rs/zerolog/log"
	"github.com/terra-farm/go-virtualbox"

	"github.com/mysteriumnetwork/hyperv-node/model"
	"github.com/mysteriumnetwork/hyperv-node/service2/daemon/client"
	"github.com/mysteriumnetwork/hyperv-node/service2/util/winutil"
	"github.com/mysteriumnetwork/hyperv-node/utils"
)

const (
	macAddress = "00155D21A42C"
	VMname     = "Myst VM (alpine)"

	KeyOSProd       = "/VirtualBox/GuestInfo/OS/Product"
	KeyIPv4         = "/VirtualBox/GuestInfo/Net/0/V4/IP"
	KeyInternalIPv4 = "/VirtualBox/GuestInfo/Net/1/V4/IP"

	VMBootPollSeconds    = 3
	VMBootTimeoutMinutes = 1
)

type Manager struct {
	cfg     *model.Config
	vmName  string
	cimV2   *wmi.WMI
	rootWmi *wmi.WMI

	// guest KV map
	Kvp map[string]interface{}

	// ethernet notifier
	notifier   winutil.Notifier
	MinAdapter Adapter

	// restrict concurrent VM start
	mu sync.Mutex
	id uint32 // operation counter - to detect concurrent operations
}

// NewVMManager returns a new Manager type
func NewVMManager(vmName string, cfg *model.Config) (*Manager, error) {
	cimv2, err := wmi.NewConnection(".", `root\cimv2`)
	if err != nil {
		return nil, err
	}

	rootWmi, err := wmi.NewConnection(".", `root\WMI`)
	if err != nil {
		return nil, err
	}

	n := winutil.NewNotifier()
	sw := &Manager{
		vmName:   vmName,
		cfg:      cfg,
		cimV2:    cimv2,
		rootWmi:  rootWmi,
		Kvp:      nil,
		notifier: n,
	}
	return sw, nil
}

func (m *Manager) getVM() (*virtualbox.Machine, error) {
	vm, err := virtualbox.GetMachine(VMname)
	return vm, err
}

func (m *Manager) CreateVM(vhdFilePath string, opt ImportOptions) error {
	log.Println("CreateVM >>", opt)
	cwd, _ := os.Getwd()
	log.Println(cwd)

	vm, err := virtualbox.CreateMachine(VMname, cwd)
	log.Println(vm, err)

	if errors.Is(err, virtualbox.ErrMachineExist) {
		vm, err = virtualbox.GetMachine(VMname)
		log.Println("vm>", vm, err)
	}

	vm.Flag |= virtualbox.IOAPIC
	vm.Flag |= virtualbox.RTCUSEUTC
	vm.Flag |= virtualbox.ACPI
	vm.Firmware = "EFI"
	vm.OSType = "Linux_64"
	vm.BootOrder = []string{"disk", "none", "none", "none"}
	vm.Memory = 256
	vm.Modify()

	err = vm.SetNIC(1, virtualbox.NIC{
		Network:       virtualbox.NICNetBridged,
		Hardware:      virtualbox.VirtIO,
		HostInterface: opt.AdapterName,
		MacAddr:       macAddress,
	})
	log.Println("SetNIC", err)

	// define internal host-only network
	err = vm.SetNIC(2, virtualbox.NIC{
		Network:       virtualbox.NICNetHostonly,
		Hardware:      virtualbox.VirtIO,
		HostInterface: "VirtualBox Host-Only Ethernet Adapter",
		MacAddr:       macAddress,
	})
	log.Println("SetNIC", err)

	storageCtl := VMname + "_IDE_1"
	err = vm.AddStorageCtl(storageCtl, virtualbox.StorageController{
		SysBus:      virtualbox.SysBusIDE,
		Ports:       2,
		Chipset:     "PIIX4",
		HostIOCache: true,
		Bootable:    true,
	})
	log.Println("AddStorageCtl", err)

	img := filepath.Join(cwd, `vhdx\alpine-vm-disk.vdi`)
	log.Println("img >", img)

	err = vm.AttachStorage(storageCtl, virtualbox.StorageMedium{Port: 0, Device: 0, DriveType: virtualbox.DriveHDD, Medium: img})
	log.Println("AttachStorage", err)

	return nil
}

func (m *Manager) SetNicAndRestartVM() error {
	log.Println("Manager !SetNicAndRestartVM")

	vm, err := m.getVM()
	if err != nil {
		return errors.Wrap(err, "GetOne")
	}
	if err = vm.Stop(); err != nil {
		return err
	}

	err = utils.Retry(5, 1*time.Second, func() error {
		err := vm.SetNIC(1, virtualbox.NIC{
			Network:       virtualbox.NICNetBridged,
			Hardware:      virtualbox.VirtIO,
			HostInterface: m.MinAdapter.Name,
			MacAddr:       macAddress,
		})
		log.Println("Manager !SetNicAndRestartVM !SetNIC", err)
		return err
	})

	if err = vm.Start(); err != nil {
		return err
	}
	virtualbox.SetGuestProperty(VMname, KeyIPv4, "")
	return nil
}

func (m *Manager) actionExecutor(action func() error, actName string) error {

	nextID := atomic.AddUint32(&m.id, 1)
	log.Printf("%s > nextID %d ", actName, nextID)

	m.mu.Lock()
	defer m.mu.Unlock()

	ID := atomic.LoadUint32(&m.id)
	log.Printf("%s > nextID %d ID %d", actName, nextID, ID)

	if ID == nextID {
		err := action()
		atomic.AddUint32(&m.id, 1)
		return err
	} else {
		log.Println("Concurrent action detected")
		atomic.AddUint32(&m.id, 1)
	}
	return nil
}

// always restart
func (m *Manager) RestartVMAndWait() error {

	// start VM only if network is online
	if !m.IsNetworkOnline() {
		err := errors.New("Network is not online")
		return err
	}

	action := func() error {
		err := m.SetNicAndRestartVM()
		if err != nil {
			log.Println("SetNicAndRestartVM failed")
			return err
		}
		m.WaitVMReady()
		m.ImportKeystore(nil)
		m.SetLauncherVersion()
		return nil
	}
	return m.actionExecutor(action, "RestartVMAndWait")
}

func (m *Manager) StartVM() error {
	log.Println("Manager !StartVM")
	vm, err := m.getVM()
	if err != nil {
		return errors.Wrap(err, "GetOne")
	}
	err = vm.Start()

	virtualbox.SetGuestProperty(VMname, KeyIPv4, "")
	return err
}

func (m *Manager) WaitVMReady() error {
	log.Println("Manager !StartVM")

	err := m.WaitUntilBoot(
		time.Duration(VMBootPollSeconds)*time.Second,
		time.Duration(VMBootTimeoutMinutes)*time.Minute,
	)
	if err != nil {
		log.Println("WaitUntilBoot", err)
		return errors.Wrap(err, "WaitUntilBoot")
	}
	log.Println("WaitUntilBoot OK>")

	err = m.WaitUntilGotIP(
		time.Duration(VMBootPollSeconds)*time.Second,
		time.Duration(VMBootTimeoutMinutes)*time.Minute,
	)
	if err != nil {
		log.Println("WaitUntilGotIP", err)
		return errors.Wrap(err, "WaitUntilBoot")
	}
	log.Println("WaitUntilGotIP OK>")

	return err
}

func (m *Manager) IsNetworkOnline() bool {
	log.Println("Manager !IsNetworkOnline")
	return m.MinAdapter.Metric > 0
}

func (m *Manager) StopVM() error {
	log.Println("StopVM")
	vm, err := m.getVM()
	if err != nil {
		return errors.Wrap(err, "GetOne")
	}
	err = vm.Stop()
	return err
}

func (m *Manager) GetGuestKVP() error {

	m.Kvp = make(model.KVMap)

	getKey := func(keySpec, key string) error {
		val, err := virtualbox.GetGuestProperty(VMname, keySpec)
		if err != nil && err.Error() != "No match with get guestproperty output" {
			log.Println("GetGuestProperty", err)
			return err
		} else {
			// log.Println("GetGuestProperty >", keySpec, val)

			m.Kvp[key] = val
			return nil
		}
	}
	getKey(KeyIPv4, "IP")
	getKey(KeyInternalIPv4, "IP_int")
	getKey(KeyOSProd, "OS")

	return nil
}

func (m *Manager) StartGuestFileService() error {
	return nil
}

func (m *Manager) EnableGuestServices() error {
	return nil
}

func (m *Manager) CopyFile(src string) error {
	log.Println("CopyFile>", src)

	ip := m.Kvp["IP_int"].(string)
	err := client.VmAgentUploadKeystore(ip, src)
	return err
}

func (m *Manager) SetLauncherVersion() error {
	log.Println("SetLauncherVersion>")

	ip := m.Kvp["IP_int"].(string)
	err := utils.Retry(5, 1*time.Second, func() error {
		return client.VmAgentSetLauncherVersion(ip)
	})
	return err
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

			ip := m.Kvp["IP_int"]
			if ip != nil && ip != "" {
				log.Println("VM IP_int:", ip)
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
	vm, err := m.getVM()
	if errors.Is(err, virtualbox.ErrMachineNotExist) {
		return nil
	}
	if err != nil {
		log.Println("getVM", vm, err)
		return errors.Wrap(err, "RemoveVM")
	}

	//cmd := exec.CommandContext()
	//cmd.Run()

	err = vm.Delete()
	return err
}
