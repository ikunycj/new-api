BEGIN;

INSERT INTO users (
  username, password, display_name, role, user_type, status, quota, used_quota,
  request_count, "group", aff_code, setting, created_at
)
SELECT
  'loadtest_user_' || lpad(number::text, 5, '0'),
  'loadtest-login-disabled',
  'Load Test ' || number,
  1,
  'toB',
  1,
  2000000000,
  0,
  0,
  'default',
  'LT' || lpad(number::text, 14, '0'),
  '{}',
  extract(epoch FROM now())::bigint
FROM generate_series(1, :loadtest_users) AS number
ON CONFLICT (username) DO UPDATE SET
  status = EXCLUDED.status,
  user_type = EXCLUDED.user_type,
  quota = EXCLUDED.quota,
  "group" = EXCLUDED."group";

INSERT INTO tokens (
  user_id, "key", status, name, created_time, accessed_time, expired_time,
  remain_quota, unlimited_quota, model_limits_enabled, model_limits,
  "group", group_candidates, cross_group_retry
)
SELECT
  id,
  'loadtest' || lpad(substring(username FROM '[0-9]+$'), 5, '0'),
  1,
  'Load Test Token',
  extract(epoch FROM now())::bigint,
  0,
  -1,
  0,
  true,
  false,
  '',
  'default',
  '',
  false
FROM users
WHERE username LIKE 'loadtest_user_%'
  AND substring(username FROM '[0-9]+$')::int <= :loadtest_users
ON CONFLICT ("key") DO UPDATE SET
  user_id = EXCLUDED.user_id,
  status = EXCLUDED.status,
  unlimited_quota = EXCLUDED.unlimited_quota,
  "group" = EXCLUDED."group";

DELETE FROM abilities
WHERE channel_id IN (SELECT id FROM channels WHERE name LIKE 'Load Test Channel %');

DELETE FROM billing_group_channels
WHERE channel_id IN (SELECT id FROM channels WHERE name LIKE 'Load Test Channel %');

DELETE FROM channels WHERE name LIKE 'Load Test Channel %';

INSERT INTO channels (
  type, "key", status, name, weight, created_time, response_time,
  base_url, balance, balance_updated_time, models, "group", used_quota,
  other, other_info, channel_info, settings, setting, header_override, priority
)
SELECT
  1,
  seed.api_key,
  1,
  seed.name,
  100,
  extract(epoch FROM now())::bigint,
  0,
  seed.base_url,
  0,
  0,
  'gpt-3.5-turbo,gpt-5.6-sol,gpt-5.6-terra,gpt-5.6-luna,gpt-5.4-mini,gpt-4o,gpt-4.1',
  'default',
  0,
  '',
  '',
  '{}',
  '',
  '{"error_source":"channel"}',
  '',
  seed.priority
FROM (VALUES
  ('sk-local-mock-a', 'Load Test Channel A', 'http://mock-upstream:8080', 600),
  ('sk-local-mock-b', 'Load Test Channel B', 'http://mock-upstream-b:8080', 500)
) AS seed(api_key, name, base_url, priority);

INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT 'default', models.model, channels.id, true, channels.priority, 100
FROM channels
JOIN (VALUES
  ('gpt-3.5-turbo'),
  ('gpt-5.6-sol'),
  ('gpt-5.6-terra'),
  ('gpt-5.6-luna'),
  ('gpt-5.4-mini'),
  ('gpt-4o'),
  ('gpt-4.1')
) AS models(model) ON true
WHERE name LIKE 'Load Test Channel %'
ON CONFLICT ("group", model, channel_id) DO UPDATE SET
  enabled = EXCLUDED.enabled,
  priority = EXCLUDED.priority,
  weight = EXCLUDED.weight;

INSERT INTO billing_group_routes (
  billing_group, name, mode, enabled, max_total_attempts, total_timeout_ms,
  circuit_failure_threshold, circuit_window_seconds, circuit_cooldown_seconds,
  circuit_half_open_requests, created_time, updated_time
)
VALUES (
  'default', 'Default load-test route', 'balanced', true, 4, 10000,
  3, 30, 30, 1, extract(epoch FROM now())::bigint, extract(epoch FROM now())::bigint
)
ON CONFLICT (billing_group) DO UPDATE SET
  name = EXCLUDED.name,
  mode = EXCLUDED.mode,
  enabled = EXCLUDED.enabled,
  max_total_attempts = EXCLUDED.max_total_attempts,
  total_timeout_ms = EXCLUDED.total_timeout_ms,
  updated_time = EXCLUDED.updated_time;

DELETE FROM billing_group_channels
WHERE billing_group_route_id = (SELECT id FROM billing_group_routes WHERE billing_group = 'default');

INSERT INTO billing_group_channels (
  billing_group_route_id, channel_id, priority, weight, max_attempts, enabled, cost_factor
)
SELECT
  (SELECT id FROM billing_group_routes WHERE billing_group = 'default'),
  id,
  CASE name WHEN 'Load Test Channel A' THEN 100 ELSE 90 END,
  100,
  1,
  true,
  CASE name WHEN 'Load Test Channel A' THEN 0.2 ELSE 1.0 END
FROM channels
WHERE name IN ('Load Test Channel A', 'Load Test Channel B');

INSERT INTO channel_error_mappings (
  channel_id, channel_type, raw_code, status_code, alltoken_code,
  category, failure_scope, action, retryable, enabled
)
VALUES
  (0, 0, 'channel_exhausted', 503, 205001, 'upstream', 'channel', 'switch_channel', true, true),
  (0, 0, 'mock_error', 503, 205002, 'upstream', 'channel', 'switch_channel', true, true)
ON CONFLICT (channel_id, channel_type, raw_code, status_code) DO UPDATE SET
  alltoken_code = EXCLUDED.alltoken_code,
  category = EXCLUDED.category,
  failure_scope = EXCLUDED.failure_scope,
  action = EXCLUDED.action,
  retryable = EXCLUDED.retryable,
  enabled = EXCLUDED.enabled;

COMMIT;

SELECT
  (SELECT count(*) FROM users WHERE username LIKE 'loadtest_user_%') AS users,
  (SELECT count(*) FROM tokens WHERE "key" LIKE 'loadtest%') AS tokens,
  (SELECT count(*) FROM channels WHERE name LIKE 'Load Test Channel %') AS channels;
