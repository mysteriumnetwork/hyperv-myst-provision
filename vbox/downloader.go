package vbox

import (
	"fmt"
	"log"
	"os"

	"github.com/artdarek/go-unzip/pkg/unzip"

	"github.com/mysteriumnetwork/hyperv-node/model"
	"github.com/mysteriumnetwork/hyperv-node/provisioner"
	"github.com/mysteriumnetwork/myst-launcher/utils"
)

type ProgressFunc func(progress int)

const (
	assetAlpineImg = "alpine-vm-disk-vdi.zip"
)

type DownloadOptions struct {
	Force bool
	Cfg   *model.Config
}

func (m *Manager) DownloadRelease(opt DownloadOptions, pf ProgressFunc) (string, error) {
	downloadAgain := false

	_, err := os.Stat(assetAlpineImg)
	if err != nil {
		downloadAgain = true
	}

	if m.cfg.ImageVersion == "" {
		downloadAgain = true
	}
	releases, err := provisioner.GitReleases("mysteriumnetwork", "hyperv-myst-provision", 1)
	if err != nil {
		return "", err
	}
	latestVersion := releases[0].TagName
	if latestVersion != m.cfg.ImageVersion {
		downloadAgain = true
	}

	log.Println("downloadAgain>", downloadAgain)
	if downloadAgain {
		assetName, assetUrl := "", ""
		for _, a := range releases[0].Assets {

			if a.Name == assetAlpineImg {
				assetName, assetUrl = a.Name, a.BrowserDownloadUrl
				break
			}
		}

		err = utils.DownloadFile(assetName, assetUrl, func(progress int) {
			if pf != nil {
				pf(progress)
			}

			if progress%10 == 0 {
				log.Println(fmt.Sprintf("%s - %d%%", assetName, progress))
			}
		})
		if err != nil {
			return "", err
		}
	}

	uz := unzip.New()
	files, err := uz.Extract(assetAlpineImg, `.\vhdx`)
	if err != nil {
		fmt.Println(err)
	}
	m.cfg.ImageVersion = latestVersion
	m.cfg.Save()

	wd, _ := os.Getwd()
	return wd + `.\vhdx\` + files[0], nil
}
