# runweb installer for Windows
# usage: irm https://iwohost.github.io/HostLab/runweb/install.ps1 | iex

$ErrorActionPreference = 'Stop'

$url        = 'https://iwohost.github.io/HostLab/runweb/runweb.py'
$installDir = "$env:LOCALAPPDATA\runweb"
$script     = "$installDir\runweb.py"
$shim       = "$installDir\runweb.cmd"

Write-Host ""
Write-Host "  runweb installer" -ForegroundColor White
Write-Host "  serve any page on your LAN" -ForegroundColor Green
Write-Host ""

# runweb is a python script — needs python 3 on PATH
$py = $null
foreach ($candidate in @('python', 'python3', 'py')) {
    $cmd = Get-Command $candidate -ErrorAction SilentlyContinue
    if ($cmd) {
        $ver = & $candidate -c "import sys; print(sys.version_info[0])" 2>$null
        if ($ver -eq '3') { $py = $candidate; break }
    }
}
if (-not $py) {
    Write-Host "  x python 3 required but not found" -ForegroundColor Red
    Write-Host "    install it from https://www.python.org/downloads/ (tick 'Add to PATH') and run this again"
    return
}

$pyver = & $py -c "import sys; print('%d.%d' % sys.version_info[:2])"
Write-Host "  -> found python $pyver ($py)" -ForegroundColor Yellow

if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir | Out-Null
}

Write-Host "  -> downloading runweb..." -ForegroundColor Yellow
Invoke-WebRequest -Uri $url -OutFile $script -UseBasicParsing

# .cmd shim so "runweb" works as a plain command
"@echo off`r`n$py `"%~dp0runweb.py`" %*" | Set-Content -Path $shim -Encoding ASCII

Write-Host "  v installed -> $shim" -ForegroundColor Green

# add to user PATH (persistent, survives reboots)
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($userPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable('Path', "$userPath;$installDir", 'User')
    Write-Host "  v added to PATH (user)" -ForegroundColor Green
}

# also update PATH in THIS session so runweb works immediately — no restart needed
if ($env:PATH -notlike "*$installDir*") {
    $env:PATH = "$env:PATH;$installDir"
}

Write-Host ""
Write-Host "  ready — cd into a folder and type:" -ForegroundColor White
Write-Host ""
Write-Host "    runweb index.html -o" -ForegroundColor Green
Write-Host ""
Write-Host "  -p port   -o open browser   -q QR code   -s https" -ForegroundColor DarkGreen
Write-Host ""
