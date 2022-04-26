package vbox

import (
	"encoding/json"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	consts "github.com/mysteriumnetwork/hyperv-node/const"

	"github.com/mysteriumnetwork/hyperv-node/service/daemon/client"
	"github.com/mysteriumnetwork/hyperv-node/service/util/winutil"
	"github.com/mysteriumnetwork/hyperv-node/utils"
	"github.com/pkg/errors"
)

type ImportOptions struct {
	Force                bool
	VMBootPollSeconds    int64
	VMBootTimeoutMinutes int64
	KeystorePath         string
	PreferEthernet       bool
	AdapterID            string
	AdapterName          string
	//UseMinAdapterFlag bool
}

type VMInfo struct {
	AdapterName  string
	NodeIdentity string
	OS           string
}

func (m *Manager) ImportVM(opt ImportOptions, pf ProgressFunc, vi *VMInfo) error {
	log.Println("ImportVM >", opt)

	// if opt.Force {
	if err := m.RemoveVM(); err != nil {
		return errors.Wrap(err, "RemoveVM")
	}
	// }

	aa, _ := m.GetAdapters()
	for _, a := range aa {
		if (a.NetType == 9 && !opt.PreferEthernet) || (a.NetType != 0 && opt.PreferEthernet) {
			opt.AdapterID = a.ID
			opt.AdapterName = a.Name

			vi.AdapterName = a.Name
			break
		}
	}

	vhdFilePath, err := m.DownloadRelease(DownloadOptions{false, m.cfg}, pf)
	if err != nil {
		return err
	}

	err = m.CreateVM(vhdFilePath, opt)
	if err != nil {
		return errors.Wrap(err, "CreateVM")
	}
	m.cfg.KeystorePath = opt.KeystorePath
	m.cfg.Save()

	//if err = m.StartVM(); err != nil {
	//	return errors.Wrap(err, "StartVM")
	//}
	//m.WaitVMReady()
	//m.ImportKeystore(vi)

	return nil
}

func (m *Manager) ImportKeystore(vi *VMInfo) error {

	// copy keystore
	keystorePath := m.cfg.KeystorePath
	if keystorePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return errors.Wrap(err, "UserHomeDir")
		}
		keystorePath = path.Join(homeDir, consts.KeystorePath)
	}
	if _, err := os.Stat(keystorePath); os.IsNotExist(err) {
		return errors.Wrap(err, "Keystore not found")
	}

	ip := m.Kvp["IP_int"].(string)
	err := utils.Retry(5, time.Second, func() error {
		return client.VmAgentGetState(ip)
	})
	if err != nil {
		return err
	}

	log.Println("keystorePath >", keystorePath)
	if vi != nil {
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
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "Walk")
		}
	}

	err = m.CopyFile(keystorePath)
	log.Println("CopyFile", err)

	return nil
}
