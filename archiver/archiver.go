package archiver

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func Archive(path, destination string) error {
	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer out.Close()

	w := zip.NewWriter(out)
	defer w.Close()

	walker := func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		file, err := os.Open(p)
		if err != nil {
			return err
		}
		defer file.Close()

		relPath := strings.TrimPrefix(p, filepath.Dir(path))
		f, err := w.Create(relPath)
		if err != nil {
			return err
		}

		fmt.Printf("Zipping: %#v\n", p)
		_, err = io.Copy(f, file)
		if err != nil {
			return err
		}

		return nil
	}

	return filepath.Walk(path, walker)
}
