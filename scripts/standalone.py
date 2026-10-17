#!/usr/bin/env python3
"""Check modules without workspace replacements or unpublished checksum leakage."""

import difflib
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile

ROOT = Path(__file__).resolve().parent.parent
PREFIX = "github.com/tr1v3r/pkg/"


def run(command, directory, environment):
    subprocess.run(command, cwd=directory, env=environment, check=True)


def external_sums(content):
    return "".join(line for line in content.splitlines(keepends=True) if not line.startswith(PREFIX))


def main():
    arguments = sys.argv[1:]
    published = arguments[:1] == ["--published"]
    if published:
        arguments.pop(0)
    tidy = arguments == ["--tidy"]
    check_tidy = arguments == ["--check-tidy"]
    if not arguments:
        sys.exit("usage: standalone.sh [--published] command [args...] | --tidy | --check-tidy")
    modules = subprocess.check_output(
        [str(ROOT / "scripts/modules.sh"), "--list"], text=True
    ).splitlines()
    selected = os.environ.get("MODULE", "").removeprefix("./")
    if selected:
        if selected not in modules:
            sys.exit(f"unknown module: {selected}")
        modules = [selected]

    with tempfile.TemporaryDirectory(prefix="pkg-standalone-") as temporary:
        temporary = Path(temporary)
        environment = dict(os.environ, GOWORK="off")
        if not published:
            spec = importlib.util.spec_from_file_location("module_proxy", ROOT / "scripts/module-proxy.py")
            proxy = importlib.util.module_from_spec(spec)
            spec.loader.exec_module(proxy)
            proxy.build_proxy(temporary / "proxy")
            environment.update(
                GOMODCACHE=str(temporary / "modcache"),
                GOPROXY=(temporary / "proxy").as_uri() + "," + os.environ.get(
                    "GOPROXY", "https://proxy.golang.org,direct"
                ),
                GOPRIVATE="",
                GONOPROXY="",
                GONOSUMDB="github.com/tr1v3r/pkg,github.com/tr1v3r/pkg/*" + (
                    "," + os.environ["GONOSUMDB"] if os.environ.get("GONOSUMDB") else ""
                ),
            )
        # Go makes downloaded modules read-only; restore permissions for cleanup.
        try:
            for module in modules:
                source = ROOT / module
                directory = temporary / "source" / module
                shutil.copytree(source, directory, ignore=shutil.ignore_patterns(
                    "*.out", "*.test", "coverage.html", "__pycache__"
                ))
                original_sums = (source / "go.sum").read_text() if (source / "go.sum").exists() else ""
                if not published and original_sums:
                    # Existing real release sums must never be compared to synthetic bytes.
                    (directory / "go.sum").write_text(external_sums(original_sums))
                print(f"standalone ({'published' if published else 'checkout proxy'}): {module}", flush=True)
                if tidy or check_tidy:
                    run(["go", "mod", "tidy"], directory, environment)
                    sums = directory / "go.sum"
                    content = sums.read_text() if sums.exists() else ""
                    if check_tidy:
                        for name, before, after in (
                            ("go.mod", (source / "go.mod").read_text(), (directory / "go.mod").read_text()),
                            ("go.sum", external_sums(original_sums), external_sums(content)),
                        ):
                            if before != after:
                                sys.stderr.writelines(difflib.unified_diff(
                                    before.splitlines(keepends=True), after.splitlines(keepends=True),
                                    fromfile=f"{module}/{name}", tofile=f"tidy/{module}/{name}"
                                ))
                                sys.exit(1)
                    else:
                        shutil.copyfile(directory / "go.mod", source / "go.mod")
                        if not published:
                            # Preserve real internal sums already recorded by the maintainer.
                            internal = [line for line in original_sums.splitlines(keepends=True)
                                        if line.startswith(PREFIX)]
                            content = "".join(sorted(external_sums(content).splitlines(keepends=True) + internal))
                        if content:
                            (source / "go.sum").write_text(content)
                        else:
                            (source / "go.sum").unlink(missing_ok=True)
                else:
                    # Internal versions are deliberately absent from committed sums until
                    # release. Download them only into this throwaway source/cache.
                    manifest = json.loads(subprocess.check_output(
                        ["go", "mod", "edit", "-json"], cwd=directory, env=environment, text=True
                    ))
                    dependencies = [item["Path"] for item in (manifest.get("Require") or [])
                                    if item["Path"].startswith(PREFIX)]
                    if dependencies:
                        run(["go", "mod", "download", *dependencies], directory, environment)
                    run(arguments, directory, environment)
        finally:
            for directory, _, _ in os.walk(temporary):
                os.chmod(directory, 0o700)


if __name__ == "__main__":
    try:
        main()
    except subprocess.CalledProcessError as error:
        sys.exit(error.returncode)
