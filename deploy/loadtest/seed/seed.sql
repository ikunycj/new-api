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

INSERT INTO channels (
  type, "key", status, name, weight, created_time, response_time,
  base_url, balance, balance_updated_time, models, "group", used_quota,
  other, other_info, channel_info, settings
)
SELECT
  1,
  'sk-local-mock-upstream',
  1,
  'Load Test Mock OpenAI',
  100,
  extract(epoch FROM now())::bigint,
  0,
  'http://mock-upstream:8080',
  0,
  0,
  'gpt-3.5-turbo',
  'default',
  0,
  '',
  '',
  '{}',
  ''
WHERE NOT EXISTS (
  SELECT 1 FROM channels WHERE name = 'Load Test Mock OpenAI'
);

INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT 'default', 'gpt-3.5-turbo', id, true, 0, 100
FROM channels
WHERE name = 'Load Test Mock OpenAI'
ON CONFLICT ("group", model, channel_id) DO UPDATE SET
  enabled = EXCLUDED.enabled,
  priority = EXCLUDED.priority,
  weight = EXCLUDED.weight;

COMMIT;

SELECT
  (SELECT count(*) FROM users WHERE username LIKE 'loadtest_user_%') AS users,
  (SELECT count(*) FROM tokens WHERE "key" LIKE 'loadtest%') AS tokens,
  (SELECT count(*) FROM channels WHERE name = 'Load Test Mock OpenAI') AS channels;
