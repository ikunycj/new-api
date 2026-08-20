[CmdletBinding(DefaultParameterSetName = 'Emails')]
param(
    [Parameter(Mandatory = $true, ParameterSetName = 'Emails')]
    [ValidateNotNullOrEmpty()]
    [string[]]$Email,
    [Parameter(Mandatory = $true, ParameterSetName = 'All')]
    [switch]$All,
    [Parameter(Mandatory = $true, ParameterSetName = 'Snapshot')]
    [ValidateScript({ Test-Path -LiteralPath $_ -PathType Leaf })]
    [string]$SnapshotPath,
    [Parameter(ParameterSetName = 'All')]
    [ValidateNotNullOrEmpty()]
    [string]$SnapshotOutputPath,
    [ValidateNotNullOrEmpty()]
    [string]$ReportPath,
    [switch]$Apply,
    [switch]$DetailedReport
)

$ErrorActionPreference = 'Stop'

$sourceAlias = 'ikun.love-sub2api'
$targetAlias = 'ikun.love'
$sourceContainer = '1Panel-postgresql-8Kr6'
$sourceDatabase = 'sub2api'
$sourceRole = 'user_ZTNTJM'
$targetContainer = 'ikun-new-api-postgres'
$targetDatabase = 'new_api'
$targetRole = 'new_api_app'
$quotaPerCny = [decimal]500000
$migrationStamp = Get-Date -Format 'yyyyMMdd-HHmmss'

function ConvertTo-Base64Utf8([string]$Value) {
    if ($null -eq $Value) { return $null }
    return [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($Value))
}

function ConvertFrom-Base64Utf8([string]$Value) {
    if ([string]::IsNullOrEmpty($Value)) { return '' }
    return [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($Value))
}

function Normalize-Email([string]$Address) {
    if ($null -eq $Address) { return '' }
    return $Address.Trim().ToLowerInvariant()
}

function Mask-Email([string]$Address) {
    $normalized = Normalize-Email $Address
    $at = $normalized.IndexOf('@')
    if ($at -le 0 -or $at -eq ($normalized.Length - 1)) { return '[invalid-email]' }
    $local = $normalized.Substring(0, $at)
    $domain = $normalized.Substring($at + 1)
    $prefix = if ($local.Length -le 1) { '*' } else { $local.Substring(0, 1) }
    return "$prefix***@$domain"
}

function Get-EmailFingerprint([string]$Address) {
    $bytes = [Text.Encoding]::UTF8.GetBytes((Normalize-Email $Address))
    $sha = [Security.Cryptography.SHA256]::Create()
    try {
        return ([Convert]::ToHexString($sha.ComputeHash($bytes))).ToLowerInvariant().Substring(0, 12)
    } finally {
        $sha.Dispose()
    }
}

function Get-ReportEmail([string]$Address) {
    # Full migration reports must not become an export of the source address list.
    return "email:$((Get-EmailFingerprint $Address))"
}

function Invoke-Ssh([string]$Alias, [string]$RemoteCommand, [string]$InputText = '') {
    $startInfo = [Diagnostics.ProcessStartInfo]::new()
    $startInfo.FileName = 'ssh'
    [void]$startInfo.ArgumentList.Add('-T')
    [void]$startInfo.ArgumentList.Add('-o')
    [void]$startInfo.ArgumentList.Add('BatchMode=yes')
    [void]$startInfo.ArgumentList.Add('-o')
    [void]$startInfo.ArgumentList.Add('ConnectTimeout=15')
    [void]$startInfo.ArgumentList.Add($Alias)
    [void]$startInfo.ArgumentList.Add($RemoteCommand)
    $startInfo.RedirectStandardInput = $true
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true
    $startInfo.UseShellExecute = $false
    $process = [Diagnostics.Process]::new()
    $process.StartInfo = $startInfo
    [void]$process.Start()
    if ($InputText) {
        $process.StandardInput.Write($InputText)
    }
    $process.StandardInput.Close()
    $stdout = $process.StandardOutput.ReadToEnd()
    $stderr = $process.StandardError.ReadToEnd()
    $process.WaitForExit()
    if ($process.ExitCode -ne 0) {
        throw "ssh $Alias failed: $($stderr.Trim())"
    }
    return $stdout.Trim()
}

function Invoke-SourceSql([string]$Sql) {
    $command = "sudo docker exec -i $sourceContainer psql -q -v ON_ERROR_STOP=1 -U $sourceRole -d $sourceDatabase -At -F '|' -P pager=off"
    return Invoke-Ssh $sourceAlias $command $Sql
}

function Invoke-TargetSql([string]$Sql) {
    $command = "sudo docker exec -i $targetContainer psql -q -v ON_ERROR_STOP=1 -U $targetRole -d $targetDatabase -At -F '|' -P pager=off"
    return Invoke-Ssh $targetAlias $command $Sql
}

function SqlText([string]$Value) {
    if ($null -eq $Value) { return 'NULL' }
    $encoded = ConvertTo-Base64Utf8 $Value
    return "convert_from(decode('$encoded','base64'),'UTF8')"
}

function SqlNumber([object]$Value) {
    if ($null -eq $Value -or [string]::IsNullOrEmpty([string]$Value)) { return 'NULL' }
    return ([string]$Value).Replace(',', '.')
}

function Get-SourceUser([string]$Address) {
    $emailSql = $Address.Replace("'", "''")
    $sql = @"
select encode(convert_to(json_build_object(
  'user', json_build_object(
    'id', u.id,
    'email', u.email,
    'username', coalesce(u.username,''),
    'password_hash', u.password_hash,
    'notes', coalesce(u.notes,''),
    'balance', u.balance::text,
    'frozen_balance', u.frozen_balance::text,
    'status', u.status,
    'role', u.role,
    'created_at', extract(epoch from u.created_at)::bigint
  ),
  'keys', coalesce((select json_agg(json_build_object(
    'id', k.id,
    'name', coalesce(k.name,''),
    'key', k.key,
    'status', k.status,
    'quota', k.quota::text,
    'quota_used', k.quota_used::text,
    'ip_whitelist', coalesce(k.ip_whitelist, '[]'::jsonb),
    'ip_blacklist', coalesce(k.ip_blacklist, '[]'::jsonb),
    'rate_limit_5h', k.rate_limit_5h::text,
    'rate_limit_1d', k.rate_limit_1d::text,
    'rate_limit_7d', k.rate_limit_7d::text,
    'created_at', extract(epoch from k.created_at)::bigint,
    'expires_at', extract(epoch from k.expires_at)::bigint,
    'group', g.name
  ) order by k.id) from api_keys k left join groups g on g.id = k.group_id
    where k.user_id = u.id and k.deleted_at is null), '[]'::json),
  'subscriptions', coalesce((select json_agg(json_build_object(
    'id', s.id,
    'status', s.status,
    'starts_at', extract(epoch from s.starts_at)::bigint,
    'expires_at', extract(epoch from s.expires_at)::bigint,
    'daily_usage', s.daily_usage_usd::text,
    'weekly_usage', s.weekly_usage_usd::text,
    'monthly_usage', s.monthly_usage_usd::text,
    'daily_window_start', extract(epoch from s.daily_window_start)::bigint,
    'weekly_window_start', extract(epoch from s.weekly_window_start)::bigint,
    'monthly_window_start', extract(epoch from s.monthly_window_start)::bigint,
    'daily_limit', g.daily_limit_usd::text,
    'weekly_limit', g.weekly_limit_usd::text,
    'monthly_limit', g.monthly_limit_usd::text,
    'group', g.name,
    'subscription_type', coalesce(g.subscription_type,'')
  ) order by s.id) from user_subscriptions s join groups g on g.id = s.group_id
    where s.user_id = u.id and s.deleted_at is null), '[]'::json))::text,'UTF8'),'base64')
from users u
where lower(trim(u.email)) = lower('$emailSql') and u.deleted_at is null;
"@
    $encoded = Invoke-SourceSql $sql
    if (-not $encoded) { return $null }
    $rows = @($encoded -split "`r?`n" | Where-Object { $_ } | ForEach-Object {
        ConvertFrom-Base64Utf8 $_ | ConvertFrom-Json -Depth 20
    })
    if ($rows.Count -eq 0) { return $null }
    if ($rows.Count -eq 1) { return $rows[0] }
    return $rows
}

function Get-SourceSnapshotSql {
    # One repeatable read query is materially cheaper than opening three SSH/psql
    # sessions per user. The result remains base64 encoded on stdout so neither
    # passwords nor API keys can be emitted accidentally by the shell transcript.
    return @"
select encode(convert_to(coalesce(json_agg(json_build_object(
  'user', json_build_object(
    'id', u.id,
    'email', u.email,
    'username', coalesce(u.username,''),
    'password_hash', u.password_hash,
    'notes', coalesce(u.notes,''),
    'balance', u.balance::text,
    'frozen_balance', u.frozen_balance::text,
    'status', u.status,
    'role', u.role,
    'created_at', extract(epoch from u.created_at)::bigint
  ),
  'keys', coalesce((select json_agg(json_build_object(
    'id', k.id,
    'name', coalesce(k.name,''),
    'key', k.key,
    'status', k.status,
    'quota', k.quota::text,
    'quota_used', k.quota_used::text,
    'ip_whitelist', coalesce(k.ip_whitelist, '[]'::jsonb),
    'ip_blacklist', coalesce(k.ip_blacklist, '[]'::jsonb),
    'rate_limit_5h', k.rate_limit_5h::text,
    'rate_limit_1d', k.rate_limit_1d::text,
    'rate_limit_7d', k.rate_limit_7d::text,
    'created_at', extract(epoch from k.created_at)::bigint,
    'expires_at', extract(epoch from k.expires_at)::bigint,
    'group', g.name
  ) order by k.id) from api_keys k left join groups g on g.id = k.group_id
    where k.user_id = u.id and k.deleted_at is null), '[]'::json),
  'subscriptions', coalesce((select json_agg(json_build_object(
    'id', s.id,
    'status', s.status,
    'starts_at', extract(epoch from s.starts_at)::bigint,
    'expires_at', extract(epoch from s.expires_at)::bigint,
    'daily_usage', s.daily_usage_usd::text,
    'weekly_usage', s.weekly_usage_usd::text,
    'monthly_usage', s.monthly_usage_usd::text,
    'daily_window_start', extract(epoch from s.daily_window_start)::bigint,
    'weekly_window_start', extract(epoch from s.weekly_window_start)::bigint,
    'monthly_window_start', extract(epoch from s.monthly_window_start)::bigint,
    'daily_limit', g.daily_limit_usd::text,
    'weekly_limit', g.weekly_limit_usd::text,
    'monthly_limit', g.monthly_limit_usd::text,
    'group', g.name,
    'subscription_type', coalesce(g.subscription_type,'')
  ) order by s.id) from user_subscriptions s join groups g on g.id = s.group_id
    where s.user_id = u.id and s.deleted_at is null), '[]'::json)
) order by u.id), '[]'::json)::text, 'UTF8'), 'base64')
from users u
where u.deleted_at is null and u.role = 'user';
"@
}

function Get-SourceSnapshot {
    $encoded = Invoke-SourceSql (Get-SourceSnapshotSql)
    if (-not $encoded) { return @() }
    $decoded = ConvertFrom-Base64Utf8 $encoded
    if ([string]::IsNullOrWhiteSpace($decoded)) { return @() }
    return @($decoded | ConvertFrom-Json -Depth 20)
}

function Read-SourceSnapshotFile([string]$Path) {
    $raw = Get-Content -LiteralPath $Path -Raw -ErrorAction Stop
    if ([string]::IsNullOrWhiteSpace($raw)) { throw "snapshot file is empty" }
    try {
        return @($raw | ConvertFrom-Json -Depth 20)
    } catch {
        throw "snapshot file is not valid JSON: $($_.Exception.Message)"
    }
}

function Write-SourceSnapshotFile([object[]]$Rows, [string]$Path) {
    $parent = Split-Path -Parent $Path
    if ($parent) { New-Item -ItemType Directory -Force -Path $parent | Out-Null }
    $Rows | ConvertTo-Json -Depth 20 | Set-Content -LiteralPath $Path -Encoding utf8NoBOM
    # Do not leave a snapshot with inherited broad access on Windows hosts.
    $icacls = Get-Command icacls.exe -ErrorAction SilentlyContinue
    if ($null -ne $icacls) {
        & $icacls.Source $Path /inheritance:r /grant:r "${env:USERNAME}:(F)" | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "failed to restrict snapshot ACL: $Path" }
    }
    return (Resolve-Path -LiteralPath $Path).Path
}

function Get-TargetState([string]$Address) {
    $emailSql = $Address.Replace("'", "''")
    $sql = @"
select coalesce(json_agg(json_build_object(
  'id', u.id,
  'email', u.email,
  'username', u.username,
  'status', u.status,
  'role', u.role,
  'quota', u.quota,
  'used_quota', u.used_quota,
  'token_count', (select count(*) from tokens t where t.user_id = u.id and t.deleted_at is null)
) order by u.id),'[]'::json)::text
from users u
where lower(trim(u.email)) = lower('$emailSql') and u.deleted_at is null;
"@
    $raw = Invoke-TargetSql $sql
    return ($raw | ConvertFrom-Json)
}

function Get-TargetGroupRatio {
    $raw = Invoke-TargetSql "select value from options where key='GroupRatio';"
    if (-not $raw) { return @{} }
    return ($raw | ConvertFrom-Json)
}

function Get-TargetMapping([int64]$SourceUserId) {
    $tableExists = Invoke-TargetSql "select to_regclass('public.sub2api_user_migration_mappings') is not null;"
    if ($tableExists -ne 't') { return $null }

    $raw = Invoke-TargetSql @"
select json_build_object(
  'source_user_id', source_user_id,
  'source_email', source_email,
  'target_user_id', target_user_id,
  'migration_batch_id', migration_batch_id,
  'imported_quota', imported_quota,
  'imported_token_count', imported_token_count
)::text
from sub2api_user_migration_mappings
where source_user_id = $SourceUserId;
"@
    if (-not $raw) { return $null }
    return ($raw | ConvertFrom-Json)
}

function Test-TargetMigrationTable {
    return (Invoke-TargetSql "select to_regclass('public.sub2api_user_migration_mappings') is not null;") -eq 't'
}

function Get-TargetPreflight([object[]]$Sources, [hashtable]$TokenRowsBySourceId, [bool]$MappingTableExists) {
    if ($Sources.Count -eq 0) {
        return [pscustomobject]@{ Users = @{}; Mappings = @{}; Keys = @{} }
    }

    $sourceValues = @($Sources | ForEach-Object {
        $id = [int64]$_.user.id
        $email = SqlText (Normalize-Email ([string]$_.user.email))
        "($id,$email)"
    }) -join ','
    $keyValues = @($TokenRowsBySourceId.Values | ForEach-Object {
        foreach ($token in @($_)) { "(" + (SqlText ([string]$token.Key)) + ")" }
    }) -join ','
    $keyCte = if ($keyValues) { "key_values(key) as (values $keyValues)," } else { "key_values(key) as (select null::text where false)," }
    $mappingSelect = if ($MappingTableExists) {
        "coalesce((select json_agg(json_build_object('source_id',m.source_user_id,'target_id',m.target_user_id,'quota',m.imported_quota,'tokens',m.imported_token_count)) from sub2api_user_migration_mappings m join source_users s on s.source_id=m.source_user_id),'[]'::json)"
    } else {
        "'[]'::json"
    }
    $sql = @"
with source_users(source_id,email) as (values $sourceValues),
$keyCte
target_users as (
  select u.id, lower(trim(u.email)) as email, u.quota, u.used_quota,
         (select count(*) from tokens t where t.user_id=u.id) as token_count
  from users u join source_users s on lower(trim(u.email))=s.email
), target_keys as (
  select t.key, t.user_id from tokens t join key_values k on k.key=t.key
)
select encode(convert_to(json_build_object(
  'users', coalesce((select json_agg(json_build_object('id',id,'email',email,'quota',quota,'used_quota',used_quota,'token_count',token_count)) from target_users),'[]'::json),
  'mappings', $mappingSelect,
  'keys', coalesce((select json_agg(json_build_object('key',target_keys.key,'user_id',target_keys.user_id)) from target_keys),'[]'::json)
)::text,'UTF8'),'base64');
"@
    $raw = Invoke-TargetSql $sql
    if (-not $raw) { return [pscustomobject]@{ Users = @{}; Mappings = @{}; Keys = @{} } }
    $state = ConvertFrom-Base64Utf8 $raw | ConvertFrom-Json
    $users = @{}
    foreach ($row in @($state.users)) {
        $email = Normalize-Email ([string]$row.email)
        if ($email) {
            if (-not $users.ContainsKey($email)) { $users[$email] = @() }
            $users[$email] += $row
        }
    }
    $mappings = @{}
    foreach ($row in @($state.mappings)) { $mappings[[string]$row.source_id] = $row }
    $keys = @{}
    foreach ($row in @($state.keys)) { $keys[[string]$row.key] = $row }
    return [pscustomobject]@{ Users = $users; Mappings = $mappings; Keys = $keys }
}

function Get-TargetTokenConflictCount([object[]]$TokenRows) {
    if ($TokenRows.Count -eq 0) { return 0 }
    $keyExpressions = @($TokenRows | ForEach-Object { SqlText $_.Key }) -join ','
    $raw = Invoke-TargetSql @"
select count(*)
from tokens
where key in ($keyExpressions);
"@
    return [int]$raw
}

function To-Decimal([object]$Value) {
    if ($null -eq $Value -or [string]::IsNullOrEmpty([string]$Value)) { return [decimal]0 }
    return [decimal]::Parse([string]$Value, [Globalization.CultureInfo]::InvariantCulture)
}

function Get-Quota([object]$Source) {
    $balance = [math]::Max([decimal]0, (To-Decimal $Source.user.balance))
    $totalUsd = $balance
    $now = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
    foreach ($subscription in @($Source.subscriptions)) {
        if ([string]$subscription.status -ne 'active') { continue }
        if ([string]$subscription.subscription_type -ne 'subscription') { continue }
        if ($null -ne $subscription.starts_at -and [int64]$subscription.starts_at -gt $now) { continue }
        if ($null -eq $subscription.expires_at -or [int64]$subscription.expires_at -le $now) { continue }
        $windowRemaining = @()
        foreach ($pair in @(
            @{ limit = $subscription.daily_limit; usage = $subscription.daily_usage; window_start = $subscription.daily_window_start; duration = 86400 },
            @{ limit = $subscription.weekly_limit; usage = $subscription.weekly_usage; window_start = $subscription.weekly_window_start; duration = 604800 },
            @{ limit = $subscription.monthly_limit; usage = $subscription.monthly_usage; window_start = $subscription.monthly_window_start; duration = 2592000 }
        )) {
            $limit = To-Decimal $pair.limit
            if ($limit -gt 0) {
                $usage = To-Decimal $pair.usage
                if ($null -ne $pair.window_start -and [int64]$pair.window_start + [int64]$pair.duration -le $now) {
                    $usage = [decimal]0
                }
                $remaining = [math]::Max([decimal]0, $limit - $usage)
                $windowRemaining += $remaining
            }
        }
        if ($windowRemaining.Count -gt 0) {
            $totalUsd += ($windowRemaining | Measure-Object -Minimum).Minimum
        }
    }
    return [int64][math]::Floor(($totalUsd / 5) * $quotaPerCny)
}

function Get-QuotaBreakdown([object]$Source) {
    $balance = [math]::Max([decimal]0, (To-Decimal $Source.user.balance))
    $subscriptionUsd = [decimal]0
    $activeSubscriptions = 0
    $issues = @()
    $now = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
    foreach ($subscription in @($Source.subscriptions)) {
        if ([string]$subscription.status -ne 'active') { continue }
        if ([string]$subscription.subscription_type -ne 'subscription') { continue }
        if ($null -ne $subscription.starts_at -and [int64]$subscription.starts_at -gt $now) { continue }
        if ($null -eq $subscription.expires_at -or [int64]$subscription.expires_at -le $now) { continue }
        $activeSubscriptions++
        $windowRemaining = @()
        foreach ($pair in @(
            @{ name = 'daily'; limit = $subscription.daily_limit; usage = $subscription.daily_usage; window_start = $subscription.daily_window_start; duration = 86400 },
            @{ name = 'weekly'; limit = $subscription.weekly_limit; usage = $subscription.weekly_usage; window_start = $subscription.weekly_window_start; duration = 604800 },
            @{ name = 'monthly'; limit = $subscription.monthly_limit; usage = $subscription.monthly_usage; window_start = $subscription.monthly_window_start; duration = 2592000 }
        )) {
            $limit = To-Decimal $pair.limit
            if ($limit -gt 0) {
                $usage = To-Decimal $pair.usage
                if ($null -ne $pair.window_start -and [int64]$pair.window_start + [int64]$pair.duration -le $now) {
                    $usage = [decimal]0
                }
                $windowRemaining += [math]::Max([decimal]0, $limit - $usage)
            }
        }
        if ($windowRemaining.Count -eq 0) {
            $issues += "subscription:$($subscription.id):no_numeric_window"
            continue
        }
        $subscriptionUsd += ($windowRemaining | Measure-Object -Minimum).Minimum
    }
    $totalUsd = $balance + $subscriptionUsd
    $quotaDecimal = [math]::Floor(($totalUsd / 5) * $quotaPerCny)
    if ($quotaDecimal -gt [int64]::MaxValue) { throw 'calculated quota exceeds int64 range' }
    return [pscustomobject]@{
        Quota = [int64]$quotaDecimal
        BalanceUsd = $balance
        SubscriptionUsd = $subscriptionUsd
        ActiveSubscriptions = $activeSubscriptions
        Issues = @($issues)
    }
}

function Resolve-TargetGroup([string]$SourceGroup, [object]$TargetRatios) {
    $mapping = @{
        'OpenAI官方转发' = 'ChatGPT官转'
        'ChatGPT 生图专用' = '生图'
        'ChatGPT Plus' = 'ChatGPT Plus'
        'ChatGPT Pro' = 'ChatGPT Pro'
        'Claude' = 'Claude Kiro'
        'Claude Max 20x' = 'Claude Max满血'
        'ChatGPT 羊毛福利' = '羊毛福利'
    }
    $candidate = if ($mapping.ContainsKey([string]$SourceGroup)) { $mapping[[string]$SourceGroup] } else { 'ChatGPT Plus' }
    $available = if ($TargetRatios -is [hashtable]) {
        @($TargetRatios.Keys | ForEach-Object { [string]$_ })
    } else {
        @($TargetRatios.PSObject.Properties.Name)
    }
    if ($available -contains $candidate) { return $candidate }
    if ($available -contains 'ChatGPT Plus') { return 'ChatGPT Plus' }
    throw "target group ChatGPT Plus is unavailable"
}

function Test-EmptyJsonList([object]$Value) {
    if ($null -eq $Value) { return $true }
    if ($Value -is [System.Array]) { return $Value.Count -eq 0 }
    $text = ([string]$Value).Trim()
    return [string]::IsNullOrEmpty($text) -or $text -eq '[]' -or $text -eq 'null'
}

function Assert-SupportedKeyPolicy([object]$Key) {
    if ((To-Decimal $Key.quota) -ne 0) {
        throw "source key $($Key.id) has an independent quota and needs a reviewed quota conversion"
    }
    if ((To-Decimal $Key.quota_used) -ne 0) {
        throw "source key $($Key.id) has prior usage and needs an explicit used-quota mapping"
    }
    if (-not (Test-EmptyJsonList $Key.ip_whitelist) -or -not (Test-EmptyJsonList $Key.ip_blacklist)) {
        throw "source key $($Key.id) has an IP policy that needs an explicit target mapping"
    }
    foreach ($field in @('rate_limit_5h', 'rate_limit_1d', 'rate_limit_7d')) {
        if ((To-Decimal $Key.$field) -ne 0) {
            throw "source key $($Key.id) has $field and needs an explicit target policy"
        }
    }
}

function Get-TokenRows([object]$Source, [object]$TargetRatios) {
    $rows = @()
    $now = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
    $seenKeys = [Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
    foreach ($key in @($Source.keys)) {
        $rawKey = [string]$key.key
        if ($rawKey -notmatch '^sk-[0-9a-fA-F]{64}$') {
            throw "source key $($key.id) has unsupported format"
        }
        Assert-SupportedKeyPolicy $key
        $normalizedKey = $rawKey.Substring(3)
        if (-not $seenKeys.Add($normalizedKey)) {
            throw "source user $($Source.user.id) has duplicate API key rows"
        }
        $expiredTime = if ($null -eq $key.expires_at) { -1 } else { [int64]$key.expires_at }
        $rows += [pscustomobject]@{
            Name = [string]$key.name
            Key = $normalizedKey
            Status = if ([string]$key.status -eq 'active' -and ($expiredTime -eq -1 -or $expiredTime -gt $now)) { 1 } else { 0 }
            CreatedTime = if ($key.created_at) { [int64]$key.created_at } else { [DateTimeOffset]::UtcNow.ToUnixTimeSeconds() }
            ExpiredTime = $expiredTime
            SourceGroup = [string]$key.group
            Group = Resolve-TargetGroup ([string]$key.group) $TargetRatios
        }
    }
    return $rows
}

function Get-TokenPlan([object]$Source, [object]$TargetRatios) {
    $rows = @()
    $issues = @()
    $aliases = @{}
    $now = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
    $seenKeys = [Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
    $mapping = @{
        'OpenAI官方转发' = 'ChatGPT官转'
        'ChatGPT 生图专用' = '生图'
        'ChatGPT Plus' = 'ChatGPT Plus'
        'ChatGPT Pro' = 'ChatGPT Pro'
        'Claude' = 'Claude Kiro'
        'Claude Max 20x' = 'Claude Max满血'
        'ChatGPT 羊毛福利' = '羊毛福利'
    }
    foreach ($key in @($Source.keys)) {
        $rawKey = ([string]$key.key).Trim()
        if ($rawKey -notmatch '^sk-[0-9a-fA-F]{64}$') {
            $issues += "key:$($key.id):unsupported_format"
            continue
        }
        try {
            Assert-SupportedKeyPolicy $key
        } catch {
            # Keep the diagnostic useful without ever echoing the secret value.
            $issues += "key:$($key.id):$($_.Exception.Message -replace '^.*? has ', '')"
            continue
        }
        $normalizedKey = $rawKey.Substring(3)
        if (-not $seenKeys.Add($normalizedKey)) {
            $issues += "key:$($key.id):duplicate"
            continue
        }
        $sourceGroup = [string]$key.group
        $candidate = if ($mapping.ContainsKey($sourceGroup)) { $mapping[$sourceGroup] } else { 'ChatGPT Plus' }
        try {
            $targetGroup = Resolve-TargetGroup $sourceGroup $TargetRatios
        } catch {
            $issues += "key:$($key.id):target_group_unavailable"
            continue
        }
        if (-not $mapping.ContainsKey($sourceGroup) -or $candidate -ne $targetGroup) {
            $alias = if ([string]::IsNullOrWhiteSpace($sourceGroup)) { '[no-source-group]' } else { $sourceGroup }
            $aliasKey = "$alias -> $targetGroup"
            if ($aliases.ContainsKey($aliasKey)) { $aliases[$aliasKey]++ } else { $aliases[$aliasKey] = 1 }
        }
        $expiredTime = if ($null -eq $key.expires_at) { -1 } else { [int64]$key.expires_at }
        $rows += [pscustomobject]@{
            Name = [string]$key.name
            Key = $normalizedKey
            Status = if ([string]$key.status -eq 'active' -and ($expiredTime -eq -1 -or $expiredTime -gt $now)) { 1 } else { 0 }
            CreatedTime = if ($key.created_at) { [int64]$key.created_at } else { [DateTimeOffset]::UtcNow.ToUnixTimeSeconds() }
            ExpiredTime = $expiredTime
            SourceGroup = $sourceGroup
            Group = $targetGroup
        }
    }
    return [pscustomobject]@{
        Rows = @($rows)
        Issues = @($issues)
        Aliases = $aliases
        SpecialCount = $issues.Count + $aliases.Count
    }
}

function Get-SourceFingerprint([object]$Source) {
    $canonical = $Source | ConvertTo-Json -Depth 12 -Compress
    $bytes = [Text.Encoding]::UTF8.GetBytes($canonical)
    $sha = [Security.Cryptography.SHA256]::Create()
    try {
        return ([Convert]::ToHexString($sha.ComputeHash($bytes))).ToLowerInvariant()
    } finally {
        $sha.Dispose()
    }
}

function Ensure-MigrationTable {
    Invoke-TargetSql @'
create table if not exists sub2api_user_migration_mappings (
  source_user_id bigint primary key,
  source_email text not null unique,
  target_user_id bigint not null unique references users(id),
  migration_batch_id text not null,
  source_fingerprint char(64) not null,
  imported_quota bigint not null,
  imported_token_count integer not null,
  created_at bigint not null
);
'@ | Out-Null
}

function Get-TargetFieldLimits {
    $raw = Invoke-TargetSql @"
select coalesce(json_object_agg(column_name, character_maximum_length), '{}'::json)::text
from information_schema.columns
where table_schema='public' and table_name='users'
  and column_name in ('username','email','remark');
"@
    if (-not $raw) { return @{} }
    $obj = $raw | ConvertFrom-Json
    $limits = @{}
    foreach ($property in $obj.PSObject.Properties) {
        if ($null -ne $property.Value) { $limits[$property.Name] = [int]$property.Value }
    }
    return $limits
}

function Get-SourceValidationIssues([object]$Source, [hashtable]$TargetFieldLimits) {
    $issues = @()
    if ($null -eq $Source.user) { return @('source_user_missing') }
    $email = Normalize-Email ([string]$Source.user.email)
    if (-not $email -or $email.IndexOf('@') -le 0) { $issues += 'invalid_email' }
    $username = ([string]$Source.user.username).Trim()
    if (-not $username) { $username = $email }
    if (-not $username) { $issues += 'username_unresolvable' }
    if ($TargetFieldLimits.ContainsKey('email') -and $email.Length -gt $TargetFieldLimits.email) { $issues += 'email_too_long' }
    if ($TargetFieldLimits.ContainsKey('username') -and $username.Length -gt $TargetFieldLimits.username) { $issues += 'username_too_long' }
    $remark = [string]$Source.user.notes
    if ($TargetFieldLimits.ContainsKey('remark') -and $remark.Length -gt $TargetFieldLimits.remark) { $issues += 'remark_too_long' }
    $password = [string]$Source.user.password_hash
    # Do not attempt to rehash or silently replace a source password.
    if ($password -notmatch '^\$2[aby]?\$[0-9]{2}\$[./A-Za-z0-9]{53}$') { $issues += 'password_hash_not_bcrypt' }
    return @($issues)
}

function Write-TargetAudit([object]$Item, [string]$Fingerprint, [object]$TargetBefore) {
    $targetAudit = @($TargetBefore | ForEach-Object {
        [pscustomobject]@{
            id = $_.id
            email_ref = Get-ReportEmail ([string]$_.email)
            quota = $_.quota
            used_quota = $_.used_quota
            token_count = $_.token_count
        }
    })
    $audit = [pscustomobject]@{
        batch_id = $migrationStamp
        source_user_id = [int64]$Item.Source.user.id
        source_email_ref = Get-ReportEmail ([string]$Item.Source.user.email)
        source_fingerprint = $Fingerprint
        calculated_quota = [int64]$Item.Quota
        token_count = $Item.Tokens.Count
        token_group_mappings = @($Item.Tokens | ForEach-Object {
            [pscustomobject]@{ source = $_.SourceGroup; target = $_.Group }
        })
        target_before = $targetAudit
        created_at = (Get-Date).ToUniversalTime().ToString('o')
    }
    $payload = $audit | ConvertTo-Json -Depth 8 -Compress
    $encoded = ConvertTo-Base64Utf8 $payload
    $path = "/opt/new-api/migration-backups/sub2api-$modeName-$migrationStamp-source-$($Item.Source.user.id)-pre.json"
    $remote = "sudo install -d -m 700 /opt/new-api/migration-backups; umask 077; printf %s '$encoded' | base64 -d > '$path'"
    Invoke-Ssh $targetAlias $remote | Out-Null
    return $path
}

function New-ImportSql([object]$Source, [int64]$Quota, [object[]]$TokenRows, [string]$Fingerprint) {
    $email = ([string]$Source.user.email).Trim().ToLowerInvariant()
    $username = ([string]$Source.user.username).Trim()
    if (-not $username) { $username = $email.Trim().ToLowerInvariant() }
    $remark = [string]$Source.user.notes
    $createdAt = [int64]$Source.user.created_at
    $password = SqlText ([string]$Source.user.password_hash)
    $emailExpr = SqlText $email
    $usernameExpr = SqlText $username
    $remarkExpr = SqlText $remark
    $batchExpr = SqlText $migrationStamp
    $fingerprintExpr = SqlText $Fingerprint
    $values = @()
    foreach ($token in $TokenRows) {
        $values += "(" + (SqlText $token.Key) + "," + (SqlNumber $token.Status) + "," + (SqlText $token.Name) + "," + (SqlNumber $token.CreatedTime) + "," + (SqlNumber $token.ExpiredTime) + "," + (SqlText $token.Group) + ")"
    }
    # Keep the CTE valid for users who have no source keys; the WHERE clause
    # below turns the sentinel row into zero inserts.
    $tokenValues = if ($values.Count -gt 0) { $values -join "," } else { "(NULL::text,0,NULL::text,0,-1,''::text)" }
    return @"
begin;
select pg_advisory_xact_lock(hashtextextended('$($email.Replace("'", "''"))', 0));
with inserted_user as (
  insert into users (username,password,display_name,role,status,email,quota,used_quota,request_count,"group",remark,created_at,last_login_at)
  select $usernameExpr,$password,$usernameExpr,1,1,$emailExpr,$Quota,0,0,'default',$remarkExpr,$createdAt,0
  where not exists (select 1 from users where lower(trim(email))=lower(trim($emailExpr)))
  returning id
), inserted_tokens as (
  insert into tokens (user_id,key,status,name,created_time,accessed_time,expired_time,remain_quota,unlimited_quota,model_limits_enabled,model_limits,allow_ips,used_quota,"group",group_candidates,cross_group_retry)
  select inserted_user.id, token_values.key, token_values.status, token_values.name, token_values.created_time, 0, token_values.expired_time, 0, true, false, '', '', 0, token_values.group_name, '', false
  from inserted_user cross join (values $tokenValues) as token_values(key,status,name,created_time,expired_time,group_name)
  where token_values.key is not null
  returning id
), inserted_mapping as (
  insert into sub2api_user_migration_mappings (source_user_id,source_email,target_user_id,migration_batch_id,source_fingerprint,imported_quota,imported_token_count,created_at)
  select $($Source.user.id),$emailExpr,inserted_user.id,$batchExpr,$fingerprintExpr,$Quota,$($TokenRows.Count),extract(epoch from now())::bigint
  from inserted_user
  where not exists (select 1 from sub2api_user_migration_mappings where source_user_id=$($Source.user.id))
  returning source_user_id
)
select (select count(*) from inserted_user) as users_inserted, (select count(*) from inserted_tokens) as tokens_inserted, (select count(*) from inserted_mapping) as mappings_inserted;
commit;
"@
}

$targetRatios = Get-TargetGroupRatio
$targetFieldLimits = Get-TargetFieldLimits
$mappingTableExists = Test-TargetMigrationTable
$report = @()
$pending = @()
$sourceEntries = @()
$modeName = 'gray'

if ($PSCmdlet.ParameterSetName -eq 'All') {
    $modeName = 'all'
    $sourceRows = @(Get-SourceSnapshot)
    if ($SnapshotOutputPath) {
        $snapshotWritten = Write-SourceSnapshotFile $sourceRows $SnapshotOutputPath
        Write-Output "snapshot=$snapshotWritten"
    }
    foreach ($source in $sourceRows) {
        $sourceEntries += [pscustomobject]@{ Address = Normalize-Email ([string]$source.user.email); Source = $source }
    }
} elseif ($PSCmdlet.ParameterSetName -eq 'Snapshot') {
    $modeName = 'snapshot'
    foreach ($source in @(Read-SourceSnapshotFile $SnapshotPath)) {
        $sourceEntries += [pscustomobject]@{ Address = Normalize-Email ([string]$source.user.email); Source = $source }
    }
} else {
    foreach ($address in @($Email | ForEach-Object { Normalize-Email $_ } | Select-Object -Unique)) {
        if (-not $address) {
            $report += [pscustomobject]@{ Email = 'email:invalid'; Result = 'invalid_input'; SourceId = ''; TargetId = ''; Quota = ''; BalanceUsd = ''; SubscriptionUsd = ''; Tokens = ''; SpecialKeys = ''; GroupAliases = ''; Detail = 'empty email' }
            continue
        }
        $source = Get-SourceUser $address
        if ($null -eq $source) {
            $report += [pscustomobject]@{ Email = Get-ReportEmail $address; Result = 'source_not_found'; SourceId = ''; TargetId = ''; Quota = ''; BalanceUsd = ''; SubscriptionUsd = ''; Tokens = ''; SpecialKeys = ''; GroupAliases = ''; Detail = '' }
            continue
        }
        if ($source -is [array] -and $source.Count -ne 1) {
            $report += [pscustomobject]@{ Email = Get-ReportEmail $address; Result = 'source_duplicate_email'; SourceId = ''; TargetId = ''; Quota = ''; BalanceUsd = ''; SubscriptionUsd = ''; Tokens = ''; SpecialKeys = ''; GroupAliases = ''; Detail = "source returned $($source.Count) rows" }
            continue
        }
        $sourceEntries += [pscustomobject]@{ Address = $address; Source = $source }
    }
}

# A normalized-email collision is a hard stop in full mode. Never pick one row
# nondeterministically when the source database has case/whitespace duplicates.
$sourceEmailCounts = @{}
foreach ($entry in $sourceEntries) {
    if (-not $sourceEmailCounts.ContainsKey($entry.Address)) { $sourceEmailCounts[$entry.Address] = 0 }
    $sourceEmailCounts[$entry.Address]++
}
$plans = @()
$tokenRowsBySourceId = @{}
foreach ($entry in $sourceEntries) {
    $source = $entry.Source
    $address = $entry.Address
    if ($sourceEmailCounts[$address] -gt 1) {
        $report += [pscustomobject]@{ Email = Get-ReportEmail $address; Result = 'source_duplicate_email'; SourceId = $source.user.id; TargetId = ''; Quota = ''; BalanceUsd = ''; SubscriptionUsd = ''; Tokens = ''; SpecialKeys = ''; GroupAliases = ''; Detail = 'normalized source email is not unique' }
        continue
    }
    $validationIssues = @(Get-SourceValidationIssues $source $targetFieldLimits)
    if ($validationIssues.Count -gt 0) {
        $report += [pscustomobject]@{ Email = Get-ReportEmail $address; Result = 'source_data_blocked'; SourceId = $source.user.id; TargetId = ''; Quota = ''; BalanceUsd = ''; SubscriptionUsd = ''; Tokens = ''; SpecialKeys = ''; GroupAliases = ''; Detail = ($validationIssues -join ';') }
        continue
    }
    try { $quotaInfo = Get-QuotaBreakdown $source } catch {
        $report += [pscustomobject]@{ Email = Get-ReportEmail $address; Result = 'source_quota_blocked'; SourceId = $source.user.id; TargetId = ''; Quota = ''; BalanceUsd = ''; SubscriptionUsd = ''; Tokens = ''; SpecialKeys = ''; GroupAliases = ''; Detail = $_.Exception.Message }
        continue
    }
    $tokenPlan = Get-TokenPlan $source $targetRatios
    $sourceBlocked = @()
    if ([string]$source.user.role -ne 'user') { $sourceBlocked += 'not_user_role' }
    if ([string]$source.user.status -ne 'active') { $sourceBlocked += 'not_active' }
    if ((To-Decimal $source.user.frozen_balance) -ne 0) { $sourceBlocked += 'frozen_balance_nonzero' }
    if ($quotaInfo.Issues.Count -gt 0) { $sourceBlocked += @($quotaInfo.Issues) }
    if ([int64]$quotaInfo.Quota -gt 2147483647) { $sourceBlocked += 'quota_exceeds_target_int32' }
    if ($tokenPlan.Issues.Count -gt 0) { $sourceBlocked += @($tokenPlan.Issues) }
    $aliasText = @($tokenPlan.Aliases.GetEnumerator() | ForEach-Object { "$($_.Key) x$($_.Value)" }) -join ';'
    if ($sourceBlocked.Count -gt 0) {
        $report += [pscustomobject]@{ Email = Get-ReportEmail $address; Result = if ($tokenPlan.Issues.Count -gt 0) { 'source_key_policy_blocked' } else { 'source_blocked' }; SourceId = $source.user.id; TargetId = ''; Quota = $quotaInfo.Quota; BalanceUsd = $quotaInfo.BalanceUsd; SubscriptionUsd = $quotaInfo.SubscriptionUsd; Tokens = $tokenPlan.Rows.Count; SpecialKeys = $tokenPlan.SpecialCount; GroupAliases = $aliasText; Detail = ($sourceBlocked -join ';') }
        continue
    }
    $plan = [pscustomobject]@{ Address = $address; Source = $source; Quota = $quotaInfo.Quota; BalanceUsd = $quotaInfo.BalanceUsd; SubscriptionUsd = $quotaInfo.SubscriptionUsd; Tokens = @($tokenPlan.Rows); SpecialKeys = $tokenPlan.SpecialCount; GroupAliases = $aliasText }
    $plans += $plan
    $tokenRowsBySourceId[[string]$source.user.id] = @($tokenPlan.Rows)
}

$preflightSources = @($plans | ForEach-Object { $_.Source })
$targetPreflight = Get-TargetPreflight $preflightSources $tokenRowsBySourceId $mappingTableExists
$batchKeyOwners = @{}
foreach ($plan in $plans) {
    foreach ($token in @($plan.Tokens)) {
        if (-not $batchKeyOwners.ContainsKey($token.Key)) { $batchKeyOwners[$token.Key] = @() }
        $batchKeyOwners[$token.Key] += [string]$plan.Source.user.id
    }
}
foreach ($plan in $plans) {
    $sourceId = [string]$plan.Source.user.id
    $mapping = if ($targetPreflight.Mappings.ContainsKey($sourceId)) { $targetPreflight.Mappings[$sourceId] } else { $null }
    if ($null -ne $mapping) {
        $report += [pscustomobject]@{ Email = Get-ReportEmail $plan.Address; Result = 'already_migrated'; SourceId = $sourceId; TargetId = $mapping.target_id; Quota = $mapping.quota; BalanceUsd = $plan.BalanceUsd; SubscriptionUsd = $plan.SubscriptionUsd; Tokens = $mapping.tokens; SpecialKeys = $plan.SpecialKeys; GroupAliases = $plan.GroupAliases; Detail = '' }
        continue
    }
    $targetRows = if ($targetPreflight.Users.ContainsKey($plan.Address)) { @($targetPreflight.Users[$plan.Address]) } else { @() }
    if ($targetRows.Count -gt 0) {
        $report += [pscustomobject]@{ Email = Get-ReportEmail $plan.Address; Result = 'target_conflict_skipped'; SourceId = $sourceId; TargetId = $targetRows[0].id; Quota = $plan.Quota; BalanceUsd = $plan.BalanceUsd; SubscriptionUsd = $plan.SubscriptionUsd; Tokens = $plan.Tokens.Count; SpecialKeys = $plan.SpecialKeys; GroupAliases = $plan.GroupAliases; Detail = "target rows=$($targetRows.Count);target_quota=$($targetRows[0].quota);target_used_quota=$($targetRows[0].used_quota);target_tokens=$($targetRows[0].token_count)" }
        continue
    }
    $targetKeyConflict = @($plan.Tokens | Where-Object { $targetPreflight.Keys.ContainsKey($_.Key) })
    if ($targetKeyConflict.Count -gt 0) {
        $report += [pscustomobject]@{ Email = Get-ReportEmail $plan.Address; Result = 'target_key_conflict_skipped'; SourceId = $sourceId; TargetId = ''; Quota = $plan.Quota; BalanceUsd = $plan.BalanceUsd; SubscriptionUsd = $plan.SubscriptionUsd; Tokens = $plan.Tokens.Count; SpecialKeys = $plan.SpecialKeys; GroupAliases = $plan.GroupAliases; Detail = "conflicting keys=$($targetKeyConflict.Count)" }
        continue
    }
    $batchConflict = @($plan.Tokens | Where-Object { $batchKeyOwners[$_.Key].Count -gt 1 })
    if ($batchConflict.Count -gt 0) {
        $report += [pscustomobject]@{ Email = Get-ReportEmail $plan.Address; Result = 'batch_key_conflict_blocked'; SourceId = $sourceId; TargetId = ''; Quota = $plan.Quota; BalanceUsd = $plan.BalanceUsd; SubscriptionUsd = $plan.SubscriptionUsd; Tokens = $plan.Tokens.Count; SpecialKeys = $plan.SpecialKeys; GroupAliases = $plan.GroupAliases; Detail = "duplicate keys in batch=$($batchConflict.Count)" }
        continue
    }
    $pending += $plan
    $report += [pscustomobject]@{ Email = Get-ReportEmail $plan.Address; Result = 'ready'; SourceId = $sourceId; TargetId = ''; Quota = $plan.Quota; BalanceUsd = $plan.BalanceUsd; SubscriptionUsd = $plan.SubscriptionUsd; Tokens = $plan.Tokens.Count; SpecialKeys = $plan.SpecialKeys; GroupAliases = $plan.GroupAliases; Detail = '' }
}

Write-Output ('mode=' + ($(if ($Apply) { 'apply' } else { 'dry-run' })) + ";scope=$modeName;source_count=$($sourceEntries.Count)")
$summary = @($report | Group-Object Result | ForEach-Object { "$($_.Name)=$($_.Count)" }) -join ';'
Write-Output "summary=$summary"
if ($DetailedReport -or $report.Count -le 25) {
    $report | Format-Table -AutoSize | Out-String | Write-Output
} else {
    Write-Output 'detail=omitted; use -DetailedReport or the JSON report for masked row diagnostics'
    $report | Select-Object -First 25 | Format-Table -AutoSize | Out-String | Write-Output
}

$reportPath = if ($ReportPath) { $ReportPath } else { Join-Path $PSScriptRoot "migration-reports\$modeName-$migrationStamp.json" }
$reportParent = Split-Path -Parent $reportPath
if ($reportParent) { New-Item -ItemType Directory -Force $reportParent | Out-Null }
([pscustomobject]@{ created_at = (Get-Date).ToUniversalTime().ToString('o'); scope = $modeName; applied = @(); skipped = $report }) |
    ConvertTo-Json -Depth 8 | Set-Content -Path $reportPath -Encoding utf8NoBOM
Write-Output "report=$reportPath"

if (-not $Apply) { return }

# Applying a full snapshot with hidden blockers would create a partial migration
# that looks successful. Fail closed; `already_migrated` is the only non-ready
# result allowed in an idempotent rerun.
$blocking = @($report | Where-Object { $_.Result -notin @('ready', 'already_migrated') })
if ($blocking.Count -gt 0) {
    throw "apply blocked by $($blocking.Count) non-ready source rows; resolve dry-run blockers first"
}

$applied = @()
if ($pending.Count -gt 0) { Ensure-MigrationTable }
foreach ($item in $pending) {
    $fingerprint = Get-SourceFingerprint $item.Source
    $targetBefore = @(Get-TargetState $item.Address)
    if ($targetBefore.Count -gt 0) { throw "target conflict appeared before import for $(Get-ReportEmail $item.Address)" }
    if ($null -ne (Get-TargetMapping ([int64]$item.Source.user.id))) { throw "migration mapping appeared before import for $(Get-ReportEmail $item.Address)" }
    if ((Get-TargetTokenConflictCount $item.Tokens) -gt 0) { throw "target key conflict appeared before import for $(Get-ReportEmail $item.Address)" }
    $auditPath = Write-TargetAudit $item $fingerprint $targetBefore
    $sql = New-ImportSql $item.Source $item.Quota $item.Tokens $fingerprint
    $result = Invoke-TargetSql $sql
    $parts = $result -split '\|'
    if ($parts.Count -lt 3 -or $parts[0] -ne '1' -or [int]$parts[1] -ne $item.Tokens.Count -or $parts[2] -ne '1') {
        throw "import did not insert exactly one user and $($item.Tokens.Count) tokens for $(Get-ReportEmail $item.Address): $result"
    }
    $target = @(Get-TargetState $item.Address)
    if ($target.Count -ne 1) { throw "post-import target lookup failed for $(Get-ReportEmail $item.Address)" }
    $applied += [pscustomobject]@{ Email = Get-ReportEmail $item.Address; SourceId = $item.Source.user.id; TargetId = $target[0].id; Quota = $item.Quota; Tokens = $item.Tokens.Count; Audit = $auditPath }
}

([pscustomobject]@{ created_at = (Get-Date).ToUniversalTime().ToString('o'); scope = $modeName; applied = $applied; skipped = $report | Where-Object Result -ne 'ready' }) |
    ConvertTo-Json -Depth 8 | Set-Content -Path $reportPath -Encoding utf8NoBOM
Write-Output "report=$reportPath"
$applied | Format-Table -AutoSize | Out-String | Write-Output
