package hyperv_wmi

import (
	"fmt"
	"github.com/mysteriumnetwork/hyperv-node/provisioner"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/pkg/errors"
)

type ImportOptions struct {
	Force                bool
	VMBootPollSeconds    int64
	VMBootTimeoutMinutes int64
	KeystoreDir          string
}

func (m *Manager) ImportVM(opt ImportOptions) error {
	fmt.Println("ImportVM", opt)

	if err := m.CreateExternalNetworkSwitchIfNotExistsAndAssign(); err != nil {
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

		vhdFilePath, err := provisioner.DownloadRelease()
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

	var keystorePath string
	if opt.KeystoreDir != "" {
		keystorePath = opt.KeystoreDir
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return errors.Wrap(err, "UserHomeDir")
		}
		keystorePath = fmt.Sprintf(`%s\%s`, homeDir, `.mysterium\keystore`)
	}
	fmt.Println("keystorePath >", keystorePath)
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
