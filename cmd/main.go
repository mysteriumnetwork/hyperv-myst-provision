package main

import (
	"fmt"
	"github.com/mysteriumnetwork/hyperv-node/archiver"
	"github.com/mysteriumnetwork/hyperv-node/common"
	"github.com/mysteriumnetwork/hyperv-node/downloader"
	"github.com/mysteriumnetwork/hyperv-node/hyperv"
	"github.com/mysteriumnetwork/hyperv-node/powershell"
	"github.com/mysteriumnetwork/hyperv-node/provisioner"
	"os"
	"reflect"
	"runtime"
)

const (
	vmName = "Myst HyperV Alpine"

	vmBaseImageName = "Myst.HyperV.Alpine.zip"
	vmBaseImageUrl  = "https://github.com/mysteriumnetwork/hyperv-myst-provision/releases/download/0.01-hyperV-image/" + vmBaseImageName

	exportPath   = `E:\vm-export`
	privateKey   = `E:\hyperv\ssh\id_rsa`
	publicKey    = `E:\hyperv\ssh\id_rsa.pub`
	dotMysterium = ".mysterium"
)

func main() {
	doOrDie(cleanup)

	dirs, err := common.OSDirs()
	if err != nil {
		panic(err)
	}

	shell := powershell.New(powershell.OptionDebugPrint)

	dloader := downloader.NewDLoader(shell)
	doOrDie(func() error {
		return dloader.DownloadAndExtract(vmBaseImageUrl, dirs.Temp)
	})

	hyperV := hyperv.New(vmName, dirs.Temp, dirs.VMExport, shell)
	hyperV.StopVM()
	hyperV.RemoveVM()
	doOrDie(hyperV.ImportVM)
	doOrDie(func() error {
		return hyperV.SetVMMaxRam(512)
	})
	doOrDie(hyperV.CreateExternalNetworkSwitchIfNotExistsAndAssign)
	doOrDie(hyperV.StartVM)
	doOrDie(hyperV.WaitUntilBooted)

	provision, err := provisioner.NewProvisioner(shell)
	if err != nil {
		panic(err)
	}
	doOrDie(func() error {
		address, err := hyperV.VMIP4Address()
		if err != nil {
			return err
		}
		return provision.CopyKeystoreRecursive(
			fmt.Sprintf("%s/%s/keystore", dirs.Home, dotMysterium),
			"/root/.mysterium",
			privateKey,
			"root",
			address,
		)
	})

	doOrDie(func() error {
		address, err := hyperV.VMIP4Address()
		if err != nil {
			return err
		}
		return provision.InstallMystClean(privateKey, "root", address)
	})

	doOrDie(hyperV.StopVM)
	doOrDie(hyperV.RemoveVMSnapshots)
	doOrDie(hyperV.ExportVM)
	doOrDie(hyperV.RemoveVM)

	doOrDie(func() error {
		zipName := fmt.Sprintf("%s-%s.zip", vmName, provision.NodeVersion)
		return archiver.Archive(common.Path(dirs.VMExport, vmName), common.Path(dirs.VMExport, zipName))
	})
}

func cleanup() error {
	dirs, err := common.OSDirs()
	if err != nil {
		return err
	}
	return os.RemoveAll(dirs.Temp)
}

func doOrDie(task func() error) {
	name := runtime.FuncForPC(reflect.ValueOf(task).Pointer()).Name()
	fmt.Println("> ", name)
	if err := task(); err != nil {
		fmt.Println(">>> ", err, " <<<")
		panic(err)
	}
}
