package filestore

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
)

type FileStore struct {
}

func NewFileStore() *FileStore {
	return &FileStore{}
}

func (s *FileStore) MkdirAll(name string, perm os.FileMode) error {
	return os.MkdirAll(name, perm)
}

func (s *FileStore) Store(dir string, files []domain.TaskFile) error {
	err := s.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create task data dir: %w", err)
	}

	for _, file := range files {
		if err := s.storeFile(dir, file); err != nil {
			return err
		}
	}

	return nil
}

func (s *FileStore) storeFile(dir string, file domain.TaskFile) error {
	dstPath := filepath.Join(dir, file.Name)

	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", dstPath, err)
	}
	defer func() {
		_ = dst.Close()
	}()

	if _, err := io.Copy(dst, file.File); err != nil {
		return fmt.Errorf("failed to write file %s: %w", dstPath, err)
	}

	return nil
}
