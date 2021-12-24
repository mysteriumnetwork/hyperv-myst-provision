package provisioner

import (
	"fmt"
	"log"
	"os"

	"github.com/artdarek/go-unzip/pkg/unzip"
	"github.com/mysteriumnetwork/myst-launcher/utils"
)

func DownloadRelease() (string, error) {
	releases, err := gitReleases("mysteriumnetwork", "hyperv-myst-provision", 1)
	if err != nil {
		return "", err
	}
	assetName, assetUrl := "", ""
	for _, a := range releases[0].Assets {
		if a.Name == "alpine-vm-disk.zip" {
			assetName, assetUrl = a.Name, a.BrowserDownloadUrl
			break
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	fmt.Println(assetName, assetUrl, dir)

	err = utils.DownloadFile(dir+`\`+assetName, assetUrl, func(progress int) {
		if progress%10 == 0 {
			log.Println(fmt.Sprintf("%s - %d%%", assetName, progress))
		}
	})
	if err != nil {
		return "", err
	}

	uz := unzip.New()
	files, err := uz.Extract(dir+`\`+assetName, dir+`\unzip`)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(files)
	return dir + `\unzip\` + files[0], nil
}
