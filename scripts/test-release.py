#!/usr/bin/env python3
"""Offline tests for release selection, ordering, idempotency and publication gates."""

import importlib.util
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest
from unittest import mock

ROOT = Path(__file__).resolve().parent.parent
SPEC = importlib.util.spec_from_file_location("auto_release", ROOT / "scripts/auto-release.py")
release = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(release)


class ReleaseTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="pkg-release-test-")
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.git("init", "-b", "master")
        (self.root / "scripts").mkdir()
        shutil.copy2(ROOT / "scripts/modules.sh", self.root / "scripts/modules.sh")
        (self.root / "go.mod").write_text("module github.com/tr1v3r/pkg\n\ngo 1.24.0\n")
        for name in ["a", "b"]:
            (self.root / name).mkdir()
            requirement = "\nrequire github.com/tr1v3r/pkg/a v0.1.0\n" if name == "b" else ""
            (self.root / name / "go.mod").write_text(f"module github.com/tr1v3r/pkg/{name}\n\ngo 1.24.0\n" + requirement)
            (self.root / name / f"{name}.go").write_text(f"package {name}\n")
        self.initial = self.commit("initial modules")
        self.published = {"v0.3.0", "a/v0.1.0", "b/v0.1.0"}

    def git(self, *arguments):
        return subprocess.check_output(["git", *arguments], cwd=self.root, text=True, stderr=subprocess.DEVNULL).strip()

    def commit(self, message):
        self.git("add", ".")
        self.git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.com", "commit", "-m", message)
        return self.git("rev-parse", "HEAD")

    def tag_initial(self):
        for tag in self.published:
            self.git("tag", tag, self.initial)

    def change(self, path, content="// changed\n"):
        source = self.root / path
        source.write_text(source.read_text() + content)
        return self.commit("change " + path)

    def plan(self, head="HEAD", changed_paths=None, published=None):
        return release.build_plan(self.root, head, self.published if published is None else published, changed_paths)

    def test_bootstrap_orders_unpublished_dependencies_and_boundary(self):
        plan = self.plan(changed_paths=["b/b.go"], published=set())
        self.assertEqual([entry["tag"] for entry in plan], ["v0.3.0", "a/v0.1.0", "b/v0.1.0"])

    def test_docs_only_merge_does_not_bootstrap_or_release(self):
        (self.root / "a/README.md").write_text("documentation\n")
        self.commit("documentation")
        self.assertEqual(self.plan(changed_paths=["a/README.md"], published=set()), [])

    def test_patch_only_changed_module_and_ignore_prerelease_tags(self):
        self.tag_initial()
        self.git("tag", "a/v0.1.9", self.initial)
        self.git("tag", "a/v1.0.0-rc.1", self.initial)
        self.change("a/a.go")
        self.assertEqual([item["tag"] for item in self.plan(changed_paths=["a/a.go"])], ["a/v0.1.10"])

    def test_cumulative_diff_includes_earlier_changes_in_rebased_pr(self):
        self.tag_initial()
        self.change("a/a.go")
        (self.root / "README.md").write_text("later documentation\n")
        self.commit("later commit")
        self.assertEqual([item["tag"] for item in self.plan(changed_paths=["a/a.go", "README.md"])], ["a/v0.1.1"])

    def test_go_file_deletion_and_manifest_changes_release_modules(self):
        self.tag_initial()
        (self.root / "a/a.go").unlink()
        self.change("b/go.mod", "\n// dependency metadata\n")
        self.assertEqual([item["tag"] for item in self.plan(changed_paths=["a/a.go", "b/go.mod"])],
                         ["a/v0.1.1", "b/v0.1.1"])

    def test_rerun_reuses_unreleased_tag_and_does_not_bump_again(self):
        self.tag_initial()
        head = self.change("a/a.go")
        self.git("tag", "a/v0.1.1", head)
        self.assertEqual([item["tag"] for item in self.plan(changed_paths=["a/a.go"])], ["a/v0.1.1"])
        self.published.add("a/v0.1.1")
        self.assertEqual(self.plan(changed_paths=["a/a.go"]), [])

    def test_older_run_does_not_release_after_newer_tag(self):
        self.tag_initial()
        older = self.change("a/a.go")
        newer = self.change("a/a.go", "// newer\n")
        self.git("tag", "a/v0.1.1", newer)
        self.assertEqual(self.plan(head=older, changed_paths=["a/a.go"]), [])

    def test_only_merged_same_repository_dev_to_master_is_eligible(self):
        pull = {"merged_at": "2026-10-17T00:00:00Z", "merge_commit_sha": "target",
                "base": {"ref": "master"}, "head": {"ref": "dev", "repo": {"full_name": "tr1v3r/pkg"}}}
        self.assertTrue(release.merged_from_dev([pull], "tr1v3r/pkg", "target"))
        for change in [dict(merged_at=None), dict(merge_commit_sha="other"), dict(base={"ref": "dev"}),
                       dict(head={"ref": "feature", "repo": {"full_name": "tr1v3r/pkg"}}),
                       dict(head={"ref": "dev", "repo": {"full_name": "someone/pkg"}})]:
            self.assertFalse(release.merged_from_dev([dict(pull, **change)], "tr1v3r/pkg", "target"))

    def test_all_four_toolchains_are_validated_with_matching_goroot(self):
        entry = {"module": "a", "tag": "a/v0.1.0", "head": self.initial}
        roots = [f"/fixture/go1.{minor}" for minor in [24, 25, 26, 27]]
        with mock.patch.object(release.subprocess, "run") as run:
            release.validate(self.root, entry, roots)
        self.assertEqual(run.call_count, 4)
        self.assertEqual([call.kwargs["env"]["GOROOT"] for call in run.call_args_list], roots)
        self.assertTrue(all(call.kwargs["env"]["GOTOOLCHAIN"] == "local" for call in run.call_args_list))

    def test_publisher_pushes_exact_tag_and_retries_failed_release_without_new_version(self):
        remote = self.root / "origin.git"
        self.git("init", "--bare", str(remote))
        self.git("remote", "add", "origin", str(remote))
        entry = {"module": "a", "tag": "a/v0.1.0", "head": self.initial}
        published = set()
        original_run = subprocess.run
        calls = []

        def fail_metadata(command, **kwargs):
            if command[0] == "gh":
                calls.append(command)
                raise subprocess.CalledProcessError(1, command)
            return original_run(command, **kwargs)

        with mock.patch.object(release, "validate"):
            with mock.patch.object(release.subprocess, "run", side_effect=fail_metadata):
                with self.assertRaises(subprocess.CalledProcessError):
                    release.publish(self.root, entry, "tr1v3r/pkg", published, ["/fixture/go"])
            self.assertEqual(self.git("--git-dir=" + str(remote), "rev-parse", entry["tag"]), self.initial)
            retried = []

            def complete_metadata(command, **kwargs):
                if command[0] == "gh":
                    retried.append(command)
                    return subprocess.CompletedProcess(command, 0)
                return original_run(command, **kwargs)

            with mock.patch.object(release.subprocess, "run", side_effect=complete_metadata):
                release.publish(self.root, entry, "tr1v3r/pkg", published, ["/fixture/go"])
                self.assertEqual(len(retried), 1)
                self.assertEqual(retried[0][:4], ["gh", "release", "create", entry["tag"]])
        self.assertEqual(self.git("tag", "--list"), entry["tag"])
        self.assertEqual(published, {entry["tag"]})
        self.assertEqual(len(calls), 1)

    def test_failed_validation_cannot_create_tag_or_release(self):
        before = self.git("tag", "--list")
        entry = {"module": "a", "tag": "a/v0.1.0", "head": self.initial}
        with mock.patch.object(release, "validate", side_effect=RuntimeError("test failure")):
            with mock.patch.object(release.subprocess, "run") as mutate:
                with self.assertRaisesRegex(RuntimeError, "test failure"):
                    release.publish(self.root, entry, "tr1v3r/pkg", set(), ["/fixture/go"])
                mutate.assert_not_called()
        self.assertEqual(self.git("tag", "--list"), before)


if __name__ == "__main__":
    unittest.main(verbosity=2)
