package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/itzg/go-flagsfiller"
	"github.com/mysteriumnetwork/hyperv-node/archiver"
	"github.com/mysteriumnetwork/hyperv-node/common"
	"github.com/mysteriumnetwork/hyperv-node/downloader"
	"github.com/mysteriumnetwork/hyperv-node/hyperv"
	"github.com/mysteriumnetwork/hyperv-node/powershell"
	"github.com/mysteriumnetwork/hyperv-node/provisioner"
	"log"
	"os"
	"time"
)

type flagsSet struct {
	BaseImageURL         string `default:"https://github.com/mysteriumnetwork/hyperv-myst-provision/releases/download/0.01-hyperV-image/Myst.HyperV.Alpine.zip" usage:"url of preconfigured base image to install latest node to"`
	InstallScriptURL     string `default:"https://raw.githubusercontent.com/mysteriumnetwork/hyperv-myst-provision/mvp/provision/assets/alpine/install-myst.sh" usage:"url of provisioning ash script that will install myst node and it's dependencies'"`
	VMName               string `default:"Myst HyperV Alpine" usage:"hyper-v guest VM name"`
	VMBootPollSeconds    int64  `default:"5" usage:"poll interval (seconds) to check whether guest VM has booted"`
	VMBootTimeoutMinutes int64  `default:"5" usage:"timeout period (minutes) in case no successful response from guest VM"`
	PrivateKeyDir        string `usage:"private key for ssh operation in guest VM"`
	NodeVersion          string `usage:"node version to be installed on guest VM, if no specified latest will be used"`
	WorkDir              string `usage:"work directory where all files will be downloaded and exported too. If unspecified, will use OS temp directory."`
}

func (fs *flagsSet) validate() error {
	if fs.PrivateKeyDir == "" {
		return errors.New("-private-key-dir is required")
	}

	if exists, _ := common.Exists(fs.PrivateKeyDir); !exists {
		return errors.New("-private-key-dir - no file")
	}
	return nil
}

var flags flagsSet
var shell = powershell.New(powershell.OptionDebugPrint)

func main() {
	flagsParse()

	err := cleanup()
	if err != nil {
		log.Fatal("error cleaning up", err)
	}

	dirs, err := common.WorkingDirs(flags.WorkDir)
	if err != nil {
		log.Fatal(err)
	}

	err = downloader.NewDLoader(shell).DownloadAndExtract(flags.BaseImageURL, dirs.WorkDir)
	if err != nil {
		log.Fatal(err)
	}

	hyperV := hyperv.New(flags.VMName, dirs.WorkDir, dirs.VMExport, shell)
	err = hyperV.ImportVM()
	if err != nil {
		log.Fatal(err)
	}

	err = hyperV.SetVMMaxRam(512)
	if err != nil {
		log.Fatal(err)
	}

	err = hyperV.CreateExternalNetworkSwitchIfNotExistsAndAssign()
	if err != nil {
		log.Fatal(err)
	}

	err = hyperV.StartVM()
	if err != nil {
		log.Fatal(err)
	}

	err = hyperV.WaitUntilBooted(
		time.Duration(flags.VMBootPollSeconds)*time.Second,
		time.Duration(flags.VMBootTimeoutMinutes)*time.Minute,
	)
	if err != nil {
		log.Fatal(err)
	}

	vmIP, err := hyperV.VMIP4Address()
	if err != nil {
		log.Fatal(err)
	}

	provision, err := provisioner.NewProvisioner(shell, flags.NodeVersion)
	if err != nil {
		log.Fatal(err)
	}

	err = provision.InstallMystClean(flags.PrivateKeyDir, "root", vmIP, flags.InstallScriptURL)
	if err != nil {
		log.Fatal(err)
	}

	err = hyperV.StopVM()
	if err != nil {
		log.Fatal(err)
	}

	err = hyperV.RemoveVMSnapshots()
	if err != nil {
		log.Fatal(err)
	}

	err = hyperV.DisconnectVMNetworkSwitch()
	if err != nil {
		log.Fatal(err)
	}

	err = hyperV.ExportVM()
	if err != nil {
		log.Fatal(err)
	}

	err = hyperV.RemoveVM()
	if err != nil {
		log.Fatal(err)
	}

	zipName := fmt.Sprintf("%s-%s.zip", flags.VMName, provision.NodeVersion)
	err = archiver.Archive(common.Path(dirs.VMExport, flags.VMName), common.Path(dirs.VMExport, zipName))
	if err != nil {
		log.Fatal(err)
	}
}

func flagsParse() {
	if err := flagsfiller.New().Fill(flag.CommandLine, &flags); err != nil {
		log.Fatal(err)
	}
	flag.Parse()
	if err := flags.validate(); err != nil {
		log.Fatal(err)
	}

}

func cleanup() error {
	dirs, err := common.WorkingDirs(flags.WorkDir)
	hyperV := hyperv.New(flags.VMName, dirs.WorkDir, dirs.VMExport, shell)
	hyperV.StopVM()
	hyperV.RemoveVM()
	if err != nil {
		return err
	}
	return os.RemoveAll(dirs.WorkDir)
}
