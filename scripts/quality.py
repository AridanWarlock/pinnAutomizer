#!/usr/bin/env python3
"""Project checks: preserve complete logs, print only a bounded failure summary."""
import hashlib
import json
import os
from pathlib import Path
import platform
import re
import subprocess
import sys
import tarfile
import tempfile
import urllib.request

ROOT = Path(__file__).resolve().parents[1]
VERSION = "2.14.0"
TOOL = ROOT / ".tools" / "golangci-lint"
LOGS = ROOT / "out" / "quality"
PACKAGES = ["./pkg/...", "./services/auth/...", "./services/tasks/...",
            "./services/solver/...", "./services/gateway/..."]
MAX_LINES = 20


def install():
    system = {"Darwin": "darwin", "Linux": "linux"}.get(platform.system())
    arch = {"x86_64": "amd64", "arm64": "arm64", "aarch64": "arm64"}.get(platform.machine())
    if not system or not arch:
        raise RuntimeError("installer supports macOS/Linux on amd64/arm64")
    asset = f"golangci-lint-{VERSION}-{system}-{arch}.tar.gz"
    base = f"https://github.com/golangci/golangci-lint/releases/download/v{VERSION}/"
    with tempfile.TemporaryDirectory() as directory:
        archive = Path(directory) / asset
        with urllib.request.urlopen(base + asset, timeout=60) as response:
            archive.write_bytes(response.read())
        with urllib.request.urlopen(base + f"golangci-lint-{VERSION}-checksums.txt", timeout=60) as response:
            checksums = response.read().decode().splitlines()
        expected = next(line.split()[0] for line in checksums if line.split()[-1] == asset)
        if hashlib.sha256(archive.read_bytes()).hexdigest() != expected:
            raise RuntimeError("release checksum mismatch")
        with tarfile.open(archive) as bundle:
            member = bundle.getmember(f"golangci-lint-{VERSION}-{system}-{arch}/golangci-lint")
            with bundle.extractfile(member) as binary:
                TOOL.parent.mkdir(exist_ok=True)
                TOOL.write_bytes(binary.read())
        TOOL.chmod(0o755)
    print(f"tools: golangci-lint {VERSION} installed")


def tool():
    if not TOOL.is_file():
        raise RuntimeError("run make tools-install first")
    version = subprocess.check_output([str(TOOL), "version"], text=True)
    if not re.search(rf"version (?:v)?{re.escape(VERSION)}(?:\s|$)", version):
        raise RuntimeError(f"expected golangci-lint {VERSION}; run make tools-install")
    return str(TOOL)


def check_env():
    env = os.environ.copy()
    # A clone need not contain the ignored developer go.work file.
    if not env.get("GOWORK") and not (ROOT / "go.work").is_file():
        LOGS.mkdir(parents=True, exist_ok=True)
        modules = [(ROOT / path.removeprefix("./").removesuffix("/...")).resolve() for path in PACKAGES]
        versions = [re.search(r"^go (.+)$", (module / "go.mod").read_text(), re.MULTILINE).group(1)
                    for module in modules]
        version = max(versions, key=lambda value: tuple(map(int, value.split("."))))
        work = LOGS / "go.work"
        work.write_text(f"go {version}\n\nuse (\n" +
                        "".join(f"\t{json.dumps(str(module))}\n" for module in modules) + ")\n")
        env["GOWORK"] = str(work)
    return env


def run_logged(command, name):
    LOGS.mkdir(parents=True, exist_ok=True)
    log = LOGS / f"{name}.log"
    with log.open("w") as output:
        result = subprocess.run(command, cwd=ROOT, env=check_env(), stdout=output, stderr=subprocess.STDOUT)
    return result.returncode, log


def report(name, code, log, lines=None):
    if code == 0:
        print(f"{name}: OK")
        return 0
    if lines is None:
        lines = log.read_text(errors="replace").splitlines()
    print(f"{name}: FAILED")
    print("\n".join(lines[:MAX_LINES]))
    if len(lines) > MAX_LINES:
        print(f"... {len(lines) - MAX_LINES} more diagnostic lines")
    print(f"Full log: {log.relative_to(ROOT)}")
    return code or 1


def test_failures(log):
    # Successful runs never scan the log. On failure, select only failed tests/packages.
    failed = set()
    with log.open() as events:
        for line in events:
            try:
                event = json.loads(line)
            except json.JSONDecodeError:
                continue
            if event.get("Action") in ("fail", "build-fail"):
                failed.add((event.get("Package"), event.get("Test")))
    diagnostics = []
    with log.open() as events:
        for line in events:
            try:
                event = json.loads(line)
            except json.JSONDecodeError:
                diagnostics.append(line.rstrip())  # Compiler/setup diagnostics on stderr.
                continue
            key = (event.get("Package"), event.get("Test"))
            if key in failed and "Output" in event:
                diagnostics.extend(event["Output"].rstrip().splitlines())
    return diagnostics


def tests(args):
    code, log = run_logged(["go", "test", "-json", *(args or PACKAGES)], "test")
    result = report("test", code, log, test_failures(log) if code else None)
    if not code and not os.environ.get("PINN_TEST_POSTGRES_DSN"):
        print("PostgreSQL integration: not enabled (PINN_TEST_POSTGRES_DSN is unset)")
    return result


def main():
    if len(sys.argv) < 2:
        raise RuntimeError("usage: quality.py install|fmt|fmt-check|lint|test|check [go test arguments]")
    action, args = sys.argv[1], sys.argv[2:]
    if action == "install":
        install()
        return 0
    if action == "self-test":
        code, log = run_logged([sys.executable, "-m", "unittest", "discover", "-s", "scripts", "-p", "test_quality.py"], "runner-tests")
        return report("runner-tests", code, log)
    if action == "test":
        return tests(args)
    if action == "check":
        for check in ("fmt-check", "lint", "self-test", "test"):
            code = subprocess.call([sys.executable, __file__, check], cwd=ROOT)
            if code:
                return code
        return 0
    if action in ("fmt", "fmt-check"):
        command = [tool(), "fmt", "--config", str(ROOT / ".golangci.yml")]
        if action == "fmt-check":
            command += ["--diff"]
        code, log = run_logged(command + PACKAGES, action)
        if action == "fmt-check":
            diff = log.read_text(errors="replace")
            files = re.findall(r"^diff .*\.orig (.+)$", diff, re.MULTILINE)
            if files:
                return report(action, 1, log, files)
        return report(action, code, log)
    if action == "lint":
        code, log = run_logged([tool(), "run", "--config", str(ROOT / ".golangci.yml"), *PACKAGES], action)
        return report(action, code, log)
    raise RuntimeError(f"unknown command: {action}")


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, RuntimeError, StopIteration, subprocess.CalledProcessError) as error:
        print(f"quality: {error}", file=sys.stderr)
        sys.exit(1)
