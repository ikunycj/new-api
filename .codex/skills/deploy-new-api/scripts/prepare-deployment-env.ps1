[CmdletBinding()]
param(
    [string]$Repository = 'F:\Project\My-Project\new-api',
    [Parameter(Mandatory)]
    [string]$DeploymentAlias,
    [string]$ApproveSha256 = ''
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Get-Sha256 {
    param([Parameter(Mandatory)] [string]$Path)

    return (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()
}

function New-HexSecret {
    $bytes = [byte[]]::new(32)
    [Security.Cryptography.RandomNumberGenerator]::Fill($bytes)
    return [Convert]::ToHexString($bytes).ToLowerInvariant()
}

function Protect-LocalFile {
    param([Parameter(Mandatory)] [string]$Path)

    if (-not $IsWindows) {
        return $false
    }

    $identity = [Security.Principal.WindowsIdentity]::GetCurrent().Name
    $permission = '{0}:F' -f $identity
    & icacls $Path /inheritance:r /grant:r $permission | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw "Could not restrict ACL for $Path"
    }
    return $true
}

function Read-EnvMap {
    param([Parameter(Mandatory)] [string]$Path)

    $values = [ordered]@{}
    foreach ($line in [IO.File]::ReadAllLines($Path)) {
        if ([string]::IsNullOrWhiteSpace($line) -or $line.TrimStart().StartsWith('#')) {
            continue
        }
        if ($line -notmatch '^([A-Z][A-Z0-9_]*)=(.*)$') {
            throw "Invalid env syntax in ${Path}; values were not displayed"
        }
        $key = $Matches[1]
        $value = $Matches[2]
        if ($values.Contains($key)) {
            throw "Duplicate env key in ${Path}: $key"
        }
        if ($value -match '[\s\x00-\x1f\x7f"''`$();|&<>\\]') {
            throw "Unsafe shell characters in env key: $key"
        }
        $values[$key] = $value
    }
    return $values
}

if (
    [string]::IsNullOrWhiteSpace($DeploymentAlias) -or
    $DeploymentAlias -notmatch '^[A-Za-z0-9][A-Za-z0-9._-]*$'
) {
    throw "Invalid deployment SSH alias: $DeploymentAlias"
}

$repoRoot = (Resolve-Path -LiteralPath $Repository).Path
$deployRoot = [IO.Path]::GetFullPath((Join-Path $repoRoot 'deploy'))
$deploymentDirectory = [IO.Path]::GetFullPath((Join-Path $deployRoot $DeploymentAlias))
$deployRootWithSeparator = $deployRoot.TrimEnd([IO.Path]::DirectorySeparatorChar, [IO.Path]::AltDirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
if (-not $deploymentDirectory.StartsWith($deployRootWithSeparator, [StringComparison]::OrdinalIgnoreCase)) {
    throw "Deployment path escapes deploy directory: $deploymentDirectory"
}
if (-not (Test-Path -LiteralPath $deploymentDirectory -PathType Container)) {
    throw "Missing deployment directory for SSH alias ${DeploymentAlias}: $deploymentDirectory"
}
$deploymentItem = Get-Item -LiteralPath $deploymentDirectory
if (($deploymentItem.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) {
    throw "Deployment directory must not be a symlink or reparse point: $deploymentDirectory"
}

$templatePath = Join-Path $deploymentDirectory '.env.example'
$envPath = Join-Path $deploymentDirectory '.env'
$approvalPath = Join-Path $deploymentDirectory '.env.approved.sha256'
if (-not (Test-Path -LiteralPath $templatePath -PathType Leaf)) {
    throw "Missing deployment env template: $templatePath"
}

$ignored = & git -C $repoRoot check-ignore --quiet -- $envPath
if ($LASTEXITCODE -ne 0) {
    throw "Deployment env must be ignored by Git: $envPath"
}
& git -C $repoRoot ls-files --error-unmatch -- $envPath 2>$null | Out-Null
if ($LASTEXITCODE -eq 0) {
    throw "Deployment env is tracked by Git: $envPath"
}

$beforeExists = Test-Path -LiteralPath $envPath -PathType Leaf
$beforeLength = if ($beforeExists) { (Get-Item -LiteralPath $envPath).Length } else { 0 }
$beforeSha = if ($beforeExists) { Get-Sha256 -Path $envPath } else { '' }
$created = $false

$secretKeys = @(
    'POSTGRES_ADMIN_PASSWORD',
    'POSTGRES_APP_PASSWORD',
    'REDIS_PASSWORD',
    'SESSION_SECRET',
    'CRYPTO_SECRET'
)

if (-not $beforeExists) {
    $lines = [IO.File]::ReadAllLines($templatePath)
    $seenSecrets = [Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
    for ($index = 0; $index -lt $lines.Length; $index++) {
        if ($lines[$index] -match '^([A-Z][A-Z0-9_]*)=(.*)$' -and $secretKeys -contains $Matches[1]) {
            $key = $Matches[1]
            $lines[$index] = "$key=$(New-HexSecret)"
            [void]$seenSecrets.Add($key)
        }
    }
    if ($seenSecrets.Count -ne $secretKeys.Count) {
        throw 'The env template does not contain every required generated secret key'
    }

    $temporaryPath = Join-Path $deploymentDirectory ('.env.tmp.{0}.{1}' -f $PID, [Guid]::NewGuid().ToString('N'))
    try {
        $content = ($lines -join "`n") + "`n"
        $utf8NoBom = [Text.UTF8Encoding]::new($false)
        $stream = [IO.File]::Open($temporaryPath, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None)
        try {
            $bytes = $utf8NoBom.GetBytes($content)
            $stream.Write($bytes, 0, $bytes.Length)
            $stream.Flush($true)
        }
        finally {
            $stream.Dispose()
        }
        [void](Protect-LocalFile -Path $temporaryPath)
        Move-Item -LiteralPath $temporaryPath -Destination $envPath
        $created = $true
    }
    finally {
        if (Test-Path -LiteralPath $temporaryPath) {
            Remove-Item -LiteralPath $temporaryPath -Force
        }
    }
}

$envItem = Get-Item -LiteralPath $envPath
if (($envItem.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) {
    throw "Deployment env must not be a symlink or reparse point: $envPath"
}
$aclRestricted = Protect-LocalFile -Path $envPath
$templateValues = Read-EnvMap -Path $templatePath
$envValues = Read-EnvMap -Path $envPath

# The Ikun target is a fresh, isolated install.  The generated file is still
# operator-reviewed, but changing topology values after generation must not
# silently retarget an existing service, volume, port, or database.  Keep
# these assertions in the local preparation gate as well as the remote
# bootstrap gate so a bad review is caught before upload.
if ($DeploymentAlias -eq 'ikun.love') {
    $fixedValues = [ordered]@{
        COMPOSE_PROJECT_NAME = 'ikun-new-api'
        NETWORK_NAME = 'ikun-new-api-network'
        POSTGRES_VOLUME = 'ikun-new-api-postgres-data'
        REDIS_VOLUME = 'ikun-new-api-redis-data'
        APP_PORT = '3000'
        METRICS_HOST_PORT = '8006'
        METRICS_PORT = '8006'
        IMAGE_TAG = 'new-api:ikun'
        CONTAINER = 'ikun-new-api'
        POSTGRES_IMAGE = 'postgres:18.4-alpine'
        POSTGRES_CONTAINER = 'ikun-new-api-postgres'
        POSTGRES_ADMIN_USER = 'ikun_pg_admin'
        POSTGRES_ADMIN_DB = 'postgres'
        POSTGRES_DB = 'new_api'
        POSTGRES_APP_USER = 'new_api_app'
        REDIS_IMAGE = 'redis:8.8.0'
        REDIS_CONTAINER = 'ikun-new-api-redis'
        REDIS_DB = '0'
        NODE_NAME = 'ikun-new-api-1'
        NODE_TYPE = 'master'
        METRICS_BIND_ADDRESS = '0.0.0.0'
        TRUSTED_REDIRECT_DOMAINS = 'ikun.love'
        TZ = 'Asia/Shanghai'
    }
    foreach ($key in $fixedValues.Keys) {
        if (-not $envValues.Contains($key) -or [string]$envValues[$key] -cne [string]$fixedValues[$key]) {
            throw "Ikun deployment env has an unexpected value for $key"
        }
    }
}

foreach ($key in $templateValues.Keys) {
    if (-not $envValues.Contains($key)) {
        throw "Missing env key: $key"
    }
}
foreach ($key in $envValues.Keys) {
    if (-not $templateValues.Contains($key)) {
        throw "Unknown env key: $key"
    }
}

$secretValues = [Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
foreach ($key in $secretKeys) {
    $value = [string]$envValues[$key]
    if ($value -notmatch '^[0-9a-fA-F]{64}$') {
        throw "$key must be a 64-character hexadecimal value"
    }
    if (-not $secretValues.Add($value)) {
        throw 'Generated deployment secrets must be distinct'
    }
}

$envSha = Get-Sha256 -Path $envPath
$approved = $false
if (Test-Path -LiteralPath $approvalPath) {
    $approvalItem = Get-Item -LiteralPath $approvalPath
    if (($approvalItem.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) {
        throw "Approval record must not be a symlink or reparse point: $approvalPath"
    }
}
if (-not [string]::IsNullOrWhiteSpace($ApproveSha256)) {
    $approval = $ApproveSha256.Trim().ToLowerInvariant()
    if ($approval -notmatch '^[0-9a-f]{64}$') {
        throw 'ApproveSha256 must be a SHA-256 value'
    }
    if ($approval -ne $envSha) {
        throw 'Approval SHA does not match the current deployment env'
    }
    [IO.File]::WriteAllText($approvalPath, "$envSha`n", [Text.UTF8Encoding]::new($false))
    [void](Protect-LocalFile -Path $approvalPath)
    $approved = $true
}
elseif (Test-Path -LiteralPath $approvalPath -PathType Leaf) {
    $recordedApproval = [IO.File]::ReadAllText($approvalPath).Trim().ToLowerInvariant()
    $approved = $recordedApproval -eq $envSha
}

[pscustomobject]@{
    Status = if ($created) { 'created' } else { 'existing-valid' }
    DeploymentAlias = $DeploymentAlias
    Path = $envPath
    BeforeExists = $beforeExists
    BeforeLength = $beforeLength
    BeforeSha256 = $beforeSha
    SizeBytes = $envItem.Length
    LastWriteTimeUtc = $envItem.LastWriteTimeUtc.ToString('o')
    Sha256 = $envSha
    TemplateSha256 = Get-Sha256 -Path $templatePath
    KeyCount = $envValues.Count
    AclRestricted = $aclRestricted
    ReviewRequired = -not $approved
    ReadyForUpload = $approved
}
