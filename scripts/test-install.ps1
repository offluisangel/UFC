$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$tmp = Join-Path ([IO.Path]::GetTempPath()) ([guid]::NewGuid().ToString())
$server = $null
try {
  New-Item -ItemType Directory -Path $tmp | Out-Null
  $payload = Join-Path $tmp 'payload'; New-Item -ItemType Directory -Path $payload | Out-Null
  $fixturePath = Join-Path $payload 'ufc.exe'
  [IO.File]::WriteAllText($fixturePath, 'fixture-v1', [Text.UTF8Encoding]::new($false))
  if (-not (Test-Path -LiteralPath $fixturePath -PathType Leaf) -or
      [IO.File]::ReadAllText($fixturePath) -ne 'fixture-v1') {
    throw "Fixture file missing or invalid before archiving: $fixturePath"
  }
  $releaseDir = Join-Path $tmp 'v1'; New-Item -ItemType Directory -Path $releaseDir | Out-Null
  Compress-Archive -LiteralPath $fixturePath -DestinationPath (Join-Path $releaseDir 'ufc-windows-amd64.zip')
  $hash = (Get-FileHash (Join-Path $releaseDir 'ufc-windows-amd64.zip') -Algorithm SHA256).Hash.ToLower()
  Set-Content -LiteralPath (Join-Path $releaseDir 'SHA256SUMS') -Value "$hash  ufc-windows-amd64.zip"
  $listener = [Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback, 0); $listener.Start()
  $port = $listener.LocalEndpoint.Port; $listener.Stop()
  $server = Start-Job -ArgumentList $tmp,$port -ScriptBlock {
    param($directory,$port)
    $listener = [Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback,$port); $listener.Start()
    while ($true) {
      $client=$listener.AcceptTcpClient(); $stream=$client.GetStream(); $reader=[IO.StreamReader]::new($stream)
      $request=$reader.ReadLine(); while ($reader.ReadLine() -ne '') {}
      $name=($request -split ' ')[1].TrimStart('/'); $path=Join-Path $directory $name
      if (Test-Path $path) { $bytes=[IO.File]::ReadAllBytes($path); $status='200 OK' } else { $bytes=[Text.Encoding]::UTF8.GetBytes('missing'); $status='404 Not Found' }
      $header=[Text.Encoding]::ASCII.GetBytes("HTTP/1.1 $status`r`nContent-Length: $($bytes.Length)`r`nConnection: close`r`n`r`n"); $stream.Write($header); $stream.Write($bytes); $client.Close()
    }
  }
  Start-Sleep -Milliseconds 300
  $dest=Join-Path $tmp 'isolated user programs'
  & (Join-Path $root 'install.ps1') -Version v1 -BaseUrl "http://127.0.0.1:$port" -Destination $dest
  if ((Get-Content (Join-Path $dest 'ufc.exe') -Raw) -ne 'fixture-v1') { throw 'install content mismatch' }
  try { & (Join-Path $root 'install.ps1') -Version v1 -BaseUrl "http://127.0.0.1:$port" -Destination $dest; throw 'existing file was overwritten' } catch { if ($_.Exception.Message -eq 'existing file was overwritten') { throw } }
  & (Join-Path $root 'install.ps1') -Version v1 -BaseUrl "http://127.0.0.1:$port" -Destination $dest -Force
  Set-Content -LiteralPath (Join-Path $releaseDir 'SHA256SUMS') -Value ('0' * 64 + '  ufc-windows-amd64.zip')
  try { & (Join-Path $root 'install.ps1') -Version v1 -BaseUrl "http://127.0.0.1:$port" -Destination $dest -Force; throw 'bad checksum accepted' } catch { if ($_.Exception.Message -eq 'bad checksum accepted') { throw } }
  if ((Get-Content (Join-Path $dest 'ufc.exe') -Raw) -ne 'fixture-v1') { throw 'checksum failure changed installed file' }
  Write-Output 'Windows installer fixture tests passed'
} finally {
  if ($server) { Stop-Job $server; Remove-Job $server }
  if (Test-Path $tmp) { Remove-Item -LiteralPath $tmp -Recurse -Force }
}
