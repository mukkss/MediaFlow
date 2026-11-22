# MediaFlow Windows Build & Packaging Script
# Requires: PowerShell 5.1+, Go 1.26+, Node.js 18+, Inno Setup 6+
Param(
    [string]$Version = "1.0.0",
    [string]$FFmpegRelease = "7.1",
    [switch]$SkipFrontend = $false
)

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RootDir = Split-Path -Parent $ScriptDir
Set-Location $RootDir

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "   MediaFlow Windows Release Builder" -ForegroundColor Cyan
Write-Host "   Target Version: $Version" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

# 1. Pinned Dependencies Verification
Write-Host "[1/5] Verifying toolchain..." -ForegroundColor Yellow
$goVer = go version
Write-Host "Found Go: $goVer"

# 2. Build Svelte Frontend Assets
if (-not $SkipFrontend) {
    Write-Host "[2/5] Building Svelte frontend..." -ForegroundColor Yellow
    Set-Location "$RootDir/frontend"
    npm install --no-audit --no-fund
    npm run build
    Set-Location $RootDir
}

# 3. Compile Windows Binary
Write-Host "[3/5] Compiling MediaFlow.exe (Windows AMD64)..." -ForegroundColor Yellow
if (-not (Test-Path "$RootDir/bin")) {
    New-Item -ItemType Directory -Path "$RootDir/bin" | Out-Null
}

$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -ldflags "-s -w -H windowsgui -X main.version=$Version" -o "$RootDir/bin/MediaFlow.exe" ./cmd/desktop

if (-not (Test-Path "$RootDir/bin/MediaFlow.exe")) {
    throw "Build failed: bin/MediaFlow.exe not generated"
}
Write-Host "Generated: bin/MediaFlow.exe" -ForegroundColor Green

# 4. Check for Bundled External Dependencies
Write-Host "[4/5] Checking bundled assets (FFmpeg & Codecs)..." -ForegroundColor Yellow
$ffmpegDir = "$RootDir/resources/ffmpeg"
if (-not (Test-Path "$ffmpegDir/ffmpeg.exe")) {
    Write-Warning "resources/ffmpeg/ffmpeg.exe not found. To bundle, download BtbN LGPL static release $FFmpegRelease and place ffmpeg.exe & ffprobe.exe in $ffmpegDir."
}

# 5. Compile Inno Setup Package
Write-Host "[5/5] Checking Inno Setup compiler..." -ForegroundColor Yellow
$innoPath = "C:\Program Files (x86)\Inno Setup 6\ISCC.exe"
if (Test-Path $innoPath) {
    Write-Host "Compiling Inno Setup package..." -ForegroundColor Green
    & $innoPath "$RootDir/installer/setup.iss"
    Write-Host "Installer package generated in bin/" -ForegroundColor Green
} else {
    Write-Host "Inno Setup compiler (ISCC.exe) not found at default location. When compiling on Windows, run: ISCC installer/setup.iss" -ForegroundColor DarkGray
}

Write-Host "Build complete!" -ForegroundColor Cyan
