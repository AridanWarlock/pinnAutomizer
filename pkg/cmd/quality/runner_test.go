package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func goVersion(t *testing.T) string {
	t.Helper()
	data, err := exec.Command("go", "env", "GOVERSION").Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimPrefix(strings.TrimSpace(string(data)), "go")
}

func probe(t *testing.T, source string) (int, string, string) {
	t.Helper()
	t.Setenv("GOWORK", "off")
	root := t.TempDir()
	write(t, filepath.Join(root, "go.mod"), "module example.com/probe\n\ngo "+goVersion(t)+"\n")
	write(t, filepath.Join(root, "probe_test.go"), source)
	var output bytes.Buffer
	r := runner{root: root, packages: []string{"./..."}, output: &output, ctx: context.Background()}
	code, err := r.tests("test", nil)
	if err != nil {
		t.Fatal(err)
	}
	log, err := os.ReadFile(r.logPath("test"))
	if err != nil {
		t.Fatal(err)
	}
	return code, output.String(), string(log)
}

func TestSuccessfulRunIsQuiet(t *testing.T) {
	code, out, log := probe(t, `package probe
import "testing"
func TestPassing(t *testing.T){t.Log("SUCCESS_NOISE")}
`)
	if code != 0 || !strings.Contains(out, "test: OK") || strings.Contains(out, "SUCCESS_NOISE") || !strings.Contains(log, "SUCCESS_NOISE") {
		t.Fatalf("code=%d, output=%q", code, out)
	}
	// No file exists here: a successful report must not even open its log.
	var output bytes.Buffer
	r := runner{root: t.TempDir(), output: &output}
	if _, err := r.report("missing", 0, nil); err != nil {
		t.Fatal(err)
	}
}

func TestOnlyFailedTestsArePrinted(t *testing.T) {
	code, out, log := probe(t, `package probe
import "testing"
func TestPassing(t *testing.T){t.Log("SUCCESS_NOISE")}

func TestFailure(t *testing.T){t.Fatal("EXPECTED_FAILURE")}
`)
	if code == 0 || !strings.Contains(out, "EXPECTED_FAILURE") || strings.Contains(out, "SUCCESS_NOISE") || !strings.Contains(log, "SUCCESS_NOISE") || !strings.Contains(out, "Full log:") {
		t.Fatalf("code=%d, output=%q", code, out)
	}
}

func TestBuildErrorsAreVisible(t *testing.T) {
	code, out, _ := probe(t, `package probe
import "testing"
func TestBuild(t *testing.T){missingName()}
`)
	if code == 0 || !strings.Contains(out, "missingName") {
		t.Fatalf("code=%d, output=%q", code, out)
	}
}

func TestCleanCloneWorkspace(t *testing.T) {
	t.Setenv("GOWORK", "")
	root := t.TempDir()
	version := goVersion(t)
	for _, name := range []string{"one", "two"} {
		write(t, filepath.Join(root, name, "go.mod"), "module example.com/"+name+"\n\ngo "+version+"\n")
		write(t, filepath.Join(root, name, "code.go"), "package "+name+"\n")
	}
	r := runner{root: root, packages: []string{"./one/...", "./two/..."}, ctx: context.Background()}
	env, err := r.environment()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "list", "./one/...", "./two/...")
	cmd.Dir = root
	cmd.Env = env
	data, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list: %v: %s", err, data)
	}
	if len(strings.Fields(string(data))) != 2 {
		t.Fatalf("unexpected packages: %s", data)
	}
}

func TestExplicitWorkspacePreserved(t *testing.T) {
	t.Setenv("GOWORK", "off")
	r := runner{root: t.TempDir()}
	env, err := r.environment()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(env, "\n"), "GOWORK=off") {
		t.Fatal("explicit workspace overwritten")
	}
}

func TestDiagnosticsAreBounded(t *testing.T) {
	root := t.TempDir()
	var output bytes.Buffer
	r := runner{root: root, output: &output}
	var log strings.Builder
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&log, "error %d\n", i)
	}
	write(t, r.logPath("probe"), log.String())
	code, err := r.report("probe", 7, nil)
	if err != nil || code != 7 || strings.Contains(output.String(), "error 99") || !strings.Contains(output.String(), "80 more diagnostic lines") {
		t.Fatalf("code=%d, error=%v, output=%q", code, err, output.String())
	}
}

func TestVersionOrdering(t *testing.T) {
	if !versionLess("1.9.9", "1.26.1") || versionLess("1.26.2", "1.26.1") || versionLess("1.26.1", "1.26.1") {
		t.Fatal("Go versions compared incorrectly")
	}
}

func TestCommandExitCodeAndCancellation(t *testing.T) {
	t.Setenv("GOWORK", "off")
	r := runner{root: t.TempDir(), ctx: context.Background()}
	code, err := r.logged("exit", "/bin/sh", "-c", "exit 7")
	if err != nil || code != 7 {
		t.Fatalf("code=%d, error=%v", code, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r.ctx = ctx
	code, err = r.logged("canceled", "go", "version")
	if code == 0 && err == nil {
		t.Fatal("canceled command succeeded")
	}
}

func TestInstallerChecksumAndAtomicReplacement(t *testing.T) {
	name := fmt.Sprintf("golangci-lint-%s-%s-%s", linterVersion, runtime.GOOS, runtime.GOARCH)
	var archive bytes.Buffer
	compressed := gzip.NewWriter(&archive)
	bundle := tar.NewWriter(compressed)
	payload := []byte("test linter binary")
	if err := bundle.WriteHeader(&tar.Header{Name: name + "/golangci-lint", Mode: 0755, Size: int64(len(payload)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := bundle.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := bundle.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}
	var valid atomic.Bool
	valid.Store(true)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".tar.gz") {
			_, _ = w.Write(archive.Bytes())
			return
		}
		digest := fmt.Sprintf("%x", sha256.Sum256(archive.Bytes()))
		if !valid.Load() {
			digest = strings.Repeat("0", 64)
		}
		fmt.Fprintf(w, "%s  %s.tar.gz\n", digest, name)
	}))
	defer server.Close()
	root := t.TempDir()
	if err := install(context.Background(), root, server.URL+"/"); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, ".tools", "golangci-lint")
	data, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(data, payload) {
		t.Fatalf("installed binary: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil || info.Mode().Perm()&0111 == 0 {
		t.Fatal("binary is not executable")
	}
	valid.Store(false)
	if err := install(context.Background(), root, server.URL+"/"); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("invalid checksum accepted: %v", err)
	}
	data, err = os.ReadFile(target)
	if err != nil || !bytes.Equal(data, payload) {
		t.Fatal("failed installation replaced existing tool")
	}
}

func TestInstallerHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	err := install(context.Background(), t.TempDir(), server.URL+"/")
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("download failure hidden: %v", err)
	}
}

func TestBootstrapWithoutWorkspace(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "pkg", "go.mod"), "module example.com/tools\n\ngo "+goVersion(t)+"\n")
	sourceDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"main.go", "runner.go", "install.go"} {
		source, err := os.Open(filepath.Join(sourceDir, name))
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(root, "pkg", "cmd", "quality", name)
		if err = os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			t.Fatal(err)
		}
		destination, err := os.Create(target)
		if err != nil {
			t.Fatal(err)
		}
		_, copyErr := io.Copy(destination, source)
		closeErr := destination.Close()
		_ = source.Close()
		if copyErr != nil || closeErr != nil {
			t.Fatalf("copy source: %v %v", copyErr, closeErr)
		}
	}
	binary := filepath.Join(root, "quality")
	cmd := exec.Command("go", "build", "-o", binary, "./cmd/quality")
	cmd.Dir = filepath.Join(root, "pkg")
	cmd.Env = append(os.Environ(), "GOWORK=off")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("bootstrap: %v: %s", err, output)
	}
	cmd = exec.Command(binary)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "usage: quality") {
		t.Fatalf("runner executable: %v: %s", err, output)
	}
}

func TestFormatDiffFailsEvenWhenFormatterExitsZero(t *testing.T) {
	t.Setenv("GOWORK", "off")
	root := t.TempDir()
	tool := filepath.Join(root, ".tools", "golangci-lint")
	write(t, tool, "#!/bin/sh\nif [ \"$1\" = version ]; then\n echo 'golangci-lint has version "+linterVersion+"'\nelse\n echo 'diff -u sample.go.orig sample.go'\nfi\n")
	if err := os.Chmod(tool, 0755); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	r := runner{root: root, output: &output, ctx: context.Background()}
	code, err := r.run([]string{"fmt-check"})
	if err != nil || code != 1 || !strings.Contains(output.String(), "sample.go") {
		t.Fatalf("code=%d error=%v output=%q", code, err, output.String())
	}
}
