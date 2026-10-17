#!/bin/sh
# Run commands in each library module; the root module is only a migration boundary.
set -eu
cd "$(dirname "$0")/.."

modules() {
    find . -type d \( -name .git -o -name vendor -o -name testdata -o -name experiments \) -prune -o \
        -name go.mod -type f ! -path ./go.mod -print | sed 's|^./||; s|/go.mod$||' | sort
}

if [ "${1:-}" = '--list' ]; then
    modules
    exit 0
fi
if [ "$#" -eq 0 ]; then
    printf 'usage: %s --list | command [args...]\n' "$0" >&2
    exit 2
fi

# MODULE optionally selects one module for Makefile targets and CI.
if [ -n "${MODULE:-}" ]; then
    selected=${MODULE#./}
    if ! modules | grep -Fqx -- "$selected"; then
        printf 'unknown module: %s\n' "$MODULE" >&2
        exit 2
    fi
    printf 'module: %s\n' "$selected"
    cd "$selected"
    exec "$@"
fi

modules | while IFS= read -r module; do
    printf 'module: %s\n' "$module"
    (cd "$module" && "$@")
done
