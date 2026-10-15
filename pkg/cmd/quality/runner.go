package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const maxLines = 20

var packages = []string{"./pkg/...", "./services/auth/...", "./services/tasks/...", "./services/solver/...", "./services/gateway/..."}

type runner struct {
	root     string
	packages []string
	output   io.Writer
	ctx      context.Context
}

func (r runner) logPath(name string) string {
	return filepath.Join(r.root, "out", "quality", name+".log")
}

func (r runner) environment() ([]string, error) {
	env := os.Environ()
	if os.Getenv("GOWORK") != "" {
		return env, nil
	}
	if _, err := os.Stat(filepath.Join(r.root, "go.work")); err == nil {
		return env, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	var modules []string
	var version string
	for _, pattern := range r.packages {
		module := filepath.Join(r.root, strings.TrimSuffix(strings.TrimPrefix(pattern, "./"), "/..."))
		resolved, err := filepath.EvalSymlinks(module)
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(filepath.Join(resolved, "go.mod"))
		if err != nil {
			return nil, err
		}
		found := false
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Fields(line)
			if len(fields) == 2 && fields[0] == "go" {
				found = true
				if versionLess(version, fields[1]) {
					version = fields[1]
				}
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("missing Go version in %s/go.mod", resolved)
		}
		modules = append(modules, resolved)
	}
	work := filepath.Join(r.root, "out", "quality", "go.work")
	if err := os.MkdirAll(filepath.Dir(work), 0755); err != nil {
		return nil, err
	}
	var content strings.Builder
	fmt.Fprintf(&content, "go %s\n\nuse (\n", version)
	for _, module := range modules {
		fmt.Fprintf(&content, "\t%s\n", strconv.Quote(module))
	}
	content.WriteString(")\n")
	if err := os.WriteFile(work, []byte(content.String()), 0644); err != nil {
		return nil, err
	}
	return append(env, "GOWORK="+work), nil
}

func versionLess(a, b string) bool {
	left, right := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		var x, y int
		if i < len(left) {
			x, _ = strconv.Atoi(left[i])
		}
		if i < len(right) {
			y, _ = strconv.Atoi(right[i])
		}
		if x != y {
			return x < y
		}
	}
	return false
}

func (r runner) logged(name string, args ...string) (int, error) {
	path := r.logPath(name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return 1, err
	}
	log, err := os.Create(path)
	if err != nil {
		return 1, err
	}
	defer log.Close()
	env, err := r.environment()
	if err != nil {
		return 1, err
	}
	cmd := exec.CommandContext(r.ctx, args[0], args[1:]...)
	cmd.Dir = r.root
	cmd.Env = env
	cmd.Stdout = log
	cmd.Stderr = log
	err = cmd.Run()
	if err == nil {
		return 0, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		code := exit.ExitCode()
		if code < 1 {
			code = 1
		}
		return code, nil
	}
	return 1, err
}

// scanLog streams the file; memory use does not grow with successful test output.
func scanLog(path string, visit func(string)) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			visit(strings.TrimSuffix(line, "\n"))
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

type diagnostics struct {
	lines []string
	total int
}

func (d *diagnostics) add(line string) {
	for _, part := range strings.Split(strings.TrimRight(line, "\r\n"), "\n") {
		d.total++
		if len(d.lines) < maxLines {
			d.lines = append(d.lines, part)
		}
	}
}

type testEvent struct {
	Action  string
	Package string
	Test    string
	Output  string
}

type testKey struct {
	Package string
	Test    string
}

func failedTests(path string) (diagnostics, error) {
	failed := map[testKey]bool{}
	var d diagnostics
	err := scanLog(path, func(line string) {
		var e testEvent
		if json.Unmarshal([]byte(line), &e) == nil && (e.Action == "fail" || e.Action == "build-fail") {
			failed[testKey{e.Package, e.Test}] = true
		}
	})
	if err != nil {
		return d, err
	}
	err = scanLog(path, func(line string) {
		var e testEvent
		if json.Unmarshal([]byte(line), &e) != nil {
			d.add(line)
			return
		}
		if e.Output != "" && failed[testKey{e.Package, e.Test}] {
			d.add(e.Output)
		}
	})
	return d, err
}

func (r runner) report(name string, code int, d *diagnostics) (int, error) {
	if code == 0 {
		fmt.Fprintf(r.output, "%s: OK\n", name)
		return 0, nil
	}
	if d == nil {
		d = &diagnostics{}
		if err := scanLog(r.logPath(name), d.add); err != nil {
			return code, err
		}
	}
	fmt.Fprintf(r.output, "%s: FAILED\n", name)
	for _, line := range d.lines {
		fmt.Fprintln(r.output, line)
	}
	if d.total > len(d.lines) {
		fmt.Fprintf(r.output, "... %d more diagnostic lines\n", d.total-len(d.lines))
	}
	fmt.Fprintf(r.output, "Full log: out/quality/%s.log\n", name)
	return code, nil
}

func (r runner) tests(name string, args []string) (int, error) {
	if len(args) == 0 {
		args = r.packages
	}
	code, err := r.logged(name, append([]string{"go", "test", "-json"}, args...)...)
	if err != nil {
		return code, err
	}
	var d *diagnostics
	if code != 0 {
		found, e := failedTests(r.logPath(name))
		if e != nil {
			return code, e
		}
		d = &found
	}
	code, err = r.report(name, code, d)

	return code, err
}

func (r runner) tool() (string, error) {
	path := filepath.Join(r.root, ".tools", "golangci-lint")
	data, err := exec.CommandContext(r.ctx, path, "version").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("run make tools-install: %w", err)
	}
	pattern := regexp.MustCompile(`version (?:v)?` + regexp.QuoteMeta(linterVersion) + `(?:\s|$)`)
	if !pattern.Match(data) {
		return "", fmt.Errorf("expected golangci-lint %s; run make tools-install", linterVersion)
	}
	return path, nil
}

func (r runner) run(args []string) (int, error) {
	if len(args) == 0 {
		return 1, errors.New("usage: quality install|fmt|fmt-check|lint|test|self-test|check [go test arguments]")
	}
	action := args[0]
	switch action {
	case "install":
		if err := install(r.ctx, r.root, releaseBase); err != nil {
			return 1, err
		}
		fmt.Fprintf(r.output, "tools: golangci-lint %s installed\n", linterVersion)
		return 0, nil
	case "test":
		return r.tests("test", args[1:])
	case "self-test":
		return r.tests("runner-tests", []string{"./pkg/cmd/quality/..."})
	case "check":
		for _, check := range []string{"fmt-check", "lint", "test"} {
			code, err := r.run([]string{check})
			if code != 0 || err != nil {
				return code, err
			}
		}
		return 0, nil
	case "fmt", "fmt-check", "lint":
		tool, err := r.tool()
		if err != nil {
			return 1, err
		}
		command := []string{tool, "fmt", "--config", filepath.Join(r.root, ".golangci.yml")}
		if action == "lint" {
			command[1] = "run"
		}
		if action == "fmt-check" {
			command = append(command, "--diff")
		}
		code, err := r.logged(action, append(command, r.packages...)...)
		if err != nil {
			return code, err
		}
		if action == "fmt-check" {
			var d diagnostics
			err = scanLog(r.logPath(action), func(line string) {
				if strings.HasPrefix(line, "diff ") && strings.Contains(line, ".orig ") {
					d.add(strings.SplitN(line, ".orig ", 2)[1])
				}
			})
			if err != nil {
				return code, err
			}
			if d.total > 0 {
				return r.report(action, 1, &d)
			}
		}
		return r.report(action, code, nil)
	default:
		return 1, fmt.Errorf("unknown command: %s", action)
	}
}
