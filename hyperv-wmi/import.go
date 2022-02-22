package hyperv_wmi

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mysteriumnetwork/hyperv-node/service/util/winutil"
	"github.com/pkg/errors"
)

type ImportOptions struct {
	Force                bool
	VMBootPollSeconds    int64
	VMBootTimeoutMinutes int64
	KeystoreDir          string
	PreferEthernet       bool
	AdapterID            string
}

type VMInfo struct {
	AdapterName  string
	NodeIdentity string
	OS           string
}

func (m *Manager) ImportVM(opt ImportOptions, pf ProgressFunc, vi *VMInfo) error {
	log.Println("ImportVM >", opt)

	if opt.Force {
		if err := m.RemoveVM(); err != nil {
			return errors.Wrap(err, "RemoveVM")
		}
	}

	na := Adapter{}
	if err := m.ModifySwitchSettings(opt.PreferEthernet, opt.AdapterID, &na); err != nil {
		return errors.Wrap(err, "ModifySwitchSettings")
	}
	vi.AdapterName = na.Name

	vhdFilePath, err := m.DownloadRelease(DownloadOptions{false, m.cfg}, pf)
	if err != nil {
		return err
	}
	err = m.CreateVM(vhdFilePath)
	if err != nil {
		return errors.Wrap(err, "CreateVM")
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

	err = m.WaitUntilBoot(
		time.Duration(opt.VMBootPollSeconds)*time.Second,
		time.Duration(opt.VMBootTimeoutMinutes)*time.Minute,
	)
	if err != nil {
		return errors.Wrap(err, "WaitUntilBoot")
	}

	// copy keystore
	keystorePath := opt.KeystoreDir
	if opt.KeystoreDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return errors.Wrap(err, "UserHomeDir")
		}
		keystorePath = fmt.Sprintf(`%s\%s`, homeDir, `.mysterium\keystore`)
	}
	if _, err := os.Stat(keystorePath); os.IsNotExist(err) {
		return errors.Wrap(err, "Keystore not found")
	}

	log.Println("keystorePath >", keystorePath)
	err = filepath.Walk(keystorePath, func(path string, info fs.FileInfo, _ error) error {
		if info.IsDir() {
			return nil
		}
		if info.Name() == "remember.json" {
			file, _ := ioutil.ReadFile(path)
			data := struct {
				Identity struct {
					Address string `json:"address"`
				} `json:"identity"`
			}{}
			_ = json.Unmarshal([]byte(file), &data)
			vi.NodeIdentity = data.Identity.Address
			vi.OS = winutil.GetWindowsVersion()

		}
		return m.CopyFile(path, "/root/.mysterium/keystore/")
	})
	if err != nil {
		return errors.Wrap(err, "Walk")
	}

	//err = m.copyEnvMyst()
	//if err != nil {
	//	return errors.Wrap(err, "copyEnvMyst")
	//}
	err = m.WaitUntilGotIP(
		time.Duration(opt.VMBootPollSeconds)*time.Second,
		time.Duration(opt.VMBootTimeoutMinutes)*time.Minute,
	)
	//if err != nil {
	//	return errors.Wrap(err, "WaitUntilGotIP")
	//}
	err = m.copyEnvMyst()
	if err != nil {
		return errors.Wrap(err, "copyEnvMyst")
	}

	return nil
}

func (m *Manager) copyEnvMyst() error {
	tempDir := os.TempDir()
	envMystPath := filepath.Join(tempDir, ".env.myst")
	log.Println("envMystPath  >", envMystPath)

	txt := []byte("LAUNCHER=vmh-0.0.1/windows\n")
	err := os.WriteFile(envMystPath, txt, 0644)
	if err != nil {
		return errors.Wrap(err, "WriteFile")
	}
	return m.CopyFile(envMystPath, "/")
}
