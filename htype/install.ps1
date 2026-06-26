# htype installer for Windows
$ErrorActionPreference = "Stop"

$BinaryName = "htype.exe"
$Url = "https://iwohost.github.io/HostLab/htype/dist/htype-windows-amd64.exe"
$InstallDir = "$env:LOCALAPPDATA\htype"

Write-Host "-> downloading htype (windows/amd64)..."
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
Invoke-WebRequest -Uri $Url -OutFile "$InstallDir\$BinaryName" -UseBasicParsing
Write-Host "v installed to $InstallDir\$BinaryName"

# add to PATH permanently
$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($currentPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$currentPath;$InstallDir", "User")
    $env:Path += ";$InstallDir"
    Write-Host "  added $InstallDir to PATH"
}

Write-Host ""
Write-Host "  htype             # 25 words"
Write-Host "  htype 50          # 50 words"
Write-Host "  htype -t 60       # 60-second timed test"
Write-Host "  htype -m code     # code identifiers"
Write-Host "  htype -m quote    # famous quotes"
Write-Host ""
Write-Host "  Tab mode . T time . +/- words . ESC quit"
