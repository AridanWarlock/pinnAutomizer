package zipper

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

type Zipper struct {
}

func NewZipper() *Zipper {
	return &Zipper{}
}

func (z *Zipper) ZipFiles(dir string, writer io.Writer) error {
	zw := zip.NewWriter(writer)
	defer zw.Close()

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		f, err := zw.Create(relPath)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(f, file)
		return err
	})

	return err
}
