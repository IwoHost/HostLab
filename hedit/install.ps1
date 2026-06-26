# hedit installer for Windows
# usage: irm https://iwohost.github.io/HostLab/hedit/install.ps1 | iex

$ErrorActionPreference = 'Stop'

$base       = 'https://iwohost.github.io/HostLab/hedit/dist'
$binary     = 'hedit-windows-amd64.exe'
$url        = "$base/$binary"
$installDir = "$env:LOCALAPPDATA\hedit"
$dest       = "$installDir\hedit.exe"

Write-Host ""
Write-Host "  hedit installer" -ForegroundColor White
Write-Host "  nano, but it looks good" -ForegroundColor Green
Write-Host ""

# create install dir
if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir | Out-Null
}

Write-Host "  -> downloading hedit..." -ForegroundColor Yellow
Invoke-WebRequest -Uri $url -OutFile $dest -UseBasicParsing
Write-Host "  v installed -> $dest" -ForegroundColor Green

# add to user PATH (persistent, survives reboots)
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($userPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable('Path', "$userPath;$installDir", 'User')
    Write-Host "  v added to PATH (user)" -ForegroundColor Green
}

# also update PATH in THIS session so hedit works immediately — no restart needed
if ($env:PATH -notlike "*$installDir*") {
    $env:PATH = "$env:PATH;$installDir"
}

Write-Host ""
Write-Host "  ready — type this right now:" -ForegroundColor White
Write-Host ""
Write-Host "    hedit notes.txt" -ForegroundColor Green
Write-Host ""
Write-Host "  ^S save   ^Q quit   ^F find   ^U themes   Alt+BS del line" -ForegroundColor DarkGreen
Write-Host ""
