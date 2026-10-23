package fileservice

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AridanWarlock/pinnAutomizer/pkg/filelock"
)

type FileService struct {
}

func NewFileService() *FileService {
	return &FileService{}
}

func (s *FileService) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (s *FileService) RemoveAll(dir string) error {
	return os.RemoveAll(dir)
}

// WithTaskFiles serializes writes and cleanup using one of 256 stable lock files
// in the shared data volume. Hash collisions only defer unrelated operations.
func (s *FileService) WithTaskFiles(
	ctx context.Context,
	dataPath string,
	process func(context.Context) error,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	parent := filepath.Dir(filepath.Clean(dataPath))
	if err := os.MkdirAll(parent, 0755); err != nil {
		return err
	}
	shard := sha256.Sum256([]byte(filepath.Base(filepath.Clean(dataPath))))
	lockPath := filepath.Join(parent, fmt.Sprintf(".files-%02x.lock", shard[0]))
	return filelock.WithLock(lockPath, func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return process(ctx)
	})
}
