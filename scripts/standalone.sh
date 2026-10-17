#!/bin/sh
# See standalone.py for isolated checkout and published-dependency validation.
set -eu
root=$(CDPATH='' cd -- "$(dirname "$0")/.." && pwd)
exec python3 "$root/scripts/standalone.py" "$@"
