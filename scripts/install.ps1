# Install an official lazygit-ai release for the current Windows user.
$ErrorActionPreference = 'Stop'
$repo = 'javiermoya2504/lazygit-ai'
$installDir = $env:LAZYGIT_AI_INSTALL_DIR
if (-not $installDir) { $installDir = Join-Path $env:LOCALAPPDATA 'Programs\lazygit-ai' }
$cpu = $env:PROCESSOR_ARCHITEW6432
if (-not $cpu) { $cpu = $env:PROCESSOR_ARCHITECTURE }
$arch = switch ($cpu) {
    'AMD64' { 'x86_64' }
    'ARM64' { 'arm64' }
    'x86' { '32-bit' }
    default { throw "Unsupported CPU architecture: $cpu" }
}
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
$tag = $env:LAZYGIT_AI_VERSION
if (-not $tag) { $tag = (Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest").tag_name }
if ($tag -notmatch '^v\d+\.\d+\.\d+([.-][A-Za-z0-9.-]+)?$') { throw "Invalid release version: $tag" }
$archive = "lazygit-ai_$($tag.Substring(1))_windows_$arch.zip"
$base = "https://github.com/$repo/releases/download/$tag"
$tempDir = Join-Path ([IO.Path]::GetTempPath()) ([Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $tempDir | Out-Null
try {
    Write-Host "Downloading lazygit-ai $tag (windows/$arch)..."
    $zip = Join-Path $tempDir $archive
    Invoke-WebRequest "$base/$archive" -OutFile $zip -UseBasicParsing
    $checksums = Join-Path $tempDir 'checksums.txt'
    Invoke-WebRequest "$base/checksums.txt" -OutFile $checksums -UseBasicParsing
    $matchesForFile = @(Get-Content $checksums | Where-Object { ($_ -split '\s+')[1] -eq $archive })
    if ($matchesForFile.Count -ne 1) { throw 'Missing or duplicate SHA256 checksum.' }
    $expected = ($matchesForFile[0] -split '\s+')[0]
    if ((Get-FileHash $zip -Algorithm SHA256).Hash -ne $expected) { throw 'SHA256 verification failed; installation cancelled.' }
    Expand-Archive $zip -DestinationPath $tempDir
    New-Item -ItemType Directory -Force -Path $installDir | Out-Null
    Copy-Item (Join-Path $tempDir 'lazygit-ai.exe') $installDir -Force
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if ($installDir -notin ($userPath -split ';')) {
        [Environment]::SetEnvironmentVariable('Path', "$installDir;$userPath", 'User')
    }
    if ($installDir -notin ($env:Path -split ';')) { $env:Path = "$installDir;$env:Path" }
    Write-Host "Installed: $installDir\lazygit-ai.exe"
    Write-Host 'Open a new terminal if needed. Git and Ollama are installed separately.'
} finally {
    Remove-Item $tempDir -Recurse -Force
}
