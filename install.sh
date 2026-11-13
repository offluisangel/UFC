#!/bin/sh
set -eu
REPO=offluisangel/UFC
BASE_URL=${UFC_RELEASE_BASE_URL:-https://github.com/$REPO/releases/download}
VERSION=${UFC_VERSION:-}
PREFIX=${UFC_PREFIX:-}
FORCE=0
SYSTEM=0
usage() { echo 'Usage: install.sh [--version TAG] [--base-url URL] [--prefix DIR | --system] [--force]'; }
while [ "$#" -gt 0 ]; do
  case "$1" in
    --version) [ "$#" -ge 2 ] || { usage >&2; exit 2; }; VERSION=$2; shift 2 ;;
    --base-url) [ "$#" -ge 2 ] || { usage >&2; exit 2; }; BASE_URL=$2; shift 2 ;;
    --prefix) [ "$#" -ge 2 ] || { usage >&2; exit 2; }; PREFIX=$2; shift 2 ;;
    --system) SYSTEM=1; PREFIX=/usr/local/bin; shift ;;
    --force) FORCE=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unsupported option: $1" >&2; usage >&2; exit 2 ;;
  esac
done
OS=$(uname -s); ARCH=${UFC_TEST_ARCH:-$(uname -m)}
[ "$OS" = Linux ] || { echo "Unsupported operating system: $OS" >&2; exit 1; }
case "$ARCH" in x86_64|amd64) ARCH=amd64 ;; aarch64|arm64) ARCH=arm64 ;; *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;; esac
if [ -z "$VERSION" ]; then
  command -v curl >/dev/null 2>&1 || { echo 'curl is required' >&2; exit 1; }
  VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | awk -F '"' '/"tag_name"/ {print $4; exit}')
fi
[ -n "$VERSION" ] || { echo 'Could not resolve latest release tag' >&2; exit 1; }
if [ -z "$PREFIX" ]; then PREFIX=${HOME:?}/.local/bin; fi
DEST=$PREFIX/ufc
if [ -e "$DEST" ] && [ "$FORCE" -ne 1 ]; then echo "$DEST already exists; rerun with --force to replace it" >&2; exit 1; fi
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT HUP INT TERM
ARCHIVE="ufc-linux-$ARCH.tar.gz"
URL="$BASE_URL/$VERSION"
curl -fsSL "$URL/$ARCHIVE" -o "$TMP/$ARCHIVE" || { echo 'Could not download release archive' >&2; exit 1; }
curl -fsSL "$URL/SHA256SUMS" -o "$TMP/SHA256SUMS" || { echo 'Could not download SHA256SUMS' >&2; exit 1; }
(cd "$TMP" && grep "  $ARCHIVE\$" SHA256SUMS | sha256sum -c -) || { echo 'Release checksum verification failed' >&2; exit 1; }
mkdir "$TMP/unpacked"
tar -xzf "$TMP/$ARCHIVE" -C "$TMP/unpacked" ufc || { echo 'Invalid release archive' >&2; exit 1; }
[ -f "$TMP/unpacked/ufc" ] || { echo 'Archive does not contain ufc' >&2; exit 1; }
mkdir -p "$PREFIX"
if [ "$SYSTEM" -eq 1 ] && [ ! -w "$PREFIX" ]; then echo 'Run with sudo to install to /usr/local/bin' >&2; exit 1; fi
install -m 0755 "$TMP/unpacked/ufc" "$DEST"
case ":${PATH}:" in *":$PREFIX:"*) echo "Installed $DEST" ;; *) echo "Installed $DEST, but $PREFIX is not in PATH. Add it to your shell PATH and reopen the shell." ;; esac
