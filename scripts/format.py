#!/usr/bin/env python3
"""Format staged, unstaged and untracked Go files without shell path splitting."""

from pathlib import Path
import subprocess

root = Path(__file__).resolve().parent.parent
paths = set()
for arguments in (
    ["diff", "--name-only", "--diff-filter=ACMR", "-z", "--", "*.go"],
    ["diff", "--cached", "--name-only", "--diff-filter=ACMR", "-z", "--", "*.go"],
    ["ls-files", "--others", "--exclude-standard", "-z", "--", "*.go"],
):
    paths.update(subprocess.check_output(["git", *arguments], cwd=root).decode().strip("\0").split("\0"))
files = sorted(path for path in paths if path and (root / path).is_file())
if files:
    subprocess.run([
        "goimports-reviser", "-project-name", "github.com/tr1v3r/pkg", "-rm-unused", *files
    ], cwd=root, check=True)
else:
    print("No changed Go files to format.")
