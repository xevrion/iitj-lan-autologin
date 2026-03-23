# IITJ LAN Auto Login — Windows bootstrap
# Usage (run in PowerShell as Administrator for hosts/service setup):
#   irm <url>/bootstrap.ps1 | iex

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$Repo    = "https://github.com/xevrion/iitj-lan-autologin"
$Binary  = "iitj-login.exe"
$InstDir = "$env:LOCALAPPDATA\Programs\iitj-login"
$ApiUrl  = "https://api.github.com/repos/xevrion/iitj-lan-autologin/releases/latest"

Write-Host "IITJ LAN Auto Login — Windows Installer"
Write-Host "========================================`n"

New-Item -ItemType Directory -Force -Path $InstDir | Out-Null

$Arch = switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture) {
    "X64"   { "amd64" }
    "Arm64" { "arm64" }
    default { throw "Unsupported architecture: $([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture)" }
}

$Tag = $null
try {
    $Release = Invoke-RestMethod -Uri $ApiUrl -ErrorAction Stop
    $Tag = $Release.tag_name
} catch {
    $Tag = $null
}

if ($Tag) {
    $BinUrl = "$Repo/releases/download/$Tag/iitj-login-windows-$Arch.exe"
    Write-Host "Downloading release binary $Tag..."
    try {
        Invoke-WebRequest -Uri $BinUrl -OutFile "$InstDir\$Binary" -UseBasicParsing -ErrorAction Stop
        Write-Host "Installed to $InstDir\$Binary"
    } catch {
        Remove-Item "$InstDir\$Binary" -Force -ErrorAction SilentlyContinue
        $Tag = $null
    }
}

if (-not $Tag) {
    $GoPath = Get-Command go -ErrorAction SilentlyContinue
    $GitPath = Get-Command git -ErrorAction SilentlyContinue

    if (-not $GoPath -or -not $GitPath) {
        Write-Error "No release binary found for windows/$Arch and source build fallback is unavailable."
        exit 1
    }

    Write-Host "No downloadable release found for windows/$Arch — building from source..."
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
}

# Add install dir to user PATH if not already there.
$UserPath = [System.Environment]::GetEnvironmentVariable("PATH", "User")
if ($UserPath -notlike "*$InstDir*") {
    [System.Environment]::SetEnvironmentVariable("PATH", "$UserPath;$InstDir", "User")
    Write-Host "Added $InstDir to user PATH."
    Write-Host "Restart your shell for PATH changes to take effect.`n"
}

Write-Host ""
# Only launch the interactive installer when running in an interactive session.
# When piped via "irm ... | iex", stdin is not a terminal and credentials can't be entered.
if ([System.Environment]::UserInteractive -and [System.Console]::IsInputRedirected -eq $false) {
    Write-Host "Running installer..."
    & "$InstDir\$Binary" install
} else {
    Write-Host "Binary installed. Now run:"
    Write-Host "  iitj-login install"
}
