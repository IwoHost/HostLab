# hedit installer for Windows
# usage: irm https://iwohost.github.io/HostLab/hedit/install.ps1 | iex

$ErrorActionPreference = 'Stop'

$base    = 'https://iwohost.github.io/HostLab/hedit/dist'
$binary  = 'hedit-windows-amd64.exe'
$url     = "$base/$binary"
$installDir = "$env:LOCALAPPDATA\hedit"
$dest    = "$installDir\hedit.exe"

Write-Host ""
Write-Host "  hedit installer" -ForegroundColor White
Write-Host "  nano, but it looks good" -ForegroundColor Green
Write-Host ""

# create install dir
if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir | Out-Null
}

Write-Host "  -> downloading $binary..." -ForegroundColor Yellow
Invoke-WebRequest -Uri $url -OutFile $dest -UseBasicParsing
Write-Host "  v installed -> $dest" -ForegroundColor Green

# add to user PATH if missing
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($userPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable('Path', "$userPath;$installDir", 'User')
    Write-Host "  v added to PATH" -ForegroundColor Green
    Write-Host ""
    Write-Host "  ! restart your terminal, then:" -ForegroundColor Yellow
} else {
    Write-Host ""
    Write-Host "  ready:" -ForegroundColor White
}

Write-Host ""
Write-Host "  usage:  hedit <file>" -ForegroundColor White
Write-Host "          hedit notes.txt"
Write-Host "          hedit C:\Users\you\Desktop\todo.txt"
Write-Host ""
Write-Host "  ^S save  ^Q quit  ^F find  ^T themes" -ForegroundColor Green
Write-Host ""
