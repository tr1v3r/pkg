#!/bin/sh
# Validate a tag without using local workspace modules or the checkout proxy.
set -eu
cd "$(dirname "$0")/.."
tag=${1:?usage: check-release.sh TAG}
version=${tag##*/}
if ! printf '%s\n' "$version" | grep -Eq '^v(0|1)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$'; then
    printf 'invalid v0/v1 module version: %s\n' "$version" >&2
    exit 2
fi
if [ "$tag" = "$version" ]; then
    # The root is an empty compatibility boundary, not an aggregate library.
    test "$(GOWORK=off go list -m all)" = github.com/tr1v3r/pkg
    GOWORK=off go mod tidy -diff
    python3 scripts/check-workspace.py
else
    export MODULE="${tag%/*}"
    ./scripts/modules.sh --list | grep -Fqx -- "$MODULE"
    ./scripts/standalone.sh --published --check-tidy
    ./scripts/standalone.sh --published go test -race -count=1 -tags=integration ./...
fi
