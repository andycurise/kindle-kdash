<#
.SYNOPSIS
    Install, upgrade, or remove kdash for the current Windows 11 x64 user.
.DESCRIPTION
    Downloads and verifies kdash-windows-amd64.zip, preserves existing
    configuration, and registers a limited, current-user AtLogOn task.
    This is not an SCM service and does not run before user logon.
#>
[CmdletBinding()]
param(
    [string]$Repo = "andycurise/kindle-kdash",
    [string]$Version = "latest",
    [string]$Prefix = "",
    [switch]$NonInteractive,
    [switch]$NoStart,
    [switch]$Uninstall,
    [switch]$KeepConfig,
    [string]$ReleaseBase = $env:KDASH_RELEASE_BASE
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version 2.0
$script:TaskName = "KindleDashboard"
$script:TaskPath = "\"
$script:Stage = "initialization"
$script:TransactionActive = $false
$script:TempDir = $null
$script:LocalReleaseBase = $null
$script:RemoteReleaseBase = $null
$script:HadTask = $false
$script:OldTaskXml = $null
$script:OldTaskWasRunning = $false
$script:HadDataDir = $false
$script:HadLogDir = $false
$script:BackedUp = @{}
$script:InstallFiles = @(
    "kdash.exe", "kdash.json", "kdash.json.example", "LICENSE", "THIRD_PARTY_NOTICES.md",
    "README.md", "RELEASES.md", "start.bat", "stop.bat", "start_service.vbs",
    "install_task.bat", "install_task.ps1", "uninstall_task.bat", "uninstall_task.ps1"
)

function Get-FullPath {
    param([Parameter(Mandatory = $true)][string]$Path)
    return [System.IO.Path]::GetFullPath($Path).TrimEnd('\')
}

function Read-KdashConfig {
    param([Parameter(Mandatory = $true)][string]$Path)
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) { return $null }
    try {
        return (Get-Content -LiteralPath $Path -Raw -Encoding UTF8 | ConvertFrom-Json)
    } catch {
        # A malformed configuration must not prevent an exact-path process stop.
        # Do not emit parser errors: they can contain source JSON and tokens.
        return $null
    }
}

function Get-ConfigValue {
    param(
        [Parameter(Mandatory = $false)]$Config,
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $false)][string]$Default = ""
    )
    if ($null -eq $Config) { return $Default }
    $property = $Config.PSObject.Properties[$Name]
    if ($null -eq $property -or $null -eq $property.Value) { return $Default }
    return [string]$property.Value
}

function Get-KdashBaseUrl {
    param(
        [Parameter(Mandatory = $false)]$Config,
        [switch]$ForceLoopback
    )
    $address = Get-ConfigValue -Config $Config -Name "listen_addr" -Default ":8088"
    $hostName = ""
    $port = ""
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
    if ($ForceLoopback -or [string]::IsNullOrWhiteSpace($hostName) -or
        $hostName -eq "0.0.0.0" -or $hostName -eq "::") {
        $hostName = "127.0.0.1"
    }
    if ($hostName.Contains(':')) { $hostName = "[$hostName]" }
    return "http://${hostName}:$port"
}

function Get-KdashProcesses {
    param([Parameter(Mandatory = $true)][string]$ExePath)
    $expected = Get-FullPath -Path $ExePath
    $result = @()
    try {
        $processes = Get-CimInstance -ClassName Win32_Process -Filter "Name='kdash.exe'" -ErrorAction Stop
        foreach ($process in $processes) {
            if ([string]::IsNullOrWhiteSpace($process.ExecutablePath)) { continue }
            try {
                $actual = Get-FullPath -Path $process.ExecutablePath
                if ([string]::Equals($actual, $expected, [System.StringComparison]::OrdinalIgnoreCase)) {
                    $result += $process
                }
            } catch { }
        }
    } catch {
        # Get-Process is a fallback only; matching still requires the full path.
        foreach ($process in @(Get-Process -Name "kdash" -ErrorAction SilentlyContinue)) {
            try {
                if ([string]::Equals((Get-FullPath -Path $process.Path), $expected,
                        [System.StringComparison]::OrdinalIgnoreCase)) {
                    $result += $process
                }
            } catch { }
        }
    }
    return $result
}

function Get-KdashTask {
    return @(Get-ScheduledTask -TaskName $script:TaskName -TaskPath $script:TaskPath -ErrorAction SilentlyContinue |
        Select-Object -First 1)
}

function Assert-KdashTaskPath {
    param(
        [Parameter(Mandatory = $true)]$Task,
        [Parameter(Mandatory = $true)][string]$ExePath
    )
    $expected = Get-FullPath -Path $ExePath
    $matches = 0
    foreach ($action in @($Task.Actions)) {
        if ([string]::IsNullOrWhiteSpace($action.Execute)) { continue }
        try {
            $actual = Get-FullPath -Path $action.Execute.Trim('"')
            if ([string]::Equals($actual, $expected, [System.StringComparison]::OrdinalIgnoreCase)) {
                $matches++
            }
        } catch { }
    }
    if ($matches -eq 0) {
        throw "The scheduled task $script:TaskName points to a different executable; refusing to modify it."
    }
}

function Stop-KdashRuntime {
    param(
        [Parameter(Mandatory = $true)][string]$ExePath,
        [Parameter(Mandatory = $true)][string]$ConfigPath,
        [switch]$RequireTaskPathMatch
    )
    $task = Get-KdashTask
    if ($null -ne $task -and $RequireTaskPathMatch) {
        Assert-KdashTaskPath -Task $task -ExePath $ExePath
    }

    $matching = @(Get-KdashProcesses -ExePath $ExePath)
    if ($matching.Count -eq 0 -and $null -eq $task) { return }

    # Call shutdown only when this installation has a matching live process.
    # This avoids sending a token or shutdown request to another installation.
    if ($matching.Count -gt 0) {
        $config = Read-KdashConfig -Path $ConfigPath
        $baseUrl = Get-KdashBaseUrl -Config $config -ForceLoopback
        if ($null -ne $baseUrl) {
            $token = Get-ConfigValue -Config $config -Name "settings_token" -Default ""
            try {
                Invoke-WebRequest -UseBasicParsing -Method Post -Uri "$baseUrl/api/shutdown" `
                    -Headers @{ "X-KDash-Token" = $token; "X-KDash-Client" = "kdash" } -TimeoutSec 5 | Out-Null
            } catch {
                # Older releases do not expose /api/shutdown. Exact-path fallback follows.
            }
        }
        $deadline = [DateTime]::UtcNow.AddSeconds(10)
        do {
            $matching = @(Get-KdashProcesses -ExePath $ExePath)
            if ($matching.Count -eq 0) { break }
            Start-Sleep -Milliseconds 500
        } while ([DateTime]::UtcNow -lt $deadline)
    }

    if ($null -ne $task) {
        Assert-KdashTaskPath -Task $task -ExePath $ExePath
        Stop-ScheduledTask -TaskName $script:TaskName -TaskPath $script:TaskPath -ErrorAction SilentlyContinue | Out-Null
    }

    # Process termination is restricted to this complete ExePath.
    foreach ($process in @(Get-KdashProcesses -ExePath $ExePath)) {
        $processId = if ($null -ne $process.PSObject.Properties["ProcessId"]) { $process.ProcessId } else { $process.Id }
        Stop-Process -Id $processId -Force -ErrorAction SilentlyContinue
    }
    Start-Sleep -Milliseconds 500
    if (@(Get-KdashProcesses -ExePath $ExePath).Count -gt 0) {
        throw "The kdash process at $ExePath is still running."
    }
}

function Wait-KdashHealth {
    param(
        [Parameter(Mandatory = $true)][string]$ConfigPath,
        [Parameter(Mandatory = $true)][string]$ExePath
    )
    $config = Read-KdashConfig -Path $ConfigPath
    if ($null -eq $config) { throw "Configuration could not be read for the health check." }
    $expectedVersion = (& $ExePath -version)
    if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($expectedVersion)) {
        throw "The installed kdash executable did not return a version."
    }
    $baseUrl = Get-KdashBaseUrl -Config $config
    if ($null -eq $baseUrl) { throw "The configured listen address is invalid for a health check." }
    for ($attempt = 0; $attempt -lt 45; $attempt++) {
        try {
            $response = Invoke-WebRequest -UseBasicParsing -Method Get -Uri "$baseUrl/api/status" -TimeoutSec 3
            $status = $response.Content | ConvertFrom-Json
            if ([string]$status.version -ne [string]$expectedVersion) {
                throw "The health endpoint returned a different kdash version."
            }
            Write-Host "Health check passed: $baseUrl/api/status (version $expectedVersion)"
            return
        } catch {
            Start-Sleep -Seconds 1
        }
    }
    throw "Health check failed at $baseUrl/api/status after 45 seconds."
}

function New-RandomToken {
    $bytes = New-Object byte[] 32
    $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    try { $rng.GetBytes($bytes) } finally { $rng.Dispose() }
    return ([System.BitConverter]::ToString($bytes)).Replace("-", "").ToLowerInvariant()
}

function Write-JsonWithoutBom {
    param(
        [Parameter(Mandatory = $true)]$Config,
        [Parameter(Mandatory = $true)][string]$Path
    )
    $json = $Config | ConvertTo-Json -Depth 30
    $encoding = New-Object System.Text.UTF8Encoding -ArgumentList $false
    [System.IO.File]::WriteAllText($Path, $json + [Environment]::NewLine, $encoding)
}

function New-InitialConfig {
    param(
        [Parameter(Mandatory = $true)][string]$ExamplePath,
        [Parameter(Mandatory = $true)][string]$DestinationPath,
        [Parameter(Mandatory = $true)][string]$DataDir
    )
    $config = Get-Content -LiteralPath $ExamplePath -Raw -Encoding UTF8 | ConvertFrom-Json
    $token = New-RandomToken
    if ($null -eq $config.PSObject.Properties["settings_token"]) {
        $config | Add-Member -NotePropertyName "settings_token" -NotePropertyValue $token
    } else {
        $config.settings_token = $token
    }
    $cachePath = Join-Path $DataDir "cache_weather.json"
    if ($null -eq $config.PSObject.Properties["weather_cache_path"]) {
        $config | Add-Member -NotePropertyName "weather_cache_path" -NotePropertyValue $cachePath
    } else {
        $config.weather_cache_path = $cachePath
    }
    Write-JsonWithoutBom -Config $config -Path $DestinationPath
}

function Register-KdashResidentTask {
    param(
        [Parameter(Mandatory = $true)][string]$ExePath,
        [Parameter(Mandatory = $true)][string]$ConfigPath,
        [Parameter(Mandatory = $true)][string]$WorkDir
    )
    $identity = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
    $arguments = '-config "{0}"' -f $ConfigPath
    $action = New-ScheduledTaskAction -Execute $ExePath -Argument $arguments -WorkingDirectory $WorkDir
    $trigger = New-ScheduledTaskTrigger -AtLogOn -User $identity
    $principal = New-ScheduledTaskPrincipal -UserId $identity -LogonType Interactive -RunLevel Limited
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries `
        -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1) `
        -ExecutionTimeLimit ([TimeSpan]::Zero)
    Register-ScheduledTask -TaskName $script:TaskName -TaskPath $script:TaskPath `
        -Action $action -Trigger $trigger -Principal $principal -Settings $settings `
        -Description "kdash current-user logon task; not an SCM service" -Force | Out-Null
}

function Download-KdashFile {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][string]$Destination
    )
    for ($attempt = 1; $attempt -le 4; $attempt++) {
        try {
            if (Test-Path -LiteralPath $Destination) { Remove-Item -LiteralPath $Destination -Force }
            if ($null -ne $script:LocalReleaseBase) {
                Copy-Item -LiteralPath (Join-Path $script:LocalReleaseBase $Name) -Destination $Destination -Force
            } else {
                $uri = $script:RemoteReleaseBase.TrimEnd('/') + '/' + $Name
                Invoke-WebRequest -UseBasicParsing -Uri $uri -OutFile $Destination -TimeoutSec 300
            }
            return
        } catch {
            if ($attempt -eq 4) { throw "Could not download $Name after four attempts." }
            Start-Sleep -Seconds ($attempt * 2)
        }
    }
}

function Test-ReleaseZip {
    param([Parameter(Mandatory = $true)][string]$ZipPath)
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $archive = [System.IO.Compression.ZipFile]::OpenRead($ZipPath)
    try {
        $prefix = "kdash-windows-amd64/"
        foreach ($entry in $archive.Entries) {
            $name = $entry.FullName.Replace('\', '/')
            if (-not $name.StartsWith($prefix, [System.StringComparison]::OrdinalIgnoreCase)) {
                throw "Release ZIP contains an entry outside its top-level directory."
            }
            foreach ($part in $name.Split('/')) {
                if ($part -eq "..") { throw "Release ZIP contains parent traversal." }
            }
        }
    } finally {
        $archive.Dispose()
    }
}

function Backup-InstallFiles {
    param([Parameter(Mandatory = $true)][string]$InstallDir)
    $backupDir = Join-Path $script:TempDir "backup"
    New-Item -ItemType Directory -Path $backupDir -Force | Out-Null
    foreach ($name in $script:InstallFiles) {
        $source = Join-Path $InstallDir $name
        if (Test-Path -LiteralPath $source -PathType Leaf) {
            Copy-Item -LiteralPath $source -Destination (Join-Path $backupDir $name) -Force
            $script:BackedUp[$name] = $true
        }
    }
}

function Restore-InstallFiles {
    param([Parameter(Mandatory = $true)][string]$InstallDir)
    $backupDir = Join-Path $script:TempDir "backup"
    foreach ($name in $script:InstallFiles) {
        $destination = Join-Path $InstallDir $name
        if ($script:BackedUp.ContainsKey($name)) {
            Copy-Item -LiteralPath (Join-Path $backupDir $name) -Destination $destination -Force
        } elseif (Test-Path -LiteralPath $destination) {
            Remove-Item -LiteralPath $destination -Force
        }
    }
}

function Remove-KdashInstallation {
    param([Parameter(Mandatory = $true)][string]$InstallDir)
    $exePath = Join-Path $InstallDir "kdash.exe"
    $configPath = Join-Path $InstallDir "kdash.json"
    Stop-KdashRuntime -ExePath $exePath -ConfigPath $configPath -RequireTaskPathMatch
    $task = Get-KdashTask
    if ($null -ne $task) {
        Assert-KdashTaskPath -Task $task -ExePath $exePath
        Unregister-ScheduledTask -TaskName $script:TaskName -TaskPath $script:TaskPath -Confirm:$false | Out-Null
    }
    if ($KeepConfig) {
        foreach ($name in $script:InstallFiles) {
            if ($name -eq "kdash.json") { continue }
            $path = Join-Path $InstallDir $name
            if (Test-Path -LiteralPath $path) { Remove-Item -LiteralPath $path -Force }
        }
        foreach ($name in @("assets", "logs")) {
            $path = Join-Path $InstallDir $name
            if (Test-Path -LiteralPath $path) { Remove-Item -LiteralPath $path -Recurse -Force }
        }
    } else {
        foreach ($name in $script:InstallFiles) {
            $path = Join-Path $InstallDir $name
            if (Test-Path -LiteralPath $path) { Remove-Item -LiteralPath $path -Force }
        }
        foreach ($name in @("assets", "data", "logs")) {
            $path = Join-Path $InstallDir $name
            if (Test-Path -LiteralPath $path) { Remove-Item -LiteralPath $path -Recurse -Force }
        }
    }
    # Only remove an empty installation directory; unrelated user files survive.
    if (Test-Path -LiteralPath $InstallDir) {
        try { Remove-Item -LiteralPath $InstallDir -Force -ErrorAction Stop } catch { }
    }
    Write-Host "kdash was uninstalled from $InstallDir"
    if ($KeepConfig) { Write-Host "Configuration and data were preserved in $InstallDir" }
}

function Install-Kdash {
    param([Parameter(Mandatory = $true)][string]$InstallDir)
    $script:TempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("kdash-install-" + [Guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $script:TempDir -Force | Out-Null
    $zipPath = Join-Path $script:TempDir "kdash-windows-amd64.zip"
    $sumsPath = Join-Path $script:TempDir "SHA256SUMS"
    $extractDir = Join-Path $script:TempDir "extracted"

    $script:Stage = "download"
    Download-KdashFile -Name "SHA256SUMS" -Destination $sumsPath
    Download-KdashFile -Name "kdash-windows-amd64.zip" -Destination $zipPath

    $script:Stage = "SHA-256 verification"
    $expected = $null
    foreach ($line in @(Get-Content -LiteralPath $sumsPath)) {
        $match = [regex]::Match($line, '^(?<hash>[0-9a-fA-F]{64})\s+\*?kdash-windows-amd64\.zip\s*$')
        if ($match.Success) { $expected = $match.Groups['hash'].Value.ToLowerInvariant(); break }
    }
    if ($null -eq $expected) { throw "SHA256SUMS has no entry for kdash-windows-amd64.zip." }
    $actual = (Get-FileHash -LiteralPath $zipPath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actual -ne $expected) { throw "SHA-256 verification failed for kdash-windows-amd64.zip." }

    $script:Stage = "archive inspection"
    Test-ReleaseZip -ZipPath $zipPath
    [System.IO.Compression.ZipFile]::ExtractToDirectory($zipPath, $extractDir)
    $sourceDir = Join-Path $extractDir "kdash-windows-amd64"
    foreach ($name in @("kdash.exe", "kdash.json.example", "LICENSE", "THIRD_PARTY_NOTICES.md", "README.md", "RELEASES.md", "start.bat", "stop.bat", "start_service.vbs", "install_task.bat", "install_task.ps1", "uninstall_task.bat", "uninstall_task.ps1", "assets/fonts/licenses/NotoSansSC-OFL.txt")) {
        if (-not (Test-Path -LiteralPath (Join-Path $sourceDir $name) -PathType Leaf)) {
            throw "Release archive is missing $name."
        }
    }

    $script:Stage = "backup and stop"
    $exePath = Join-Path $InstallDir "kdash.exe"
    $configPath = Join-Path $InstallDir "kdash.json"
    $dataDir = Join-Path $InstallDir "data"
    $logDir = Join-Path $InstallDir "logs"
    $script:HadDataDir = Test-Path -LiteralPath $dataDir
    $script:HadLogDir = Test-Path -LiteralPath $logDir
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Backup-InstallFiles -InstallDir $InstallDir

    $oldTask = Get-KdashTask
    if ($null -ne $oldTask) {
        Assert-KdashTaskPath -Task $oldTask -ExePath $exePath
        $script:HadTask = $true
        $script:OldTaskXml = Export-ScheduledTask -TaskName $script:TaskName -TaskPath $script:TaskPath
        $script:OldTaskWasRunning = $oldTask.State -eq "Running"
    }
    $script:TransactionActive = $true
    Stop-KdashRuntime -ExePath $exePath -ConfigPath $configPath

    $script:Stage = "file installation"
    New-Item -ItemType Directory -Path $dataDir -Force | Out-Null
    New-Item -ItemType Directory -Path $logDir -Force | Out-Null
    $newExe = Join-Path $InstallDir ".kdash.exe.new"
    Copy-Item -LiteralPath (Join-Path $sourceDir "kdash.exe") -Destination $newExe -Force
    Move-Item -LiteralPath $newExe -Destination $exePath -Force
    foreach ($name in $script:InstallFiles) {
        if ($name -eq "kdash.exe" -or $name -eq "kdash.json") { continue }
        Copy-Item -LiteralPath (Join-Path $sourceDir $name) -Destination (Join-Path $InstallDir $name) -Force
    }
    Copy-Item -LiteralPath (Join-Path $sourceDir "assets") -Destination $InstallDir -Recurse -Force
    if (-not (Test-Path -LiteralPath $configPath -PathType Leaf)) {
        $newConfig = Join-Path $script:TempDir "kdash.json"
        New-InitialConfig -ExamplePath (Join-Path $sourceDir "kdash.json.example") `
            -DestinationPath $newConfig -DataDir $dataDir
        Copy-Item -LiteralPath $newConfig -Destination $configPath -Force
    }

    $script:Stage = "scheduled task registration"
    Register-KdashResidentTask -ExePath $exePath -ConfigPath $configPath -WorkDir $InstallDir
    if (-not $NoStart) {
        $script:Stage = "startup and health check"
        Start-ScheduledTask -TaskName $script:TaskName -TaskPath $script:TaskPath
        Wait-KdashHealth -ConfigPath $configPath -ExePath $exePath
    }
    $script:TransactionActive = $false
    Write-Host "kdash was installed successfully at $exePath"
    Write-Host "Configuration: $configPath"
    Write-Host "The current-user, limited AtLogOn task is $script:TaskName; this is not an SCM service."
    if ($NoStart) { Write-Host "The task was registered but not started (--NoStart)." }
}

function Restore-FailedInstall {
    param([Parameter(Mandatory = $true)][string]$InstallDir)
    $exePath = Join-Path $InstallDir "kdash.exe"
    $configPath = Join-Path $InstallDir "kdash.json"
    try { Stop-KdashRuntime -ExePath $exePath -ConfigPath $configPath } catch { }
    try {
        $task = Get-KdashTask
        if ($null -ne $task) {
            Unregister-ScheduledTask -TaskName $script:TaskName -TaskPath $script:TaskPath -Confirm:$false -ErrorAction SilentlyContinue | Out-Null
        }
        if ($script:HadTask) {
            Register-ScheduledTask -TaskName $script:TaskName -TaskPath $script:TaskPath `
                -Xml $script:OldTaskXml -Force | Out-Null
        }
    } catch { }
    try { Restore-InstallFiles -InstallDir $InstallDir } catch { }
    if (-not $script:HadDataDir) {
        $path = Join-Path $InstallDir "data"
        if (Test-Path -LiteralPath $path) { Remove-Item -LiteralPath $path -Recurse -Force -ErrorAction SilentlyContinue }
    }
    if (-not $script:HadLogDir) {
        $path = Join-Path $InstallDir "logs"
        if (Test-Path -LiteralPath $path) { Remove-Item -LiteralPath $path -Recurse -Force -ErrorAction SilentlyContinue }
    }
    if ($script:HadTask -and $script:OldTaskWasRunning) {
        try { Start-ScheduledTask -TaskName $script:TaskName -TaskPath $script:TaskPath } catch { }
    }
}

try {
    if ([Environment]::Is64BitOperatingSystem -ne $true) { throw "Windows 11 x64 is required." }
    if ([string]::IsNullOrWhiteSpace($Prefix)) {
        $localAppData = [Environment]::GetFolderPath([Environment+SpecialFolder]::LocalApplicationData)
        $Prefix = Join-Path $localAppData "kdash"
    }
    $installDir = Get-FullPath -Path $Prefix
    $root = [System.IO.Path]::GetPathRoot($installDir).TrimEnd('\')
    if ($installDir -eq $root) { throw "Prefix must not be a drive root." }

    if ($Uninstall) {
        Remove-KdashInstallation -InstallDir $installDir
        exit 0
    }

    if ($Version -ne "latest" -and $Version -notmatch '^v[0-9]+\.[0-9]+\.[0-9]+$') {
        throw "Version must be latest or vX.Y.Z."
    }
    if ([string]::IsNullOrWhiteSpace($ReleaseBase)) {
        if ([string]::IsNullOrWhiteSpace($Repo)) {
            throw "Repository is empty; specify -Repo OWNER/REPO or -ReleaseBase."
        }
        if ($Repo -notmatch '^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$') { throw "Repo must have the form OWNER/REPO." }
        if ($Version -eq "latest") {
            $script:RemoteReleaseBase = "https://github.com/$Repo/releases/latest/download"
        } else {
            $script:RemoteReleaseBase = "https://github.com/$Repo/releases/download/$Version"
        }
    } elseif ($ReleaseBase -match '^https?://') {
        $script:RemoteReleaseBase = $ReleaseBase.TrimEnd('/')
    } else {
        if ($ReleaseBase -match '^file://') {
            $uri = [Uri]$ReleaseBase
            $script:LocalReleaseBase = $uri.LocalPath
        } else {
            $script:LocalReleaseBase = Get-FullPath -Path $ReleaseBase
        }
        if (-not (Test-Path -LiteralPath $script:LocalReleaseBase -PathType Container)) {
            throw "Local release directory does not exist: $script:LocalReleaseBase"
        }
    }

    Install-Kdash -InstallDir $installDir
} catch {
    if ($script:TransactionActive -and $null -ne $script:TempDir) {
        Restore-FailedInstall -InstallDir $installDir
    }
    if ($script:Stage -eq "initialization") {
        throw
    }
    throw "kdash installation failed during $script:Stage; rollback was attempted. No configuration values were logged."
} finally {
    if ($null -ne $script:TempDir -and (Test-Path -LiteralPath $script:TempDir)) {
        Remove-Item -LiteralPath $script:TempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}
