param(
    [Parameter(Mandatory)]
    [ValidateNotNullOrEmpty()]
    [string[]]$Email,
    [switch]$Apply
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
    'group', g.name
  ) order by s.id) from user_subscriptions s join groups g on g.id = s.group_id
    where s.user_id = u.id and s.deleted_at is null), '[]'::json))::text,'UTF8'),'base64')
from users u
where lower(trim(u.email)) = lower('$emailSql') and u.deleted_at is null;
"@
    $encoded = Invoke-SourceSql $sql
    if (-not $encoded) { return $null }
    return (ConvertFrom-Base64Utf8 $encoded | ConvertFrom-Json)
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

function Get-TargetTokenConflictCount([object[]]$TokenRows) {
    if ($TokenRows.Count -eq 0) { return 0 }
    $keyExpressions = @($TokenRows | ForEach-Object { SqlText $_.Key }) -join ','
    $raw = Invoke-TargetSql @"
select count(*)
from tokens
where deleted_at is null and key in ($keyExpressions);
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
    $available = @($TargetRatios.PSObject.Properties.Name)
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

function Write-TargetAudit([object]$Item, [string]$Fingerprint, [object]$TargetBefore) {
    $audit = [pscustomobject]@{
        batch_id = $migrationStamp
        source_user_id = [int64]$Item.Source.user.id
        source_email = [string]$Item.Source.user.email
        source_fingerprint = $Fingerprint
        calculated_quota = [int64]$Item.Quota
        token_count = $Item.Tokens.Count
        token_group_mappings = @($Item.Tokens | ForEach-Object {
            [pscustomobject]@{ source = $_.SourceGroup; target = $_.Group }
        })
        target_before = $TargetBefore
        created_at = (Get-Date).ToUniversalTime().ToString('o')
    }
    $payload = $audit | ConvertTo-Json -Depth 8 -Compress
    $encoded = ConvertTo-Base64Utf8 $payload
    $path = "/opt/new-api/migration-backups/sub2api-gray-$migrationStamp-source-$($Item.Source.user.id)-pre.json"
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
$report = @()
$pending = @()
$batchTokenKeys = [Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
foreach ($address in @($Email | ForEach-Object { $_.Trim().ToLowerInvariant() } | Select-Object -Unique)) {
    $source = Get-SourceUser $address
    if ($null -eq $source) {
        $report += [pscustomobject]@{ Email = $address; Result = 'source_not_found'; SourceId = ''; TargetId = ''; Quota = ''; Tokens = '' }
        continue
    }
    $quota = Get-Quota $source
    try {
        $tokenRows = @(Get-TokenRows $source $targetRatios)
    } catch {
        $report += [pscustomobject]@{ Email = $address; Result = 'source_key_policy_blocked'; SourceId = $source.user.id; TargetId = ''; Quota = $quota; Tokens = ''; Detail = $_.Exception.Message }
        continue
    }
    if ([string]$source.user.role -ne 'user' -or [string]$source.user.status -ne 'active' -or (To-Decimal $source.user.frozen_balance) -ne 0) {
        $report += [pscustomobject]@{ Email = $address; Result = 'source_blocked'; SourceId = $source.user.id; TargetId = ''; Quota = $quota; Tokens = $tokenRows.Count }
        continue
    }
    $existingMapping = Get-TargetMapping ([int64]$source.user.id)
    if ($null -ne $existingMapping) {
        $report += [pscustomobject]@{ Email = $address; Result = 'already_migrated'; SourceId = $source.user.id; TargetId = $existingMapping.target_user_id; Quota = $existingMapping.imported_quota; Tokens = $existingMapping.imported_token_count }
        continue
    }
    $targetRows = @(Get-TargetState $address)
    if ($targetRows.Count -gt 0) {
        $report += [pscustomobject]@{ Email = $address; Result = 'target_conflict_skipped'; SourceId = $source.user.id; TargetId = $targetRows[0].id; Quota = $quota; Tokens = $tokenRows.Count }
        continue
    }
    if ((Get-TargetTokenConflictCount $tokenRows) -gt 0) {
        $report += [pscustomobject]@{ Email = $address; Result = 'target_key_conflict_skipped'; SourceId = $source.user.id; TargetId = ''; Quota = $quota; Tokens = $tokenRows.Count }
        continue
    }
    $duplicateInBatch = $false
    foreach ($token in $tokenRows) {
        if (-not $batchTokenKeys.Add($token.Key)) {
            $duplicateInBatch = $true
        }
    }
    if ($duplicateInBatch) {
        $report += [pscustomobject]@{ Email = $address; Result = 'batch_key_conflict_blocked'; SourceId = $source.user.id; TargetId = ''; Quota = $quota; Tokens = $tokenRows.Count }
        continue
    }
    $pending += [pscustomobject]@{ Address = $address; Source = $source; Quota = $quota; Tokens = $tokenRows }
    $report += [pscustomobject]@{ Email = $address; Result = 'ready'; SourceId = $source.user.id; TargetId = ''; Quota = $quota; Tokens = $tokenRows.Count }
}

Write-Output ('mode=' + ($(if ($Apply) { 'apply' } else { 'dry-run' })))
$report | Format-Table -AutoSize | Out-String | Write-Output

if (-not $Apply) { exit 0 }

$applied = @()
foreach ($item in $pending) {
    $fingerprint = Get-SourceFingerprint $item.Source
    $targetBefore = @(Get-TargetState $item.Address)
    if ($targetBefore.Count -gt 0) { throw "target conflict appeared before import for $($item.Address)" }
    if ($null -ne (Get-TargetMapping ([int64]$item.Source.user.id))) { throw "migration mapping appeared before import for $($item.Address)" }
    if ((Get-TargetTokenConflictCount $item.Tokens) -gt 0) { throw "target key conflict appeared before import for $($item.Address)" }
    $auditPath = Write-TargetAudit $item $fingerprint $targetBefore
    Ensure-MigrationTable
    $sql = New-ImportSql $item.Source $item.Quota $item.Tokens $fingerprint
    $result = Invoke-TargetSql $sql
    $parts = $result -split '\|'
    if ($parts.Count -lt 3 -or $parts[0] -ne '1' -or [int]$parts[1] -ne $item.Tokens.Count -or $parts[2] -ne '1') {
        throw "import did not insert exactly one user and $($item.Tokens.Count) tokens for $($item.Address): $result"
    }
    $target = @(Get-TargetState $item.Address)
    if ($target.Count -ne 1) { throw "post-import target lookup failed for $($item.Address)" }
    $applied += [pscustomobject]@{ Email = $item.Address; SourceId = $item.Source.user.id; TargetId = $target[0].id; Quota = $item.Quota; Tokens = $item.Tokens.Count; Audit = $auditPath }
}

$reportPath = Join-Path $PSScriptRoot "migration-reports\gray-$migrationStamp.json"
New-Item -ItemType Directory -Force (Split-Path $reportPath) | Out-Null
([pscustomobject]@{ created_at = (Get-Date).ToUniversalTime().ToString('o'); applied = $applied; skipped = $report | Where-Object Result -ne 'ready' }) |
    ConvertTo-Json -Depth 8 | Set-Content -Path $reportPath -Encoding utf8NoBOM
Write-Output "report=$reportPath"
$applied | Format-Table -AutoSize | Out-String | Write-Output
