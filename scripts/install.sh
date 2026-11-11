#!/bin/sh
# Build and install VerGener from this checkout without requiring root access.
set -eu

if [ "$#" -ne 0 ]; then
    printf 'Usage: INSTALL_DIR=/path/to/bin %s\n' "$0" >&2
    exit 2
fi

if ! command -v go >/dev/null 2>&1; then
    printf 'Go 1.27+ is required to build VerGener.\n' >&2
    exit 1
fi

if [ -z "${INSTALL_DIR:-}" ]; then
    : "${HOME:?Set HOME or INSTALL_DIR to choose an installation directory}"
    INSTALL_DIR=$HOME/.local/bin
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)
mkdir -p -- "$INSTALL_DIR"
install_dir=$(CDPATH= cd -- "$INSTALL_DIR" && pwd)
tmp=$(mktemp "$install_dir/.vergener.XXXXXXXX")
trap 'rm -f -- "$tmp"' EXIT HUP INT TERM

(
    cd -- "$repo_dir"
    go build -o "$tmp" ./cmd/vergener
)
chmod 755 "$tmp"
mv -f -- "$tmp" "$install_dir/vergener"
trap - EXIT HUP INT TERM
printf 'Installed VerGener to %s/vergener\n' "$install_dir"
case :"${PATH:-}": in
    *:"$install_dir":*) ;;
    *) printf 'Add %s to your PATH to run vergener.\n' "$install_dir" ;;
esac
