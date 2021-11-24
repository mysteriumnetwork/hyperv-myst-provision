package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
	"log"

	"github.com/itzg/go-flagsfiller"

	"github.com/mysteriumnetwork/hyperv-node/common"
	"github.com/mysteriumnetwork/hyperv-node/hyperv"
	"github.com/mysteriumnetwork/hyperv-node/hyperv/network"
	"github.com/mysteriumnetwork/hyperv-node/powershell"
)

type flagsSet struct {
	VMName               string `default:"Myst HyperV Alpine" usage:"hyper-v guest VM name"`
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

	m, err := network.NewVMSwitchManager()
	if err != nil {
		log.Fatal(err)
	}

	shell := powershell.New(powershell.OptionDebugPrint)
	hyperV := hyperv.New(flags.VMName, flags.WorkDir, "", shell)

	/*	if flags.Force {
			hyperV.StopVM()
			hyperV.RemoveVM()
		}

		err := hyperV.ImportVM()
		if err != nil {
			log.Fatal(err)
		}*/

	err = m.CreateExternalNetworkSwitchIfNotExistsAndAssign()
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

		hyperV.CopyVMFile(path, "/root/.mysterium/keystore/")
		return nil
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
