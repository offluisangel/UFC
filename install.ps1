[CmdletBinding()]
param(
  [string]$Version,
  [string]$BaseUrl = 'https://github.com/offluisangel/UFC/releases/download',
  [string]$Destination = (Join-Path $env:LOCALAPPDATA 'Programs\UFC'),
  [switch]$Force,
  [switch]$AddToPath
)
$ErrorActionPreference = 'Stop'
$tag = $Version
if (-not $tag) {
  $release = Invoke-RestMethod -Uri 'https://api.github.com/repos/offluisangel/UFC/releases/latest'
  $tag = $release.tag_name
}
if (-not $tag) { throw 'Could not resolve latest release tag' }
$exe = Join-Path $Destination 'ufc.exe'
if ((Test-Path $exe) -and -not $Force) { throw "$exe already exists; rerun with -Force to replace it" }
$arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
if ($arch -ne [System.Runtime.InteropServices.Architecture]::X64) { throw "Unsupported architecture: $arch" }
$archive = 'ufc-windows-amd64.zip'
$tmp = Join-Path ([IO.Path]::GetTempPath()) ([guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
  $url = "$($BaseUrl.TrimEnd('/'))/$tag"
  Invoke-WebRequest "$url/$archive" -OutFile (Join-Path $tmp $archive)
  Invoke-WebRequest "$url/SHA256SUMS" -OutFile (Join-Path $tmp 'SHA256SUMS')
  $line = Get-Content (Join-Path $tmp 'SHA256SUMS') | Where-Object { $_ -match "  $([regex]::Escape($archive))$" } | Select-Object -First 1
  if (-not $line) { throw "Checksum missing for $archive" }
  $expected = ($line -split '\s+')[0]
  $actual = (Get-FileHash (Join-Path $tmp $archive) -Algorithm SHA256).Hash
  if ($actual -ne $expected) { throw 'Release checksum verification failed' }
  Expand-Archive (Join-Path $tmp $archive) -DestinationPath $tmp
  if (-not (Test-Path (Join-Path $tmp 'ufc.exe'))) { throw 'Archive does not contain ufc.exe' }
  New-Item -ItemType Directory -Force -Path $Destination | Out-Null
  Copy-Item (Join-Path $tmp 'ufc.exe') $exe -Force:$Force
  if ($AddToPath) {
    $current = [Environment]::GetEnvironmentVariable('Path', 'User')
    if (($current -split ';') -notcontains $Destination) {
      [Environment]::SetEnvironmentVariable('Path', (($current.TrimEnd(';') + ';' + $Destination).Trim(';')), 'User')
    }
    if (($env:Path -split ';') -notcontains $Destination) { $env:Path = "$Destination;$env:Path" }
  }
  Write-Output "Installed $exe"
  if (-not $AddToPath) { Write-Output 'Run ufc.exe from this directory, or rerun with -AddToPath to update your user PATH.' }
} finally { Remove-Item -LiteralPath $tmp -Recurse -Force }
