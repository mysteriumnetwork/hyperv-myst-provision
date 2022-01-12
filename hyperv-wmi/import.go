package hyperv_wmi

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/pkg/errors"

	"github.com/mysteriumnetwork/hyperv-node/provisioner"
)

type ImportOptions struct {
	Force                bool
	VMBootPollSeconds    int64
	VMBootTimeoutMinutes int64
	KeystoreDir          string
	PreferEthernet       bool
}

func (m *Manager) ImportVM(opt ImportOptions, pf provisioner.ProgressFunc) error {
	log.Println("ImportVM", opt)

	if err := m.CreateExternalNetworkSwitchIfNotExistsAndAssign(opt.PreferEthernet); err != nil {
		return errors.Wrap(err, "CreateExternalNetworkSwitchIfNotExistsAndAssign")
	}

	if opt.Force {
		if err := m.RemoveVM(); err != nil {
			return errors.Wrap(err, "RemoveVM")
		}
	}
	vm, err := m.GetVM()
	if err != nil && !errors.Is(err, wmi.ErrNotFound) {
		return errors.Wrap(err, "GetVM")
	}
	if vm == nil || errors.Is(err, wmi.ErrNotFound) {

		vhdFilePath, err := provisioner.DownloadRelease(provisioner.DownloadOptions{false}, pf)
		if err != nil {
			return err
		}

		err = m.CreateVM(vhdFilePath)
		if err != nil {
			return errors.Wrap(err, "CreateVM")
		}
	}

	if err = m.EnableGuestServices(); err != nil {
		return errors.Wrap(err, "EnableGuestServices")
	}
	if err = m.StartVM(); err != nil {
		return errors.Wrap(err, "StartVM")
	}
	if err = m.StartGuestFileService(); err != nil {
		return errors.Wrap(err, "StartGuestFileService")
	}

	err = m.WaitUntilBooted(
		time.Duration(opt.VMBootPollSeconds)*time.Second,
		time.Duration(opt.VMBootTimeoutMinutes)*time.Minute,
	)
	if err != nil {
		return errors.Wrap(err, "WaitUntilBooted")
	}

	keystorePath := opt.KeystoreDir
	if opt.KeystoreDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return errors.Wrap(err, "UserHomeDir")
		}
		keystorePath = fmt.Sprintf(`%s\%s`, homeDir, `.mysterium\keystore`)
	}
	log.Println("keystorePath >", keystorePath)
	err = filepath.Walk(keystorePath, func(path string, info fs.FileInfo, _ error) error {
		if info.IsDir() {
			return nil
		}
		return m.CopyFile(path, "/root/.mysterium/keystore/")
	})
	if err != nil {
		return errors.Wrap(err, "Walk")
	}

	return nil
}
