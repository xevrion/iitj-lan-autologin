# IITJ LAN Auto Login — Windows bootstrap
# Usage (run in PowerShell as Administrator for hosts/service setup):
#   irm <url>/bootstrap.ps1 | iex

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$Repo    = "https://github.com/xevrion/iitj-lan-autologin"
$Binary  = "iitj-login.exe"
$InstDir = "$env:LOCALAPPDATA\Programs\iitj-login"

Write-Host "IITJ LAN Auto Login — Windows Installer"
Write-Host "========================================`n"

New-Item -ItemType Directory -Force -Path $InstDir | Out-Null

# Build from source if Go is available.
$GoPath = Get-Command go -ErrorAction SilentlyContinue
if ($GoPath) {
    Write-Host "Go found — building from source..."
    $Tmp = [System.IO.Path]::GetTempPath() + [System.Guid]::NewGuid().ToString()
    New-Item -ItemType Directory -Path $Tmp | Out-Null
    try {
        git clone --depth 1 $Repo "$Tmp\src"
        Push-Location "$Tmp\src"
        go build -o "$Tmp\$Binary" .
        Pop-Location
        Copy-Item "$Tmp\$Binary" "$InstDir\$Binary" -Force
        Write-Host "Installed to $InstDir\$Binary"
    } finally {
        Remove-Item -Recurse -Force $Tmp -ErrorAction SilentlyContinue
    }
} else {
    # Download pre-built binary from GitHub Releases.
    $ApiUrl  = "https://api.github.com/repos/xevrion/iitj-lan-autologin/releases/latest"
    $Release = Invoke-RestMethod -Uri $ApiUrl -ErrorAction Stop
    $Tag     = $Release.tag_name

    if (-not $Tag) {
        Write-Error "No releases found. Install Go and re-run, or build manually."
        exit 1
    }

    $BinUrl = "$Repo/releases/download/$Tag/iitj-login-windows-amd64.exe"
    Write-Host "Downloading $BinUrl..."
    Invoke-WebRequest -Uri $BinUrl -OutFile "$InstDir\$Binary" -UseBasicParsing
    Write-Host "Installed to $InstDir\$Binary"
}

# Add install dir to user PATH if not already there.
$UserPath = [System.Environment]::GetEnvironmentVariable("PATH", "User")
if ($UserPath -notlike "*$InstDir*") {
    [System.Environment]::SetEnvironmentVariable("PATH", "$UserPath;$InstDir", "User")
    Write-Host "Added $InstDir to user PATH."
    Write-Host "Restart your shell for PATH changes to take effect.`n"
}

Write-Host ""
Write-Host "Running installer..."
& "$InstDir\$Binary" install
