#!/usr/bin/env python3
"""Build a temporary Go file proxy for pre-release standalone checks.

Only internal modules are served. External dependencies still use the caller's
normal proxy. These artifacts validate this checkout, not a public release.
"""

import json
from pathlib import Path
import subprocess
import sys
import zipfile


def build_proxy(destination):
    root = Path(__file__).resolve().parent.parent
    modules = subprocess.check_output(
        [str(root / "scripts/modules.sh"), "--list"], text=True
    ).splitlines()
    versions = {"github.com/tr1v3r/pkg": {"v0.3.0"}}
    for directory in modules:
        versions["github.com/tr1v3r/pkg/" + directory] = {"v0.1.0"}
    for directory in modules:
        manifest = json.loads(subprocess.check_output(
            ["go", "mod", "edit", "-json", str(root / directory / "go.mod")], text=True
        ))
        for dependency in (manifest.get("Require") or []):
            if dependency["Path"] in versions:
                versions[dependency["Path"]].add(dependency["Version"])
    for directory in [".", *modules]:
        source = root / directory
        module = (source / "go.mod").read_text().splitlines()[0].split()[1]
        # Map all pinned internal versions to this checkout, only in a private proxy.
        target = destination / module / "@v"
        target.mkdir(parents=True, exist_ok=True)
        (target / "list").write_text("\n".join(sorted(versions[module])) + "\n")
        for version in sorted(versions[module]):
            (target / f"{version}.mod").write_bytes((source / "go.mod").read_bytes())
            (target / f"{version}.info").write_text(json.dumps({
                "Version": version,
                "Time": "2026-10-17T00:00:00Z",
            }) + "\n")
            with zipfile.ZipFile(target / f"{version}.zip", "w", zipfile.ZIP_DEFLATED) as archive:
                for path in sorted(source.rglob("*")):
                    relative = path.relative_to(source)
                    if destination == path or destination in path.parents:
                        continue
                    if path.is_symlink() or not path.is_file():
                        continue
                    if any(part.startswith(".") or part in {"vendor", "__pycache__"} for part in relative.parts):
                        continue
                    if path.suffix in {".out", ".test"} or path.name == "coverage.html":
                        continue
                    # A module archive never contains nested modules.
                    if any((parent / "go.mod").is_file()
                           for parent in path.parents if parent != source and source in parent.parents):
                        continue
                    archive.write(path, f"{module}@{version}/{relative.as_posix()}")
                if not (source / "LICENSE").exists():
                    archive.write(root / "LICENSE", f"{module}@{version}/LICENSE")


if __name__ == "__main__":
    if len(sys.argv) != 2:
        sys.exit("usage: module-proxy.py OUTPUT_DIRECTORY")
    build_proxy(Path(sys.argv[1]).resolve())
