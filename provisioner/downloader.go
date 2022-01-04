package provisioner

import (
	"fmt"
	"github.com/artdarek/go-unzip/pkg/unzip"
	"github.com/mysteriumnetwork/myst-launcher/utils"
	"log"
	"os"
)

type ProgressFunc func(progress int)

const (
	assetAlpineImg = "alpine-vm-disk.zip"
)

type DownloadOptions struct {
	Force bool
}

func DownloadRelease(opt DownloadOptions, pf ProgressFunc) (string, error) {
	_, err := os.Stat(assetAlpineImg)
	useExisting := err == nil && !opt.Force

	if !useExisting {
		releases, err := gitReleases("mysteriumnetwork", "hyperv-myst-provision", 1)
		if err != nil {
			return "", err
		}

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

	wd, _ := os.Getwd()
	return wd + `.\vhdx\` + files[0], nil
}
