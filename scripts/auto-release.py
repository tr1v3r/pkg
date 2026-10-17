#!/usr/bin/env python3
"""Plan module patch releases; publish only tested dev -> master merge commits."""

import argparse
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parent.parent
PREFIX = "github.com/tr1v3r/pkg/"
VERSION = re.compile(r"v(0|1)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$")


def output(root, *arguments):
    return subprocess.check_output(arguments, cwd=root, text=True).strip()


def git(root, *arguments):
    return output(root, "git", *arguments)


def ancestor(root, before, after):
    result = subprocess.run(["git", "merge-base", "--is-ancestor", before, after], cwd=root, check=False)
    if result.returncode not in (0, 1):
        raise RuntimeError("cannot determine release ancestry")
    return result.returncode == 0


def merged_from_dev(pulls, repository, head):
    return any(
        pull.get("merged_at") and pull.get("merge_commit_sha") == head
        and pull.get("base", {}).get("ref") == "master"
        and pull.get("head", {}).get("ref") == "dev"
        and (pull.get("head", {}).get("repo") or {}).get("full_name") == repository
        for pull in pulls
    )


def manifests(root):
    directories = output(root, str(root / "scripts/modules.sh"), "--list").splitlines()
    result = {}
    for directory in directories:
        data = json.loads(output(root, "go", "mod", "edit", "-json", str(root / directory / "go.mod")))
        result[directory] = [item["Path"].removeprefix(PREFIX)
                             for item in (data.get("Require") or []) if item["Path"].startswith(PREFIX)]
    return result


def build_plan(root, head, published, changed_paths=None):
    """Use cumulative code changes since each module's latest stable tag."""
    head = git(root, "rev-parse", f"{head}^{{commit}}")
    dependencies = manifests(root)
    tags = set(git(root, "tag", "--list").splitlines())
    eligible = set(dependencies)
    if changed_paths is not None:
        eligible = set()
        for path in changed_paths:
            if path.endswith(".go") or Path(path).name in {"go.mod", "go.sum"}:
                owner = max((name for name in dependencies if path.startswith(name + "/")), key=len, default="")
                if owner:
                    eligible.add(owner)
    # Bootstrap unpublished dependencies only when a changed module needs them.
    def include_unpublished(module, seen):
        if module in seen:
            return
        seen.add(module)
        for dependency in dependencies[module]:
            if dependency not in dependencies:
                raise RuntimeError(f"unknown internal module {dependency}")
            if not any(tag.startswith(dependency + "/") and VERSION.fullmatch(tag[len(dependency) + 1:]) for tag in tags):
                eligible.add(dependency)
                include_unpublished(dependency, seen)
    for module in list(eligible):
        include_unpublished(module, set())
    selected = {}
    for module in sorted(eligible):
        candidates = []
        for tag in tags:
            if tag.startswith(module + "/"):
                match = VERSION.fullmatch(tag.removeprefix(module + "/"))
                if match:
                    candidates.append((tuple(map(int, match.groups())), tag))
        if not candidates:
            selected[module] = {"module": module, "tag": module + "/v0.1.0", "head": head}
            continue
        version, previous = max(candidates)
        revision = git(root, "rev-parse", previous + "^{commit}")
        if not ancestor(root, revision, head):
            if ancestor(root, head, revision):
                continue  # A newer tested run already released this module.
            raise RuntimeError(f"{previous} is on a divergent history; refusing to move backwards")
        if revision == head:
            if previous not in published:
                selected[module] = {"module": module, "tag": previous, "head": head}
            continue
        changed = git(root, "diff", "--name-only", "--no-renames", revision, head, "--", module).splitlines()
        # Attribute files to the deepest module, in case nested modules are added.
        changed = [path for path in changed if max(
            (name for name in dependencies if path.startswith(name + "/")), key=len, default=""
        ) == module]
        if any(path.endswith(".go") or Path(path).name in {"go.mod", "go.sum"} for path in changed):
            major, minor, patch = version
            selected[module] = {"module": module, "tag": f"{module}/v{major}.{minor}.{patch + 1}", "head": head}

    ordered, visiting, visited = [], set(), set()

    def visit(module):
        if module in visiting:
            raise RuntimeError(f"internal dependency cycle at {module}")
        if module in visited:
            return
        visiting.add(module)
        for dependency in sorted(dependencies[module]):
            if dependency not in dependencies:
                raise RuntimeError(f"{module} requires unknown internal module {dependency}")
            if dependency in selected:
                visit(dependency)
        visiting.remove(module)
        visited.add(module)
        ordered.append(selected[module])

    for module in sorted(selected):
        visit(module)
    if ordered:
        boundary = "v0.3.0"
        if boundary not in tags or boundary not in published:
            revision = git(root, "rev-parse", boundary + "^{commit}") if boundary in tags else head
            if not ancestor(root, revision, head):
                raise RuntimeError("root migration boundary is not an ancestor of the release")
            ordered.insert(0, {"module": ".", "tag": boundary, "head": revision})
    return ordered


def validate(root, entry, go_roots):
    for go_root in go_roots:
        environment = dict(os.environ, GOROOT=go_root, GOTOOLCHAIN="local")
        environment["PATH"] = str(Path(go_root) / "bin") + os.pathsep + environment["PATH"]
        print(f"Validate {entry['tag']} with {go_root}", flush=True)
        subprocess.run([str(root / "scripts/check-release.sh"), entry["tag"]],
                       cwd=root, env=environment, check=True)


def publish(root, entry, repository, published, go_roots):
    tag = entry["tag"]
    if tag in published:
        return
    # No tag or release mutation occurs until every supported toolchain passes.
    validate(root, entry, go_roots)
    existing = git(root, "tag", "--list", tag)
    if existing:
        if git(root, "rev-parse", tag + "^{commit}") != entry["head"]:
            raise RuntimeError(f"refusing to move existing tag {tag}")
    else:
        subprocess.run(["git", "tag", tag, entry["head"]], cwd=root, check=True)
        subprocess.run(["git", "push", "origin", "refs/tags/" + tag], cwd=root, check=True)
    with tempfile.NamedTemporaryFile(mode="w", suffix=".md") as notes:
        notes.write(f"Release `{tag}` from tested dev → master merge `{entry['head']}`.\n\n"
                    "This module is versioned independently. See MODULES.md for migration and dependency rules.\n")
        notes.flush()
        subprocess.run([
            "gh", "release", "create", tag, "--repo", repository, "--verify-tag",
            "--latest=" + ("true" if entry["module"] == "." else "false"),
            "--title", tag, "--notes-file", notes.name,
        ], cwd=root, check=True)
    published.add(tag)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--publish", action="store_true", help="publish after validating the dev merge and all toolchains")
    parser.add_argument("--check-merge", action="store_true", help="report whether this is a dev merge")
    parser.add_argument("--head", default=os.environ.get("GITHUB_SHA", "HEAD"))
    arguments = parser.parse_args()
    head = git(ROOT, "rev-parse", arguments.head + "^{commit}")
    published = set()
    repository = os.environ.get("GITHUB_REPOSITORY", "tr1v3r/pkg")
    changed_paths = None
    if not arguments.check_merge:
        if head != git(ROOT, "rev-parse", "HEAD"):
            raise RuntimeError("check out the exact release commit before planning")
        if git(ROOT, "status", "--porcelain"):
            raise RuntimeError("release planning requires a clean committed checkout")
    if arguments.publish or arguments.check_merge:
        if os.environ.get("GITHUB_EVENT_NAME") != "push" or os.environ.get("GITHUB_REF") != "refs/heads/master":
            raise RuntimeError("automatic publication requires a master push event")
        pulls = json.loads(output(ROOT, "gh", "api", f"repos/{repository}/commits/{head}/pulls"))
        eligible = merged_from_dev(pulls, repository, head)
        if arguments.check_merge:
            if os.environ.get("GITHUB_OUTPUT"):
                with open(os.environ["GITHUB_OUTPUT"], "a") as destination:
                    destination.write("eligible=" + str(eligible).lower() + "\n")
            print("Eligible dev merge:", eligible)
            return
        if not eligible:
            print("No merged dev → master PR for this commit; skipping automatic release.")
            return
        changed_paths = []
        for pull in pulls:
            if merged_from_dev([pull], repository, head):
                changed_paths.extend(output(ROOT, "gh", "api", "--paginate",
                    f"repos/{repository}/pulls/{pull['number']}/files", "--jq",
                    '.[] | .filename, (.previous_filename // empty)').splitlines())
        subprocess.run(["git", "fetch", "origin", "--tags"], cwd=ROOT, check=True)
        # Reconcile tags/releases after acquiring the workflow's publication lock.
        released = output(ROOT, "gh", "api", "--paginate", f"repos/{repository}/releases",
                          "--jq", '.[] | select(.draft == false) | .tag_name')
        published = set(released.splitlines())
    plan = build_plan(ROOT, head, published, changed_paths)
    print(json.dumps(plan, indent=2), flush=True)
    if arguments.publish:
        go_roots = [os.environ[f"RELEASE_GO{version}"] for version in (124, 125, 126, 127)]
        for entry in plan:
            publish(ROOT, entry, repository, published, go_roots)
    if os.environ.get("GITHUB_STEP_SUMMARY"):
        with open(os.environ["GITHUB_STEP_SUMMARY"], "a") as summary:
            summary.write("### Module release plan\n\n")
            summary.write("\n".join(f"- `{entry['tag']}`" for entry in plan) or "No module code changes to release.")
            summary.write("\n")


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, subprocess.CalledProcessError) as error:
        raise SystemExit(str(error)) from error
