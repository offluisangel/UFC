#!/bin/sh
# Check the single-publisher contract, optionally including built release files.
set -eu
ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
python3 - "$ROOT" "${1:-}" <<'PY'
import hashlib
import pathlib
import sys
import tarfile
import zipfile

root = pathlib.Path(sys.argv[1])
release = (root / '.github/workflows/release.yml').read_text()
windows = (root / '.github/workflows/release-windows.yml').read_text()
assert 'dist/*' in release, 'publish the complete dist directory, including installers'
assert 'sh scripts/package-release.sh' in release, 'use the tested release packaging script'
assert release.count('softprops/action-gh-release@') == 1, 'one release publisher required'
assert 'softprops/action-gh-release@' not in windows, 'Windows must not overwrite shared release assets'
assert 'workflow_call:' in windows, 'Windows validation must be called by the release workflow'
assert 'needs: windows' in release, 'publish only after Windows checks'
if sys.argv[2]:
    dist = pathlib.Path(sys.argv[2])
    assets = {'ufc-linux-amd64.tar.gz', 'ufc-linux-arm64.tar.gz',
              'ufc-windows-amd64.zip', 'install.sh', 'install.ps1'}
    assert {p.name for p in dist.iterdir()} == assets | {'SHA256SUMS'}, 'unexpected or missing assets'
    lines = (dist / 'SHA256SUMS').read_text().splitlines()
    checksums = dict(line.split('  ', 1)[::-1] for line in lines)
    assert len(lines) == len(assets) and set(checksums) == assets, 'one checksum per asset required'
    for name in assets:
        assert hashlib.sha256((dist / name).read_bytes()).hexdigest() == checksums[name], name
    for arch in ('amd64', 'arm64'):
        with tarfile.open(dist / f'ufc-linux-{arch}.tar.gz') as archive:
            assert archive.getnames() == ['ufc'], 'Linux archive must contain only ufc'
            assert archive.getmember('ufc').mode & 0o111, 'Linux executable mode missing'
    with zipfile.ZipFile(dist / 'ufc-windows-amd64.zip') as archive:
        assert archive.namelist() == ['ufc.exe'], 'portable Windows ZIP contract changed'
    for name in ('install.sh', 'install.ps1'):
        assert (dist / name).read_bytes() == (root / name).read_bytes(), name
print('release contract checks passed')
PY
