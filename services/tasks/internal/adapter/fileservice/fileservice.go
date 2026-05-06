package fileservice

import "os"

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
