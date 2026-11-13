#!/bin/sh
set -eu
ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
TMP=$(mktemp -d)
cleanup() {
  if [ -n "${SERVER_PID:-}" ]; then kill "$SERVER_PID" 2>/dev/null || true; wait "$SERVER_PID" 2>/dev/null || true; fi
  rm -rf "$TMP"
}
trap cleanup EXIT HUP INT TERM
mkdir -p "$TMP/home" "$TMP/archive" "$TMP/v1" "$TMP/foreign"
PREFIX="$TMP/home/bin with spaces"
make_release() {
  printf '#!/bin/sh\nprintf "ufc %s\\n"\n' "$1" > "$TMP/archive/ufc"
  chmod +x "$TMP/archive/ufc"
  tar -czf "$TMP/v1/ufc-linux-amd64.tar.gz" -C "$TMP/archive" ufc
  (cd "$TMP/v1" && sha256sum ufc-linux-amd64.tar.gz > SHA256SUMS)
}
make_release fixture-v1
python3 - "$TMP" <<'PY' >"$TMP/server.log" 2>&1 &
import http.server
import os
import sys
os.chdir(sys.argv[1])
server = http.server.ThreadingHTTPServer(('127.0.0.1', 0), http.server.SimpleHTTPRequestHandler)
with open('port', 'w') as port:
    port.write(str(server.server_address[1]))
server.serve_forever()
PY
SERVER_PID=$!
i=0
while [ ! -s "$TMP/port" ]; do
  i=$((i+1)); [ "$i" -lt 100 ] || { cat "$TMP/server.log" >&2; exit 1; }
  sleep .05
done
read -r PORT < "$TMP/port" || [ -n "$PORT" ]
BASE="http://127.0.0.1:$PORT"
run() {
  LC_ALL=C HOME="$TMP/home" PATH="$PREFIX:/usr/bin:/bin" UFC_TEST_ARCH="${UFC_TEST_ARCH:-amd64}" \
    sh "$ROOT/install.sh" --base-url "$BASE" --version v1 --prefix "$PREFIX" "$@"
}
fail() { cat "$TMP/out" "$TMP/err" >&2; echo "$1" >&2; exit 1; }

# A real command lookup, from a foreign cwd, with no Go or global installation.
cd "$TMP/foreign"
run >"$TMP/out" 2>"$TMP/err" || fail 'install failed'
EXPECTED=$(printf 'ufc-linux-amd64.tar.gz: OK\nInstalled %s/ufc' "$PREFIX")
[ "$(cat "$TMP/out")" = "$EXPECTED" ] && [ ! -s "$TMP/err" ] || fail 'unexpected success output'
PATH="$PREFIX:/usr/bin:/bin" ufc --version >"$TMP/out" 2>"$TMP/err"
[ "$(cat "$TMP/out")" = 'ufc fixture-v1' ] && [ ! -s "$TMP/err" ] || fail 'foreign cwd command lookup failed'
cp "$PREFIX/ufc" "$TMP/original"

if run >"$TMP/out" 2>"$TMP/err"; then fail 'existing binary unexpectedly overwritten'; fi
[ ! -s "$TMP/out" ] && [ "$(cat "$TMP/err")" = "$PREFIX/ufc already exists; rerun with --force to replace it" ] || fail 'wrong existing-file error'
cmp "$PREFIX/ufc" "$TMP/original"

make_release fixture-v2
run --force >"$TMP/out" 2>"$TMP/err" || fail 'forced replacement failed'
[ "$("$PREFIX/ufc" --version)" = 'ufc fixture-v2' ] && [ ! -s "$TMP/err" ] || fail 'forced replacement content mismatch'
cp "$PREFIX/ufc" "$TMP/original"
printf '%064d  ufc-linux-amd64.tar.gz\n' 0 > "$TMP/v1/SHA256SUMS"
if run --force >"$TMP/out" 2>"$TMP/err"; then fail 'bad checksum accepted'; fi
grep -q 'Release checksum verification failed' "$TMP/err" || fail 'wrong checksum error'
cmp "$PREFIX/ufc" "$TMP/original"

if UFC_TEST_ARCH=unsupported run --force >"$TMP/out" 2>"$TMP/err"; then fail 'unsupported architecture accepted'; fi
[ ! -s "$TMP/out" ] && [ "$(cat "$TMP/err")" = 'Unsupported architecture: unsupported' ] || fail 'wrong architecture error'
cmp "$PREFIX/ufc" "$TMP/original"

make_release fixture-v2
OTHER="$TMP/home/not in path"
run --prefix "$OTHER" >"$TMP/out" 2>"$TMP/err" || fail 'alternate destination failed'
EXPECTED=$(printf 'ufc-linux-amd64.tar.gz: OK\nInstalled %s/ufc, but %s is not in PATH. Add it to your shell PATH and reopen the shell.' "$OTHER" "$OTHER")
[ "$(cat "$TMP/out")" = "$EXPECTED" ] && [ ! -s "$TMP/err" ] || fail 'missing PATH guidance absent'
[ "$("$OTHER/ufc" --version)" = 'ufc fixture-v2' ] || fail 'alternate destination content mismatch'
printf 'installer fixture tests passed: 6 scenarios\n'
