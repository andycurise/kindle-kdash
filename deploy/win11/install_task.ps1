<#
.SYNOPSIS
    Register kdash as a limited scheduled task for the current user's logon.
.DESCRIPTION
    This is not an SCM service and does not run while the user is logged off.
    Only -config is passed to kdash.exe; listen_addr remains in kdash.json.
#>
[CmdletBinding()]
param(
    [string]$TaskName = "KindleDashboard",
    [string]$WorkDir = "",
    [string]$ExePath = "",
    [switch]$NoStart
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version 2.0
$TaskPath = "\"
$OldTaskXml = $null
$HadTask = $false
$OldTaskWasRunning = $false
$OldConfigBytes = $null
$TransactionStarted = $false

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

function Get-BaseUrl {
    param($Config, [switch]$ForceLoopback)
    $address = Get-ConfigValue -Config $Config -Name "listen_addr" -Default ":8088"
    $match = [regex]::Match($address, '^\[(?<host>[^\]]+)\]:(?<port>[0-9]+)$')
    if ($match.Success) {
        $hostName = $match.Groups['host'].Value
        $port = $match.Groups['port'].Value
    } else {
        $separator = $address.LastIndexOf(':')
        if ($separator -lt 0) { return $null }
        $hostName = $address.Substring(0, $separator)
        $port = $address.Substring($separator + 1)
    }
    if ($port -notmatch '^[0-9]+$') { return $null }
    if ($ForceLoopback -or [string]::IsNullOrWhiteSpace($hostName) -or $hostName -eq "0.0.0.0" -or $hostName -eq "::") {
        $hostName = "127.0.0.1"
    }
    if ($hostName.Contains(':')) { $hostName = "[$hostName]" }
    return "http://${hostName}:$port"
}

function Get-MatchingProcesses {
    param([string]$ExpectedExe)
    $expected = [IO.Path]::GetFullPath($ExpectedExe)
    $result = @()
    try {
        foreach ($process in @(Get-CimInstance Win32_Process -Filter "Name='kdash.exe'" -ErrorAction Stop)) {
            if ([string]::IsNullOrWhiteSpace($process.ExecutablePath)) { continue }
            if ([string]::Equals([IO.Path]::GetFullPath($process.ExecutablePath), $expected,
                    [StringComparison]::OrdinalIgnoreCase)) { $result += $process }
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
        if (-not [string]::IsNullOrWhiteSpace($action.Execute) -and
            [string]::Equals([IO.Path]::GetFullPath($action.Execute.Trim('"')), $expected,
                [StringComparison]::OrdinalIgnoreCase)) { return }
    }
    throw "Task $TaskName belongs to a different executable; refusing to change it."
}

function Stop-CurrentRuntime {
    $task = Get-CurrentTask
    if ($null -ne $task) { Assert-TaskPath -Task $task -ExpectedExe $ExePath }
    $matching = @(Get-MatchingProcesses -ExpectedExe $ExePath)
    if ($matching.Count -gt 0) {
        $config = Read-Config -Path $ConfigPath
        $baseUrl = Get-BaseUrl -Config $config -ForceLoopback
        if ($null -ne $baseUrl) {
            $token = Get-ConfigValue -Config $config -Name "settings_token"
            try {
                Invoke-WebRequest -UseBasicParsing -Method Post -Uri "$baseUrl/api/shutdown" `
                    -Headers @{ "X-KDash-Token" = $token; "X-KDash-Client" = "kdash" } -TimeoutSec 5 | Out-Null
            } catch { }
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
    foreach ($process in @(Get-MatchingProcesses -ExpectedExe $ExePath)) {
        Stop-Process -Id $process.ProcessId -Force -ErrorAction SilentlyContinue
    }
}

function New-SettingsToken {
    $bytes = New-Object byte[] 32
    $rng = [Security.Cryptography.RandomNumberGenerator]::Create()
    try { $rng.GetBytes($bytes) } finally { $rng.Dispose() }
    return ([BitConverter]::ToString($bytes)).Replace("-", "").ToLowerInvariant()
}

function Ensure-SettingsToken {
    if (-not (Test-Path -LiteralPath $ConfigPath -PathType Leaf)) {
        throw "Configuration file not found: $ConfigPath"
    }
    $config = Read-Config -Path $ConfigPath
    if ($null -eq $config) { throw "Configuration JSON is invalid; no configuration values were logged." }
    $token = Get-ConfigValue -Config $config -Name "settings_token"
    $examples = @("", "change-me", "changeme", "example", "replace-me", "your-secret-token")
    if ($examples -notcontains $token.ToLowerInvariant()) { return }

    $script:OldConfigBytes = [IO.File]::ReadAllBytes($ConfigPath)
    $token = New-SettingsToken
    if ($null -eq $config.PSObject.Properties["settings_token"]) {
        $config | Add-Member -NotePropertyName "settings_token" -NotePropertyValue $token
    } else {
        $config.settings_token = $token
    }
    $json = $config | ConvertTo-Json -Depth 30
    $encoding = New-Object System.Text.UTF8Encoding -ArgumentList $false
    [IO.File]::WriteAllText($ConfigPath, $json + [Environment]::NewLine, $encoding)
}

function Wait-Health {
    $config = Read-Config -Path $ConfigPath
    if ($null -eq $config) { throw "Configuration could not be read for the health check." }
    $baseUrl = Get-BaseUrl -Config $config
    if ($null -eq $baseUrl) { throw "listen_addr is invalid for a health check." }
    $expectedVersion = (& $ExePath -version)
    if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($expectedVersion)) { throw "kdash.exe did not return a version." }
    for ($attempt = 0; $attempt -lt 45; $attempt++) {
        try {
            $response = Invoke-WebRequest -UseBasicParsing -Method Get -Uri "$baseUrl/api/status" -TimeoutSec 3
            $status = $response.Content | ConvertFrom-Json
            if ([string]$status.version -ne [string]$expectedVersion) { throw "Health endpoint returned a different kdash version." }
            Write-Host "Health check passed: $baseUrl/api/status (version $expectedVersion)"
            return
        } catch { Start-Sleep -Seconds 1 }
    }
    throw "Health check failed at $baseUrl/api/status after 45 seconds."
}

if ([string]::IsNullOrWhiteSpace($WorkDir)) { $WorkDir = $PSScriptRoot }
$WorkDir = [IO.Path]::GetFullPath($WorkDir)
if ([string]::IsNullOrWhiteSpace($ExePath)) {
    $candidate = Join-Path $WorkDir "kdash.exe"
    if (Test-Path -LiteralPath $candidate -PathType Leaf) {
        $ExePath = $candidate
    } else {
        $candidate = Join-Path (Split-Path -Parent $WorkDir) "build\kdash-windows-amd64.exe"
        if (Test-Path -LiteralPath $candidate -PathType Leaf) { $ExePath = $candidate }
    }
}
if ([string]::IsNullOrWhiteSpace($ExePath) -or -not (Test-Path -LiteralPath $ExePath -PathType Leaf)) {
    throw "kdash.exe was not found in $WorkDir; specify -ExePath."
}
$ExePath = [IO.Path]::GetFullPath($ExePath)
$ConfigPath = Join-Path $WorkDir "kdash.json"

try {
    Write-Host "Configuring current-user logon task $TaskName (Limited, not an SCM service)."
    Write-Host "Executable: $ExePath"
    $oldTask = Get-CurrentTask
    if ($null -ne $oldTask) {
        Assert-TaskPath -Task $oldTask -ExpectedExe $ExePath
        $HadTask = $true
        $OldTaskXml = Export-ScheduledTask -TaskName $TaskName -TaskPath $TaskPath
        $OldTaskWasRunning = $oldTask.State -eq "Running"
    }
    $TransactionStarted = $true
    Stop-CurrentRuntime
    Ensure-SettingsToken

    $identity = [Security.Principal.WindowsIdentity]::GetCurrent().Name
    $arguments = '-config "{0}"' -f $ConfigPath
    $action = New-ScheduledTaskAction -Execute $ExePath -Argument $arguments -WorkingDirectory $WorkDir
    $trigger = New-ScheduledTaskTrigger -AtLogOn -User $identity
    $principal = New-ScheduledTaskPrincipal -UserId $identity -LogonType Interactive -RunLevel Limited
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries `
        -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1) `
        -ExecutionTimeLimit ([TimeSpan]::Zero)
    Register-ScheduledTask -TaskName $TaskName -TaskPath $TaskPath -Action $action `
        -Trigger $trigger -Principal $principal -Settings $settings `
        -Description "kdash current-user logon task; not an SCM service" -Force | Out-Null
    if (-not $NoStart) {
        Start-ScheduledTask -TaskName $TaskName -TaskPath $TaskPath
        Wait-Health
    }
    $TransactionStarted = $false
    Write-Host "Task $TaskName was registered for the current interactive user with Limited rights."
    if ($NoStart) { Write-Host "The task was not started (--NoStart)." }
} catch {
    if ($TransactionStarted) {
        try {
            $task = Get-CurrentTask
            if ($null -ne $task) {
                Stop-ScheduledTask -TaskName $TaskName -TaskPath $TaskPath -ErrorAction SilentlyContinue | Out-Null
                Unregister-ScheduledTask -TaskName $TaskName -TaskPath $TaskPath -Confirm:$false -ErrorAction SilentlyContinue | Out-Null
            }
            if ($null -ne $OldConfigBytes) { [IO.File]::WriteAllBytes($ConfigPath, $OldConfigBytes) }
            if ($HadTask) {
                Register-ScheduledTask -TaskName $TaskName -TaskPath $TaskPath -Xml $OldTaskXml -Force | Out-Null
                if ($OldTaskWasRunning) { Start-ScheduledTask -TaskName $TaskName -TaskPath $TaskPath }
            }
        } catch { }
    }
    throw "Task installation failed; rollback was attempted. No configuration values were logged."
}
