# IITJ LAN Auto Login — Windows bootstrap
# Usage (run in PowerShell as Administrator for hosts/service setup):
#   irm <url>/bootstrap.ps1 | iex

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$Repo    = "https://github.com/xevrion/iitj-lan-autologin"
$Binary  = "iitj-login.exe"
$InstDir = "$env:LOCALAPPDATA\Programs\iitj-login"
$TargetPath = Join-Path $InstDir $Binary
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
$InstallError = $null

function Download-ReleaseBinary {
    param(
        [Parameter(Mandatory = $true)][string]$Uri,
        [Parameter(Mandatory = $true)][string]$OutFile
    )

    $headers = @{ "User-Agent" = "iitj-login-bootstrap" }

    try {
        Invoke-WebRequest -Uri $Uri -Headers $headers -OutFile $OutFile -UseBasicParsing -ErrorAction Stop
        return
    } catch {
        $primaryError = $_.Exception.Message
    }

    try {
        $webClient = New-Object System.Net.WebClient
        $webClient.Headers.Add("User-Agent", "iitj-login-bootstrap")
        $webClient.DownloadFile($Uri, $OutFile)
        return
    } catch {
        throw "Invoke-WebRequest failed: $primaryError; WebClient failed: $($_.Exception.Message)"
    } finally {
        if ($webClient) {
            $webClient.Dispose()
        }
    }
}

function Get-TaskInfo {
    $task = @{
        Exists = $false
        Running = $false
    }

    try {
        $out = schtasks /query /tn "IITJ-LAN-AutoLogin" /fo list /v 2>$null
        if ($LASTEXITCODE -eq 0) {
            $task.Exists = $true
            foreach ($line in $out) {
                if ($line -match '^\s*Status:\s*(.+)$') {
                    $task.Running = $matches[1].ToLower().Contains("running")
                    break
                }
            }
        }
    } catch {
    }

    return $task
}

function Stop-InstalledTask {
    param(
        [Parameter(Mandatory = $true)][string]$BinaryPath
    )

    $task = Get-TaskInfo
    if ($task.Exists) {
        try {
            schtasks /end /tn "IITJ-LAN-AutoLogin" *> $null
        } catch {
        }
    }

    if (Test-Path $BinaryPath) {
        try {
            taskkill /f /t /im "iitj-login.exe" *> $null
        } catch {
        }
        Get-Process -Name "iitj-login" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
        Wait-ForBinaryUnlock -Path $BinaryPath
    }

    return $task
}

function Wait-ForBinaryUnlock {
    param(
        [Parameter(Mandatory = $true)][string]$Path
    )

    if (-not (Test-Path $Path)) {
        return
    }

    for ($i = 0; $i -lt 30; $i++) {
        try {
            $stream = [System.IO.File]::Open($Path, [System.IO.FileMode]::Open, [System.IO.FileAccess]::ReadWrite, [System.IO.FileShare]::None)
            $stream.Dispose()
            return
        } catch {
            Start-Sleep -Milliseconds 500
        }
    }

    throw ("Timed out waiting for {0} to be released by another process." -f $Path)
}

function Install-BinaryFile {
    param(
        [Parameter(Mandatory = $true)][string]$SourcePath,
        [Parameter(Mandatory = $true)][string]$DestinationPath
    )

    if (Test-Path $DestinationPath) {
        Wait-ForBinaryUnlock -Path $DestinationPath
        Remove-Item $DestinationPath -Force -ErrorAction Stop
    }

    Move-Item -Path $SourcePath -Destination $DestinationPath -Force
}

function Start-InstalledTaskIfNeeded {
    param(
        [Parameter(Mandatory = $true)]$PreviousTask
    )

    if ($PreviousTask.Exists) {
        try {
            schtasks /run /tn "IITJ-LAN-AutoLogin" *> $null
        } catch {
        }
    }
}

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
    $taskBeforeInstall = Stop-InstalledTask -BinaryPath $TargetPath
    $downloadPath = Join-Path $InstDir ($Binary + ".download")
    try {
        Download-ReleaseBinary -Uri $AssetUrl -OutFile $downloadPath
        Install-BinaryFile -SourcePath $downloadPath -DestinationPath $TargetPath
        Start-InstalledTaskIfNeeded -PreviousTask $taskBeforeInstall
        Write-Host "Installed to $TargetPath"
    } catch {
        Remove-Item $downloadPath -Force -ErrorAction SilentlyContinue
        $InstallError = $_.Exception.Message
        Start-InstalledTaskIfNeeded -PreviousTask $taskBeforeInstall
        $Tag = $null
    }
}

if (-not $Tag) {
    $GoPath = Get-Command go -ErrorAction SilentlyContinue
    $GitPath = Get-Command git -ErrorAction SilentlyContinue

    if (-not $GoPath -or -not $GitPath) {
        if ($InstallError) {
            Write-Error ("Release install failed for windows/{0}: {1}`nSource build fallback is unavailable because go and git were not found." -f $Arch, $InstallError)
        } else {
            Write-Error ("No release binary found for windows/{0} and source build fallback is unavailable." -f $Arch)
        }
        exit 1
    }

    Write-Host ("No downloadable release found for windows/{0} - building from source..." -f $Arch)
    $Tmp = [System.IO.Path]::GetTempPath() + [System.Guid]::NewGuid().ToString()
    New-Item -ItemType Directory -Path $Tmp | Out-Null
    try {
        git clone --depth 1 $Repo "$Tmp\src"
        Push-Location "$Tmp\src"
        go build -o "$Tmp\$Binary" .
        Pop-Location
        $taskBeforeInstall = Stop-InstalledTask -BinaryPath $TargetPath
        Install-BinaryFile -SourcePath "$Tmp\$Binary" -DestinationPath $TargetPath
        Start-InstalledTaskIfNeeded -PreviousTask $taskBeforeInstall
        Write-Host "Installed to $TargetPath"
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
    & $TargetPath install
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
    Write-Host "  & '$TargetPath' install"
}
