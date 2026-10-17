#!/usr/bin/env python3
"""Offline regression tests for module selection, distribution and root migration."""

import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest
import zipfile

ROOT = Path(__file__).resolve().parent.parent


class ModuleToolingTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="pkg-tooling-test-")
        self.root = Path(self.temporary.name) / "repo"
        self.root.mkdir()
        self.addCleanup(self.temporary.cleanup)
        self.addCleanup(self.make_writable)
        (self.root / "scripts").mkdir()
        for name in ["modules.sh", "module-proxy.py", "standalone.sh", "standalone.py", "check-workspace.py", "check-release.sh"]:
            shutil.copy2(ROOT / "scripts" / name, self.root / "scripts" / name)
        (self.root / "LICENSE").write_text("Fixture license\n")
        (self.root / "go.mod").write_text("module github.com/tr1v3r/pkg\n\ngo 1.24.0\n")
        (self.root / "go.work").write_text(
            "go 1.24.0\n\nuse (\n .\n ./a\n ./b\n)\n\n"
            "replace github.com/tr1v3r/pkg/a v0.1.0 => ./a\n"
        )
        for module in ["a", "b"]:
            directory = self.root / module
            directory.mkdir()
            requirement = "\nrequire github.com/tr1v3r/pkg/a v0.1.0\n" if module == "b" else ""
            (directory / "go.mod").write_text(
                f"module github.com/tr1v3r/pkg/{module}\n\ngo 1.24.0\n" + requirement
            )
            code = "const Value = 42\n" if module == "a" else (
                'import "github.com/tr1v3r/pkg/a"\nconst Value = a.Value\n'
            )
            (directory / f"{module}.go").write_text(f"package {module}\n" + code)
            (directory / f"{module}_test.go").write_text(
                f'package {module}\nimport "testing"\n'
                'func TestValue(t *testing.T) { if Value != 42 { t.Fatal(Value) } }\n'
            )
        self.environment = dict(
            os.environ, GOENV="off", GOWORK="off", GOPROXY="off", GOSUMDB="off",
            GOTOOLCHAIN=os.environ.get("GOTOOLCHAIN", "local"),
            GOMODCACHE=str(Path(self.temporary.name) / "cache"),
            GOCACHE=os.environ.get("GOCACHE", str(Path(tempfile.gettempdir()) / "pkg-tooling-build")),
            PYTHONDONTWRITEBYTECODE="1",
        )
        self.environment.pop("MODULE", None)

    def make_writable(self):
        for directory, _, _ in os.walk(self.temporary.name):
            os.chmod(directory, 0o700)

    def run_command(self, arguments, directory=None, **environment):
        return subprocess.run(
            arguments, cwd=directory or self.root, env=dict(self.environment, **environment),
            text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, check=False,
        )

    def succeeds(self, arguments, directory=None, **environment):
        result = self.run_command(arguments, directory, **environment)
        self.assertEqual(result.returncode, 0, result.stdout)
        return result.stdout

    def test_discovery_and_selection(self):
        for excluded in ["vendor/ignored", "a/testdata/ignored", "docs/experiments/ignored"]:
            directory = self.root / excluded
            directory.mkdir(parents=True)
            (directory / "go.mod").write_text("module example.com/ignored\n")
        self.assertEqual(self.succeeds(["./scripts/modules.sh", "--list"]), "a\nb\n")
        output = self.succeeds(["./scripts/modules.sh", "pwd"], MODULE="./b")
        self.assertNotIn("module: a", output)
        self.assertIn("module: b", output)
        invalid = self.run_command(["./scripts/modules.sh", "pwd"], MODULE="../outside")
        self.assertEqual(invalid.returncode, 2)

    def test_runner_stops_on_failure(self):
        result = self.run_command(["./scripts/modules.sh", "sh", "-c", "exit 9"])
        self.assertEqual(result.returncode, 9, result.stdout)
        self.assertNotIn("module: b", result.stdout)

    def test_workspace_requires_all_modules_and_version_mappings(self):
        self.succeeds(["python3", "scripts/check-workspace.py"])
        workspace = self.root / "go.work"
        original = workspace.read_text()
        workspace.write_text(original.replace(" ./b\n", ""))
        self.assertNotEqual(self.run_command(["python3", "scripts/check-workspace.py"]).returncode, 0)
        workspace.write_text(original.split("replace")[0])
        result = self.run_command(["python3", "scripts/check-workspace.py"])
        self.assertIn("missing versioned workspace replacement", result.stdout)

    def test_standalone_checks_do_not_modify_checkout_or_leak_sums(self):
        self.succeeds([
            "./scripts/standalone.sh", "sh", "-c", "go test ./... && touch scratch"
        ])
        for module in ["a", "b"]:
            self.assertFalse((self.root / module / "scratch").exists())
            self.assertFalse((self.root / module / "go.sum").exists())
        self.succeeds(["./scripts/standalone.sh", "--check-tidy"])

    def test_tidy_check_detects_drift_and_write_mode_fixes_it(self):
        manifest = self.root / "a/go.mod"
        manifest.write_text(manifest.read_text() + "\nrequire github.com/tr1v3r/pkg/b v0.1.0\n")
        result = self.run_command(["./scripts/standalone.sh", "--check-tidy"], MODULE="a")
        self.assertNotEqual(result.returncode, 0, result.stdout)
        self.assertIn("require", manifest.read_text())
        self.succeeds(["./scripts/standalone.sh", "--tidy"], MODULE="a")
        self.assertNotIn("require", manifest.read_text())
        self.assertFalse((self.root / "a/go.sum").exists())

    def test_recording_published_sums_and_preserving_them_in_checkout_mode(self):
        proxy = self.root / "proxy"
        self.succeeds(["python3", "scripts/module-proxy.py", str(proxy)])
        self.succeeds(["./scripts/standalone.sh", "--published", "--tidy"],
                      MODULE="b", GOPROXY=proxy.as_uri())
        original = (self.root / "b/go.sum").read_text()
        self.assertIn("github.com/tr1v3r/pkg/a v0.1.0 h1:", original)
        self.succeeds(["./scripts/standalone.sh", "--tidy"], MODULE="b")
        self.assertEqual((self.root / "b/go.sum").read_text(), original)
        self.succeeds(["./scripts/standalone.sh", "--check-tidy"], MODULE="b")

    def test_archives_include_license_and_exclude_nested_modules(self):
        destination = self.root / "proxy"
        self.succeeds(["python3", "scripts/module-proxy.py", str(destination)])
        with zipfile.ZipFile(destination / "github.com/tr1v3r/pkg/a/@v/v0.1.0.zip") as archive:
            self.assertEqual(archive.read("github.com/tr1v3r/pkg/a@v0.1.0/LICENSE"), b"Fixture license\n")
        with zipfile.ZipFile(destination / "github.com/tr1v3r/pkg/@v/v0.3.0.zip") as archive:
            self.assertFalse(any("/a/a.go" in name for name in archive.namelist()))

    def test_release_gate_rejects_bad_tags_and_unpublished_dependencies(self):
        for tag in ["unknown/v0.1.0", "a/v2.0.0", "a/v0.1.0;touch-injected"]:
            result = self.run_command(["./scripts/check-release.sh", tag])
            self.assertNotEqual(result.returncode, 0, result.stdout)
        self.succeeds(["./scripts/check-release.sh", "v0.3.0"])
        self.succeeds(["./scripts/check-release.sh", "a/v0.1.0"])
        result = self.run_command(["./scripts/check-release.sh", "b/v0.1.0"])
        self.assertNotEqual(result.returncode, 0, result.stdout)
        self.assertIn("published", result.stdout)
        self.assertNotIn("checkout proxy", result.stdout)

    def test_old_root_conflict_is_resolved_by_boundary_upgrade(self):
        proxy = self.root / "proxy"
        self.succeeds(["python3", "scripts/module-proxy.py", str(proxy)])
        target = proxy / "github.com/tr1v3r/pkg/@v"
        manifest = (self.root / "go.mod").read_bytes()
        (target / "v0.2.1.mod").write_bytes(manifest)
        (target / "v0.2.1.info").write_text(json.dumps({
            "Version": "v0.2.1", "Time": "2026-09-06T00:00:00Z"
        }))
        with zipfile.ZipFile(target / "v0.2.1.zip", "w") as archive:
            archive.writestr("github.com/tr1v3r/pkg@v0.2.1/go.mod", manifest)
            archive.writestr("github.com/tr1v3r/pkg@v0.2.1/a/a.go", "package a\n")
        consumer = self.root / "consumer"
        consumer.mkdir()
        (consumer / "go.mod").write_text(
            "module example.com/consumer\n\ngo 1.24.0\n\nrequire (\n"
            " github.com/tr1v3r/pkg v0.2.1\n github.com/tr1v3r/pkg/a v0.1.0\n)\n"
        )
        (consumer / "main.go").write_text(
            'package main\nimport _ "github.com/tr1v3r/pkg/a"\nfunc main() {}\n'
        )
        environment = {"GOPROXY": proxy.as_uri()}
        result = self.run_command(["go", "test", "-mod=mod", "./..."], consumer, **environment)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("ambiguous import", result.stdout)
        self.succeeds(["go", "get", "github.com/tr1v3r/pkg@v0.3.0",
                       "github.com/tr1v3r/pkg/a@v0.1.0"], consumer, **environment)
        self.succeeds(["go", "mod", "tidy"], consumer, **environment)
        self.succeeds(["go", "test", "./..."], consumer, **environment)


if __name__ == "__main__":
    unittest.main(verbosity=2)
