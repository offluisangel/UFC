#!/bin/sh
# Build the complete release locally; publishing belongs to the workflow.
set -eu
ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
OUT=${1:-"$ROOT/dist"}
mkdir -p "$OUT"
OUT=$(CDPATH= cd -- "$OUT" && pwd)
BUILD=$(mktemp -d)
trap 'rm -rf "$BUILD"' EXIT HUP INT TERM
cd "$ROOT"
VERSION=${UFC_VERSION:-dev}
for arch in amd64 arm64; do
  CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath -ldflags="-s -w -X main.version=$VERSION" -o "$BUILD/ufc" ./cmd/ufc
  tar -czf "$OUT/ufc-linux-$arch.tar.gz" -C "$BUILD" ufc
done
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w -X main.version=$VERSION" -o "$BUILD/ufc.exe" ./cmd/ufc
python3 - "$BUILD/ufc.exe" "$OUT/ufc-windows-amd64.zip" <<'PY'
import sys
import zipfile
with zipfile.ZipFile(sys.argv[2], 'w', zipfile.ZIP_DEFLATED) as archive:
    archive.write(sys.argv[1], 'ufc.exe')
PY
cp install.sh install.ps1 "$OUT/"
(cd "$OUT" && sha256sum ufc-linux-amd64.tar.gz ufc-linux-arm64.tar.gz ufc-windows-amd64.zip install.sh install.ps1 > SHA256SUMS)
