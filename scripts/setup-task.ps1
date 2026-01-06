# Cyber Range - Windows Scheduled Task Setup
# Run as Administrator when preparing the base image

param(
    [Parameter(Mandatory=$true)]
    [string]$ServerURL
)

$TaskName = "CyberRangeConfig"
$ClientPath = "C:\ProgramData\cyber-range\client.exe"

# Check admin
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Error "This script must be run as Administrator"
    exit 1
}

# Create directory
$dir = Split-Path -Parent $ClientPath
if (-not (Test-Path $dir)) {
    New-Item -Path $dir -ItemType Directory -Force | Out-Null
    Write-Host "Created directory: $dir"
}

# Check for client.exe
if (-not (Test-Path $ClientPath)) {
    Write-Warning "client.exe not found at $ClientPath"
    Write-Host "Copy client.exe to $ClientPath"
}

# Remove existing task
$existingTask = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
if ($existingTask) {
    Write-Host "Removing existing task: $TaskName"
    Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false
}

# Create task
$action = New-ScheduledTaskAction -Execute $ClientPath -Argument "-server $ServerURL"
$trigger = New-ScheduledTaskTrigger -AtStartup
$principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" -LogonType ServiceAccount -RunLevel Highest
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable

Register-ScheduledTask -TaskName $TaskName -Action $action -Trigger $trigger -Principal $principal -Settings $settings -Description "Configures hostname and network on first boot"

Write-Host ""
Write-Host "Task '$TaskName' created successfully!"
Write-Host ""
Write-Host "Details:"
Write-Host "  - Runs at: System startup (before login)"
Write-Host "  - Runs as: SYSTEM"
Write-Host "  - Command: $ClientPath -server $ServerURL"
Write-Host "  - Random delay: 0-30 seconds (staggered startup)"
Write-Host ""
Write-Host "Logs: C:\ProgramData\cyber-range\config.log"
