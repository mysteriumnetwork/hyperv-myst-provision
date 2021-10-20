package common

import (
	"errors"
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

func WorkingDirs(wd string) (Dirs, error) {
	var workDir string

	if wd == "" {
		workDir = fmt.Sprintf(`%s\%s`, os.TempDir(), "mysterium")
	} else {
		workDir = wd
	}

	err := os.MkdirAll(workDir, os.ModePerm)
	if err != nil {
		return Dirs{}, err
	}

	hyperVExportDirName := "dist"
	exportDir := fmt.Sprintf(`%s\%s`, workDir, hyperVExportDirName)
	err = os.MkdirAll(exportDir, os.ModePerm)
	if err != nil {
		return Dirs{}, err
	}

	return Dirs{
		WorkDir:      workDir,
		VMExport:     exportDir,
		VMExportName: hyperVExportDirName,
	}, nil
}

type Dirs struct {
	Home, WorkDir, VMExport, VMExportName string
}

func Exists(name string) (bool, error) {
	_, err := os.Stat(name)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}
