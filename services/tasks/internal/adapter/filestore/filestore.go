package filestore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
)

type FileService interface {
	WithDirectoryLock(
		ctx context.Context,
		directory string,
		process func(context.Context) error,
	) error
	MkdirAll(path string) error
	Create(path string) (io.WriteCloser, error)
}

type FileStore struct {
	fileService FileService
}

func NewFileStore(fileService FileService) *FileStore {
	return &FileStore{
		fileService: fileService,
	}
}

func (s *FileStore) MkdirAll(name string) error {
	return s.fileService.MkdirAll(name)
}

func (s *FileStore) WithDirectoryLock(
	ctx context.Context,
	directory string,
	process func(context.Context) error,
) error {
	return s.fileService.WithDirectoryLock(ctx, directory, process)
}

func (s *FileStore) Store(dir string, files []domain.TaskFile) error {
	err := s.MkdirAll(dir)
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

func (s *FileStore) storeFile(dir string, file domain.TaskFile) (err error) {
	dstPath := filepath.Join(dir, file.Name)

	dst, err := s.fileService.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", dstPath, err)
	}
	defer func() {
		err = errors.Join(err, dst.Close())
	}()

	if _, err := io.Copy(dst, file.File); err != nil {
		return fmt.Errorf("failed to write file %s: %w", dstPath, err)
	}

	return nil
}
