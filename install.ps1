$ErrorActionPreference = "Stop"

$Repo = "Nings-379/skillctl"
$BinaryName = "skillctl.exe"
$Filename = "skillctl-windows-amd64.exe"
$InstallDir = "$env:USERPROFILE\bin"

$DownloadURL = "https://github.com/$Repo/releases/latest/download/$Filename"

Write-Host "Installing skillctl..."
Write-Host "  Download: $DownloadURL"
Write-Host ""

if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Write-Host "  Created directory: $InstallDir"
}

$TmpFile = Join-Path $env:TEMP "skillctl-download.exe"

try {
    Invoke-WebRequest -Uri $DownloadURL -OutFile $TmpFile -UseBasicParsing
} catch {
    Write-Host "Error: Failed to download from $DownloadURL"
    Write-Host "Check if a release exists at: https://github.com/$Repo/releases"
    Write-Host $_.Exception.Message
    exit 1
}

$InstallPath = Join-Path $InstallDir $BinaryName
Move-Item -Path $TmpFile -Destination $InstallPath -Force

$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    $env:Path = "$env:Path;$InstallDir"
    Write-Host "  Added $InstallDir to user PATH"
}

Write-Host ""
Write-Host "skillctl installed successfully!"
Write-Host "  Location: $InstallPath"
Write-Host ""
Write-Host "Open a new terminal and run 'skillctl --help' to get started."
