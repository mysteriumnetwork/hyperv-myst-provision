package main

import (
	"errors"
	"flag"
	"github.com/itzg/go-flagsfiller"
	"github.com/mysteriumnetwork/hyperv-node/common"
	hyperv_wmi2 "github.com/mysteriumnetwork/hyperv-node/hyperv-wmi"
	"log"
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

func flagsParse() {
	if err := flagsfiller.New().Fill(flag.CommandLine, &flags); err != nil {
		log.Fatal(err)
	}
	flag.Parse()
	if err := flags.validate(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	flagsParse()

	mgr, err := hyperv_wmi2.NewVMManager(flags.VMName)
	if err != nil {
		log.Fatal(err)
	}

	err = mgr.ImportVM(hyperv_wmi2.ImportOptions{
		Force:                flags.Force,
		WorkDir:              flags.WorkDir,
		VMBootPollSeconds:    flags.VMBootPollSeconds,
		VMBootTimeoutMinutes: flags.VMBootTimeoutMinutes,
		KeystoreDir:          flags.KeystoreDir,
	})
	if err != nil {
		log.Fatal(err)
	}
}
