<#
.SYNOPSIS
    Stop the current kdash installation, optionally removing its logon task.
.DESCRIPTION
    Requests POST /api/shutdown first. Fallback process termination is limited
    to processes whose complete ExecutablePath equals this installation's exe.
#>
[CmdletBinding()]
param(
    [string]$TaskName = "KindleDashboard",
    [string]$WorkDir = "",
    [string]$ExePath = "",
    [switch]$StopOnly
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version 2.0
$TaskPath = "\"

function Read-Config {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) { return $null }
    try { return (Get-Content -LiteralPath $Path -Raw -Encoding UTF8 | ConvertFrom-Json) }
    catch { return $null }
}

function Get-ConfigValue {
    param($Config, [string]$Name, [string]$Default = "")
    if ($null -eq $Config) { return $Default }
    $property = $Config.PSObject.Properties[$Name]
    if ($null -eq $property -or $null -eq $property.Value) { return $Default }
    return [string]$property.Value
}

function Get-LoopbackUrl {
    param($Config)
    $address = Get-ConfigValue -Config $Config -Name "listen_addr" -Default ":8088"
    $match = [regex]::Match($address, '^\[[^\]]+\]:(?<port>[0-9]+)$')
    if ($match.Success) {
        $port = $match.Groups['port'].Value
    } else {
        $separator = $address.LastIndexOf(':')
        if ($separator -lt 0) { return $null }
        $port = $address.Substring($separator + 1)
    }
    if ($port -notmatch '^[0-9]+$') { return $null }
    return "http://127.0.0.1:$port"
}

function Get-MatchingProcesses {
    param([string]$ExpectedExe)
    $expected = [IO.Path]::GetFullPath($ExpectedExe)
    $result = @()
    try {
        foreach ($process in @(Get-CimInstance Win32_Process -Filter "Name='kdash.exe'" -ErrorAction Stop)) {
            if ([string]::IsNullOrWhiteSpace($process.ExecutablePath)) { continue }
            try {
                if ([string]::Equals([IO.Path]::GetFullPath($process.ExecutablePath), $expected,
                        [StringComparison]::OrdinalIgnoreCase)) { $result += $process }
            } catch { }
        }
    } catch { }
    return $result
}

function Get-CurrentTask {
    return @(Get-ScheduledTask -TaskName $TaskName -TaskPath $TaskPath -ErrorAction SilentlyContinue |
        Select-Object -First 1)
}

function Assert-TaskPath {
    param($Task, [string]$ExpectedExe)
    $expected = [IO.Path]::GetFullPath($ExpectedExe)
    foreach ($action in @($Task.Actions)) {
        if (-not [string]::IsNullOrWhiteSpace($action.Execute)) {
            try {
                if ([string]::Equals([IO.Path]::GetFullPath($action.Execute.Trim('"')), $expected,
                        [StringComparison]::OrdinalIgnoreCase)) { return }
            } catch { }
        }
    }
    throw "Task $TaskName points to a different executable; refusing to stop or remove it."
}

if ([string]::IsNullOrWhiteSpace($WorkDir)) { $WorkDir = $PSScriptRoot }
$WorkDir = [IO.Path]::GetFullPath($WorkDir)
if ([string]::IsNullOrWhiteSpace($ExePath)) { $ExePath = Join-Path $WorkDir "kdash.exe" }
$ExePath = [IO.Path]::GetFullPath($ExePath)
$ConfigPath = Join-Path $WorkDir "kdash.json"

$task = Get-CurrentTask
if ($null -ne $task) { Assert-TaskPath -Task $task -ExpectedExe $ExePath }
$matching = @(Get-MatchingProcesses -ExpectedExe $ExePath)

if ($matching.Count -gt 0) {
    $config = Read-Config -Path $ConfigPath
    $baseUrl = Get-LoopbackUrl -Config $config
    if ($null -ne $baseUrl) {
        $token = Get-ConfigValue -Config $config -Name "settings_token"
        try {
            Invoke-WebRequest -UseBasicParsing -Method Post -Uri "$baseUrl/api/shutdown" `
                -Headers @{ "X-KDash-Token" = $token; "X-KDash-Client" = "kdash" } -TimeoutSec 5 | Out-Null
            Write-Host "Graceful shutdown requested from the loopback API."
        } catch {
            Write-Host "Graceful shutdown is unavailable; using the scoped task/process fallback."
        }
    }
    $deadline = [DateTime]::UtcNow.AddSeconds(10)
    do {
        $matching = @(Get-MatchingProcesses -ExpectedExe $ExePath)
        if ($matching.Count -eq 0) { break }
        Start-Sleep -Milliseconds 500
    } while ([DateTime]::UtcNow -lt $deadline)
}

if ($null -ne $task) {
    Stop-ScheduledTask -TaskName $TaskName -TaskPath $TaskPath -ErrorAction SilentlyContinue | Out-Null
}

# Process termination is restricted to this complete ExePath.
foreach ($process in @(Get-MatchingProcesses -ExpectedExe $ExePath)) {
    Stop-Process -Id $process.ProcessId -Force -ErrorAction SilentlyContinue
}
Start-Sleep -Milliseconds 500
if (@(Get-MatchingProcesses -ExpectedExe $ExePath).Count -gt 0) {
    throw "A process at $ExePath could not be stopped; no other kdash process was touched."
}

if (-not $StopOnly -and $null -ne $task) {
    Unregister-ScheduledTask -TaskName $TaskName -TaskPath $TaskPath -Confirm:$false | Out-Null
    Write-Host "Removed current-user task $TaskName."
}
if ($StopOnly) {
    Write-Host "Stopped only the kdash installation at $ExePath."
} else {
    Write-Host "Task removal completed for $ExePath; program and configuration files were not deleted."
}
