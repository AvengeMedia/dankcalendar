#!/bin/sh
# Run from an unpacked release archive with its platform-specific binary.
set -eu

if [ "$#" -ne 1 ] || [ ! -f "$1" ]; then
    echo "Usage: $0 /path/to/dcal-<os>-<arch>" >&2
    exit 2
fi
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
prefix=${PREFIX:-/usr/local}
staging=${DESTDIR:-}

sh "$script_dir/install-assets.sh"
mkdir -p "$staging$prefix/bin"
install -m 755 "$1" "$staging$prefix/bin/dcal"
echo "Installed dcal and calendar file associations under $staging$prefix"
