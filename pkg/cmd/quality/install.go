package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const linterVersion = "2.14.0"
const releaseBase = "https://github.com/golangci/golangci-lint/releases/download/v" + linterVersion + "/"

func download(ctx context.Context, address, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return err
	}
	client := http.Client{Timeout: 60 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download: HTTP %d", response.StatusCode)
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(file, response.Body)
	return errors.Join(copyErr, file.Close())
}

func install(ctx context.Context, root, base string) error {
	if (runtime.GOOS != "darwin" && runtime.GOOS != "linux") || (runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64") {
		return fmt.Errorf("installer supports macOS/Linux on amd64/arm64")
	}
	directory, err := os.MkdirTemp("", "pinn-linter-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	name := fmt.Sprintf("golangci-lint-%s-%s-%s", linterVersion, runtime.GOOS, runtime.GOARCH)
	asset := name + ".tar.gz"
	archive := filepath.Join(directory, asset)
	checksums := filepath.Join(directory, "checksums.txt")
	if err = download(ctx, base+asset, archive); err != nil {
		return err
	}
	if err = download(ctx, base+"golangci-lint-"+linterVersion+"-checksums.txt", checksums); err != nil {
		return err
	}
	expected := ""
	err = scanLog(checksums, func(line string) {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == asset {
			expected = fields[0]
		}
	})
	if err != nil {
		return err
	}
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err = io.Copy(hash, file); err != nil {
		return err
	}
	if expected == "" || hex.EncodeToString(hash.Sum(nil)) != expected {
		return fmt.Errorf("release checksum mismatch")
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	compressed, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer compressed.Close()
	bundle := tar.NewReader(compressed)
	for {
		header, e := bundle.Next()
		if e != nil {
			return fmt.Errorf("linter binary not found in archive: %w", e)
		}
		if header.Name != name+"/golangci-lint" {
			continue
		}
		if header.Typeflag != tar.TypeReg {
			return fmt.Errorf("linter archive member is not a regular file")
		}
		targetDir := filepath.Join(root, ".tools")
		if err = os.MkdirAll(targetDir, 0755); err != nil {
			return err
		}
		binary, err := os.CreateTemp(targetDir, ".golangci-lint-*")
		if err != nil {
			return err
		}
		defer os.Remove(binary.Name())
		_, copyErr := io.Copy(binary, bundle)
		if err = errors.Join(copyErr, binary.Close()); err != nil {
			return err
		}
		if err = os.Chmod(binary.Name(), 0755); err != nil {
			return err
		}
		return os.Rename(binary.Name(), filepath.Join(targetDir, "golangci-lint"))
	}
}
