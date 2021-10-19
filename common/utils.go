package common

import (
	"fmt"
	"github.com/mysteriumnetwork/hyperv-node/powershell"
	"os"
)

func WrapInQuotes(s string) string {
	return `"` + s + `"`
}

func OutWithIt(out powershell.Out, err error) error {
	if err != nil {
		return err
	}
	if out.IsErr() {
		return out.GetError()
	}
	return nil
}

func Path(base string, and ...string) string {
	for _, a := range and {
		base += `\` + a
	}
	return base
}

func OSDirs() (Dirs, error) {
	homePath, err := os.UserHomeDir()
	if err != nil {
		return Dirs{}, err
	}

	MystTempDir := fmt.Sprintf(`%s\%s`, os.TempDir(), "myst")
	err = os.MkdirAll(MystTempDir, os.ModePerm)
	if err != nil {
		return Dirs{}, err
	}

	hyperVExportDirName := "dist"
	exportDir := fmt.Sprintf(`%s\%s`, MystTempDir, hyperVExportDirName)
	err = os.MkdirAll(exportDir, os.ModePerm)
	if err != nil {
		return Dirs{}, err
	}

	return Dirs{
		Home:         homePath,
		Temp:         MystTempDir,
		VMExport:     exportDir,
		VMExportName: hyperVExportDirName,
	}, nil
}

type Dirs struct {
	Home, Temp, VMExport, VMExportName string
}
