# Install UFC

Release installers are available after a maintainer publishes a `v*` tag. They install the `ufc` command independently of the current working directory. The manual `ufc-sync` development command is not distributed.

## Linux

Download and inspect the script before running it:

```sh
curl -fLO https://github.com/offluisangel/UFC/releases/latest/download/install.sh
less install.sh
sh install.sh
ufc tui
```

The default destination is `~/.local/bin`. If it is not in `PATH`, the installer prints guidance and does not modify shell configuration. For a system-wide install, use `sudo sh install.sh --system` to install in `/usr/local/bin`.

## Windows

```powershell
Invoke-WebRequest https://github.com/offluisangel/UFC/releases/latest/download/install.ps1 -OutFile install.ps1
Get-Content .\install.ps1
.\install.ps1 -AddToPath
ufc tui
```

The destination is the current user's local Programs directory. `-AddToPath` persistently adds that directory to the user PATH and updates the current PowerShell process. Without it, run `ufc.exe` from its installation directory.

## Updates and removal

Rerun the installer with `--force` on Linux or `-Force` on Windows to replace an existing executable. Without that explicit option, an existing binary is left untouched. To uninstall, remove the installed binary and its now-empty directory; remove the PATH entry manually if you opted into it.

Both installers download the platform archive and `SHA256SUMS` from the same release tag and verify the checksum before extraction. This protects against accidental corruption; it is not an independently authenticated signature. Linux archives are provided for amd64 and arm64. Windows retains the portable `ufc-windows-amd64.zip` release.

For explicit data-location behavior, see [Data paths](data-paths.md).

## Release checks (maintainers)

One release workflow builds all three archives, includes both installer scripts, and publishes a single `SHA256SUMS` covering all five downloads. Windows validation is a reusable workflow; it does not publish competing assets. Publishing waits for Windows application and installer tests.

Before publishing a tag, check the package locally (Go, Python 3, tar and sha256sum required):

```sh
output=$(mktemp -d)
UFC_VERSION=v0.0.0-test sh scripts/package-release.sh "$output"
sh scripts/test-release-assets.sh "$output"
sh scripts/test-install.sh
```

This builds and tests local files without publishing or installing into your user directories. The Linux installer test uses a temporary HTTP fixture; Windows installer tests run separately with `./scripts/test-install.ps1` in PowerShell on Windows. Linux syntax checks or cross-compilation do not verify Windows installer execution.

## Developer alternative

With [Go 1.27 or later](https://go.dev/doc/install), install from source:

```sh
go install github.com/offluisangel/UFC/cmd/ufc@latest
```
