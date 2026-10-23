package fileservice

import (
	"context"
	"fmt"
	"hash/fnv"
	"io"
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

// WithDirectoryLock serializes cooperating operations using a bounded set of
// stable lock files in the parent directory. Collisions defer unrelated work.
func (s *FileService) WithDirectoryLock(
	ctx context.Context,
	directory string,
	process func(context.Context) error,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	parent := filepath.Dir(filepath.Clean(directory))
	if err := os.MkdirAll(parent, 0755); err != nil {
		return err
	}
	const lockShards = 256
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(filepath.Base(filepath.Clean(directory))))
	shard := hash.Sum32() % lockShards
	lockPath := filepath.Join(parent, fmt.Sprintf(".files-%02x.lock", shard))
	return filelock.WithLock(lockPath, func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return process(ctx)
	})
}

func (s *FileService) MkdirAll(path string) error {
	return os.MkdirAll(path, 0755)
}

func (s *FileService) Create(path string) (io.WriteCloser, error) {
	return os.Create(path)
}
