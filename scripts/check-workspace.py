#!/usr/bin/env python3
"""Reject modules omitted from go.work and local replacements in published manifests."""

import json
from pathlib import Path
import subprocess

root = Path(__file__).resolve().parent.parent
modules = subprocess.check_output([str(root / 'scripts/modules.sh'), '--list'], text=True).splitlines()
workspace = json.loads(subprocess.check_output(['go', 'work', 'edit', '-json', str(root / 'go.work')], cwd=root, text=True))
actual = {(root / entry['DiskPath']).resolve() for entry in workspace['Use']}
expected = {root, *(root / module for module in modules)}
if actual != expected:
    raise SystemExit(f'workspace membership mismatch: missing={expected - actual}, extra={actual - expected}')
pins = set()
for module in ['.', *modules]:
    manifest = json.loads(subprocess.check_output(
        ['go', 'mod', 'edit', '-json', str(root / module / 'go.mod')], text=True
    ))
    if manifest.get('Replace'):
        raise SystemExit(f'{module}: published manifests must not contain replacements')
    expected_path = 'github.com/tr1v3r/pkg' + ('' if module == '.' else '/' + module)
    if manifest['Module']['Path'] != expected_path:
        raise SystemExit(f'{module}: incorrect module path')
    if module == '.' and (manifest.get('Require') or list(root.glob('*.go'))):
        raise SystemExit('root must remain an empty migration boundary')
    for dependency in (manifest.get('Require') or []):
        if dependency['Path'].startswith('github.com/tr1v3r/pkg/'):
            pins.add((dependency['Path'], dependency['Version']))
replacements = {}
for replacement in (workspace.get('Replace') or []):
    old, new = replacement['Old'], replacement['New']
    replacements[(old['Path'], old.get('Version', ''))] = new
for path, version in pins:
    replacement = replacements.get((path, version))
    target = root / path.removeprefix('github.com/tr1v3r/pkg/')
    if not replacement or replacement.get('Version') or (root / replacement['Path']).resolve() != target:
        raise SystemExit(f'missing versioned workspace replacement: {path}@{version} => {target}')
print(f'Workspace contains all {len(modules)} library modules and the root migration boundary.')
