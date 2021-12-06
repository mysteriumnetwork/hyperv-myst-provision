package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/itzg/go-flagsfiller"
	"github.com/mysteriumnetwork/hyperv-node/common"
	"github.com/mysteriumnetwork/hyperv-node/hyperv/network"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"
)

type flagsSet struct {
	VMName               string `default:"Myst HyperV Alpine_" usage:"hyper-v guest VM name"`
	WorkDir              string `usage:"path to hyperv VM folder"`
	KeystoreDir          string `usage:"path to keystore folder (C:\Users\<user>\.mysterium\keystore"`
	Force                bool   `default:"false" usage:"will remove any existing VM with same name"`
	VMBootPollSeconds    int64  `default:"5" usage:"poll interval (seconds) to check whether guest VM has booted"`
	VMBootTimeoutMinutes int64  `default:"5" usage:"timeout period (minutes) in case no successful response from guest VM"`
}

func (fs *flagsSet) validate() error {
	if fs.WorkDir == "" {
		return errors.New("-work-dir is required")
	}
	if exists, _ := common.Exists(fs.WorkDir); !exists {
		return errors.New("-work-dir - does not exist")
	}
	return nil
}

var flags flagsSet

func main() {
	flagsParse()

	mgr, err := network.NewVMManager(flags.VMName)
	if err != nil {
		log.Fatal(err)
	}

	if err = mgr.CreateExternalNetworkSwitchIfNotExistsAndAssign(); err != nil {
		log.Fatal(err)
	}

	if flags.Force {
		if err := mgr.RemoveVM(); err != nil {
			log.Fatal(err)
		}
	}
	vm, err := mgr.GetVM()
	if err != nil && !errors.Is(err, wmi.ErrNotFound) {
		log.Fatal(err)
	}
	if vm == nil || errors.Is(err, wmi.ErrNotFound) {
		vhdFilePath := flags.WorkDir + `\alpine-vm-disk\alpine-vm-disk.vhdx`
		err := mgr.CreateVM(vhdFilePath)
		if err != nil {
			fmt.Println(err)
		}
	}
	if err = mgr.EnableGuestServices(); err != nil {
		log.Fatal(err)
	}
	if err = mgr.StartVM(); err != nil {
		log.Fatal(err)
	}
	if err = mgr.StartGuestFileService(); err != nil {
		log.Fatal(err)
	}

	err = mgr.WaitUntilBooted(
		time.Duration(flags.VMBootPollSeconds)*time.Second,
		time.Duration(flags.VMBootTimeoutMinutes)*time.Minute,
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Copy keystore")
	var keystorePath string
	if flags.KeystoreDir != "" {
		keystorePath = flags.KeystoreDir
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}
		keystorePath = fmt.Sprintf(`%s\%s`, homeDir, `.mysterium\keystore`)
	}
	err = filepath.Walk(keystorePath, func(path string, info fs.FileInfo, _ error) error {
		if info.IsDir() {
			return nil
		}

		return mgr.CopyFile(path, "/root/.mysterium/keystore/")
	})
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
