"""Verify quiet reporting against real passing, failing, and unbuildable Go tests."""
import contextlib
import io
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import quality
import subprocess


class QualityTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.go_version = subprocess.check_output(["go", "env", "GOVERSION"], text=True).strip().removeprefix("go")

    def run_probe(self, source):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "go.mod").write_text(f"module example.com/probe\n\ngo {self.go_version}\n")
            (root / "probe_test.go").write_text(source)
            output = io.StringIO()
            with patch.multiple(quality, ROOT=root, LOGS=root / "out", PACKAGES=["./..."]):
                with patch.dict(os.environ, {"GOWORK": "off"}):
                    with contextlib.redirect_stdout(output):
                        code = quality.tests([])
                log = (root / "out/test.log").read_text()
            return code, output.getvalue(), log

    def test_success_does_not_scan_or_print_log(self):
        with patch.object(quality, "test_failures", side_effect=AssertionError("successful log scanned")):
            code, output, log = self.run_probe('''package probe
import "testing"
func TestPassing(t *testing.T) { t.Log("SUCCESS_NOISE") }
''')
        self.assertEqual(code, 0)
        self.assertIn("test: OK", output)
        self.assertNotIn("SUCCESS_NOISE", output)
        self.assertIn("SUCCESS_NOISE", log)

    def test_only_failed_test_output_is_printed(self):
        code, output, log = self.run_probe('''package probe
import "testing"
func TestPassing(t *testing.T) { t.Log("SUCCESS_NOISE") }
func TestFailure(t *testing.T) { t.Fatal("EXPECTED_FAILURE") }
''')
        self.assertNotEqual(code, 0)
        self.assertIn("EXPECTED_FAILURE", output)
        self.assertNotIn("SUCCESS_NOISE", output)
        self.assertIn("SUCCESS_NOISE", log)
        self.assertIn("Full log:", output)

    def test_compiler_errors_are_not_hidden(self):
        code, output, _ = self.run_probe('''package probe
import "testing"
func TestBuild(t *testing.T) { missingName() }
''')
        self.assertNotEqual(code, 0)
        self.assertIn("missingName", output)

    def test_workspace_is_created_for_a_clean_clone(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for name in ("one", "two"):
                module = root / name
                module.mkdir()
                (module / "go.mod").write_text(f"module example.com/{name}\n\ngo {self.go_version}\n")
                (module / "code.go").write_text(f"package {name}\n")
            with patch.multiple(quality, ROOT=root, LOGS=root / "out", PACKAGES=["./one/...", "./two/..."]):
                with patch.dict(os.environ):
                    os.environ.pop("GOWORK", None)
                    env = quality.check_env()
                result = subprocess.run(["go", "list", "./one/...", "./two/..."], cwd=root,
                                        env=env, capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(len(result.stdout.splitlines()), 2)

    def test_diagnostics_are_bounded(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            log = root / "failure.log"
            log.write_text("".join(f"error {number}\n" for number in range(100)))
            output = io.StringIO()
            with patch.object(quality, "ROOT", root), contextlib.redirect_stdout(output):
                code = quality.report("probe", 1, log)
            self.assertEqual(code, 1)
            self.assertNotIn("error 99", output.getvalue())
            self.assertIn("80 more diagnostic lines", output.getvalue())


if __name__ == "__main__":
    unittest.main()
