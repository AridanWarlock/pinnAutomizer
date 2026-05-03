package filewriter

import (
	"fmt"
	"io"
	"os"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

type FileWriter struct {
}

func NewFileWriter() *FileWriter {
	return &FileWriter{}
}

func (w *FileWriter) Write(dst string, writer io.Writer) error {
	file, err := os.Open(dst)
	if err != nil {
		return errs.ErrNotExists
	}
	defer file.Close()

	_, err = io.Copy(writer, file)
	if err != nil {
		return fmt.Errorf("copy file: %w", err)
	}
	return nil
}
