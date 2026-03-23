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

function Get-Arch {
    if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64" -or $env:PROCESSOR_ARCHITEW6432 -eq "ARM64") {
        return "arm64"
    }

    if ([System.Environment]::Is64BitOperatingSystem) {
        return "amd64"
    }

    try {
        $runtimeInfo = [System.Type]::GetType("System.Runtime.InteropServices.RuntimeInformation")
        if ($runtimeInfo) {
            $osArch = $runtimeInfo.GetProperty("OSArchitecture")
            if ($osArch) {
                $value = $osArch.GetValue($null).ToString()
                switch ($value) {
                    "X64"   { return "amd64" }
                    "Arm64" { return "arm64" }
                }
            }
        }
    } catch {
    }

    throw "Unsupported architecture for this installer."
}

$Arch = Get-Arch
$AssetName = "iitj-login-windows-$Arch.exe"

$Tag = $null
$AssetUrl = $null
try {
    $Release = Invoke-RestMethod -Uri $ApiUrl -ErrorAction Stop
    $Tag = $Release.tag_name
    if ($Release.assets) {
        $Asset = $Release.assets | Where-Object { $_.name -eq $AssetName } | Select-Object -First 1
        if ($Asset) {
            if ($Asset.browser_download_url) {
                $AssetUrl = $Asset.browser_download_url
            } elseif ($Asset.url) {
                $AssetUrl = $Asset.url
            }
        }
    }
} catch {
    $Tag = $null
}

if ($Tag -and $AssetUrl) {
    Write-Host "Downloading release binary $Tag..."
    try {
        $headers = @{ "User-Agent" = "iitj-login-bootstrap" }
        Invoke-WebRequest -Uri $AssetUrl -Headers $headers -OutFile "$InstDir\$Binary" -UseBasicParsing -ErrorAction Stop
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
    $env:PATH = "$env:PATH;$InstDir"
    Write-Host "Added $InstDir to user PATH."
    Write-Host "Updated PATH for this PowerShell session too.`n"
}

Write-Host ""
# Only launch the interactive installer when running in an interactive session.
# When piped via "irm ... | iex", stdin is not a terminal and credentials can't be entered.
if ([System.Environment]::UserInteractive -and [System.Console]::IsInputRedirected -eq $false) {
    Write-Host "Running installer..."
    & "$InstDir\$Binary" install
    Write-Host ""
    Write-Host "Installation step complete."
    Write-Host "Use a normal Command Prompt or normal PowerShell window for daily commands such as:"
    Write-Host "  iitj-login status"
    Write-Host "  iitj-login start"
    Write-Host "  iitj-login stop"
} else {
    Write-Host "Binary installed. Now run:"
    Write-Host "  iitj-login install"
    Write-Host "If PATH has not refreshed yet, run:"
    Write-Host "  & '$InstDir\$Binary' install"
}
