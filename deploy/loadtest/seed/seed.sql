BEGIN;

INSERT INTO users (
  username, password, display_name, role, status, quota, used_quota,
  request_count, "group", aff_code, setting, created_at
)
SELECT
  'loadtest_user_' || lpad(number::text, 5, '0'),
  'loadtest-login-disabled',
  'Load Test ' || number,
  1,
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

INSERT INTO clusters (id, name, type, status, remark, archived, created_time, updated_time)
VALUES
  (1, 'Mock Cluster A', 'custom', 1, 'Deterministic load-test cluster', false, extract(epoch FROM now())::bigint, extract(epoch FROM now())::bigint),
  (2, 'Mock Cluster B', 'custom', 1, 'Deterministic failover target', false, extract(epoch FROM now())::bigint, extract(epoch FROM now())::bigint)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, type = EXCLUDED.type, status = EXCLUDED.status, archived = false, updated_time = EXCLUDED.updated_time;

INSERT INTO cluster_pools (id, cluster_id, tier, name, status, cost_factor, remark, created_time, updated_time)
VALUES
  (1, 1, 1, 'Free', 1, 0.0, '', extract(epoch FROM now())::bigint, extract(epoch FROM now())::bigint),
  (2, 1, 2, 'Pro/Plus', 1, 1.0, '', extract(epoch FROM now())::bigint, extract(epoch FROM now())::bigint),
  (3, 1, 3, 'Fallback', 1, 1.5, '', extract(epoch FROM now())::bigint, extract(epoch FROM now())::bigint),
  (4, 2, 1, 'Free', 1, 0.0, '', extract(epoch FROM now())::bigint, extract(epoch FROM now())::bigint),
  (5, 2, 2, 'Pro/Plus', 1, 1.0, '', extract(epoch FROM now())::bigint, extract(epoch FROM now())::bigint),
  (6, 2, 3, 'Fallback', 1, 1.5, '', extract(epoch FROM now())::bigint, extract(epoch FROM now())::bigint)
ON CONFLICT (id) DO UPDATE SET cluster_id = EXCLUDED.cluster_id, tier = EXCLUDED.tier, name = EXCLUDED.name, status = EXCLUDED.status, cost_factor = EXCLUDED.cost_factor, updated_time = EXCLUDED.updated_time;

INSERT INTO failover_policies (
  id, name, mode, enabled, same_pool_retries, connect_timeout_ms, first_byte_timeout_ms,
  max_pool_attempts, max_cluster_attempts, max_total_attempts, total_failover_budget_ms,
  switch_status_codes, switch_error_codes, circuit_failure_threshold, circuit_window_seconds,
  circuit_cooldown_seconds, circuit_half_open_requests, allow_paid_escalation, allow_fallback,
  max_cost_multiplier, created_time, updated_time
)
VALUES
  (1, 'Balanced load test', 'balanced', true, 0, 1500, 3000, 3, 2, 6, 10000, '[429,500,502,503,504]', '["pool_exhausted","all_pools_exhausted"]', 3, 30, 30, 1, true, true, 2.0, extract(epoch FROM now())::bigint, extract(epoch FROM now())::bigint)
ON CONFLICT (id) DO UPDATE SET enabled = true, max_cluster_attempts = 2, max_total_attempts = 6, total_failover_budget_ms = 10000, updated_time = EXCLUDED.updated_time;

DELETE FROM abilities
WHERE channel_id IN (SELECT id FROM channels WHERE name LIKE 'Load Test Cluster %');

DELETE FROM channels WHERE name LIKE 'Load Test Cluster %';

INSERT INTO channels (
  type, "key", status, name, weight, created_time, response_time,
  base_url, balance, balance_updated_time, models, "group", used_quota,
  other, other_info, channel_info, settings, setting, header_override,
  priority, cluster_id, cluster_pool_id
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
  '{"error_source":"cluster"}',
  seed.header_override,
  seed.priority,
  seed.cluster_id,
  seed.cluster_pool_id
FROM (VALUES
  ('sk-local-mock-a-p1', 'Load Test Cluster A P1', 'http://mock-upstream:8080', '{"X-Mock-Pool-Tier":"1"}', 600, 1, 1),
  ('sk-local-mock-a-p2', 'Load Test Cluster A P2', 'http://mock-upstream:8080', '{"X-Mock-Pool-Tier":"2"}', 500, 1, 2),
  ('sk-local-mock-a-p3', 'Load Test Cluster A P3', 'http://mock-upstream:8080', '{"X-Mock-Pool-Tier":"3"}', 400, 1, 3),
  ('sk-local-mock-b-p1', 'Load Test Cluster B P1', 'http://mock-upstream-b:8080', '{"X-Mock-Pool-Tier":"1"}', 300, 2, 4),
  ('sk-local-mock-b-p2', 'Load Test Cluster B P2', 'http://mock-upstream-b:8080', '{"X-Mock-Pool-Tier":"2"}', 200, 2, 5),
  ('sk-local-mock-b-p3', 'Load Test Cluster B P3', 'http://mock-upstream-b:8080', '{"X-Mock-Pool-Tier":"3"}', 100, 2, 6)
) AS seed(api_key, name, base_url, header_override, priority, cluster_id, cluster_pool_id);

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
WHERE name LIKE 'Load Test Cluster %'
ON CONFLICT ("group", model, channel_id) DO UPDATE SET
  enabled = EXCLUDED.enabled,
  priority = EXCLUDED.priority,
  weight = EXCLUDED.weight;

COMMIT;

SELECT
  (SELECT count(*) FROM users WHERE username LIKE 'loadtest_user_%') AS users,
  (SELECT count(*) FROM tokens WHERE "key" LIKE 'loadtest%') AS tokens,
  (SELECT count(*) FROM channels WHERE name LIKE 'Load Test Cluster %') AS channels;
