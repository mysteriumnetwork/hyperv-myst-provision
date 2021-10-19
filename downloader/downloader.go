package downloader

import (
	"fmt"
	"github.com/mysteriumnetwork/hyperv-node/common"
	"github.com/mysteriumnetwork/hyperv-node/powershell"
	"os"
)

type DLoader struct {
	shell *powershell.PowerShell
}

func NewDLoader(shell *powershell.PowerShell) *DLoader {
	return &DLoader{shell: shell}
}

func (d *DLoader) DownloadAndExtract(sourceURL, tempDir string) error {
	targetPath := fmt.Sprintf(`%s\%s`, tempDir, "myst-hyperv.zip")
	err := d.download(sourceURL, targetPath)
	if err != nil {
		return err
	}

	err = d.extract(targetPath, tempDir)
	if err != nil {
		return err
	}

	return os.RemoveAll(targetPath)
}

func (d *DLoader) download(sourceURL, targetPath string) error {
	return common.OutWithIt(d.shell.Execute(
		`(New-Object Net.WebClient).DownloadFile(`,
		common.WrapInQuotes(sourceURL),
		",",
		common.WrapInQuotes(targetPath),
		`)`,
	))
}

func (d *DLoader) extract(source, target string) error {
	return common.OutWithIt(d.shell.Execute(
		`Expand-Archive -LiteralPath`,
		common.WrapInQuotes(source),
		"-DestinationPath",
		common.WrapInQuotes(target),
	))
}
