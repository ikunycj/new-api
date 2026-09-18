# APISIX HTTP audit gateway deployment

Use this playbook when the requested change includes replacing OpenResty with APISIX, configuring `http-logger`, deploying the simple audit collector, switching public gateway traffic, or rolling back those components. Application deployment remains governed by `deployment-playbook.md` and must stay independently reversible.

## 1. Confirm scope and authorization

Before remote work, confirm:

- application environment and `$deployHost`;
- `$gatewayHost`;
- `$collectorHost`;
- audit PostgreSQL host;
- public domain;
- requested operation: discovery, database setup, collector deployment, APISIX candidate, plugin update, 80/443 cutover, or rollback;
- whether DNS changes are authorized.

For production, require explicit authorization before every gateway, TLS, 80/443, collector, audit-database, body-capture, or DNS write. Permission to deploy new-api does not authorize these operations.

## 2. Architecture contract

```text
client -> APISIX -> new-api
             |
             +-> http-logger -> audit-collector -> independent PostgreSQL
```

- APISIX, collector, audit PostgreSQL, and new-api are separate deployment and rollback units.
- APISIX and collector changes must not modify `/opt/new-api/.env` or recreate the application's PostgreSQL and Redis.
- Audit PostgreSQL uses separate credentials and requires separate authorization.
- The audit path is fail-open: collector or audit-database failure must not fail an otherwise successful AI request.

## 3. First-stage scope

The collector receives an APISIX JSON log object, places it into a bounded in-memory queue, and asynchronously inserts the object into PostgreSQL as JSONB. JSONB preserves fields and values but not the original byte representation, whitespace, key order, or duplicate keys.

Do not add:

- SSE or AI protocol parsing;
- question/answer derivation;
- body/header masking;
- idempotency;
- local WAL;
- metrics or alerts;
- object storage or search indexing.

Document that this simplified scope can lose acknowledged in-memory entries after collector failure and can insert duplicates after APISIX retries.

## 4. APISIX mode

Use Standalone file-driven mode for the initial single-node gateway:

```yaml
deployment:
  role: data_plane
  role_data_plane:
    config_provider: yaml
```

Do not add etcd solely for this migration. Keep the Admin API disabled. Ensure `conf/apisix.yaml` ends with `#END`.

Pin the APISIX image version and record its digest. Never deploy `latest`. Preserve the previous APISIX image and configuration snapshot.

## 5. Read-only discovery

Before changes:

1. Verify app, gateway, collector, and audit-database hosts.
2. Verify live DNS records and TTL.
3. Identify current 80/443 listeners and gateway service manager.
4. Record current OpenResty/Nginx configuration metadata and SHA-256.
5. Inspect TLS certificate subject, SANs, expiry, fingerprint, and renewal mechanism without displaying the private key.
6. Verify direct new-api health and Docker network/upstream address.
7. Verify current public `/api/status`.
8. Capture one ordinary AI request and one SSE timing baseline.
9. Check whether APISIX or collector already exists; do not create duplicates.

## 6. Runtime layout

Recommended paths:

```text
/opt/new-api/           application
/opt/new-api-gateway/   APISIX
/opt/new-api-audit/     collector
```

Keep TLS private keys, collector authentication, audit PostgreSQL credentials, and runtime-rendered configuration outside Git and ordinary release archives. Never print or diff secret values.

## 7. Route and logger policy

Enable `http-logger` only on approved text-generation POST routes:

```text
/v1/completions
/v1/chat/completions
/v1/responses
/v1/responses/compact
/v1/messages
```

Exclude WebSocket, image, audio, video, embedding, rerank, file, login, registration, payment, and administrative routes.

Required logger behavior:

- `include_req_body: true`;
- `include_resp_body: true`;
- explicit `max_req_body_bytes` and `max_resp_body_bytes`;
- `batch_max_size: 1`; verify the target version sends one JSON object during candidate testing;
- `concat_method: json`;
- short timeout;
- nonzero retry count;
- private collector URI;
- collector authorization supplied from a protected environment value;
- `log_format_extra` for gateway request ID, environment, and gateway instance.

Do not set unlimited body sizes. Start with a bounded candidate configuration and test real request/response sizes and APISIX worker memory.

The collector stores the APISIX event as received. No masking is performed in the first stage, so the audit database may contain request headers, response headers, Authorization, Cookie, API keys, and sensitive request/response content. This is an explicitly accepted first-stage limitation.

## 8. Collector contract

Endpoint:

```text
POST /v1/audit/events
Content-Type: application/json
Authorization: <collector secret>
```

The collector accepts one JSON object or a single-element JSON array. It does not support multi-event batches or NDJSON. Candidate testing must capture the actual `http-logger` payload shape for the pinned APISIX version.

Response semantics:

- `202`: JSON entered the bounded in-memory queue;
- `400`: invalid JSON;
- `401/403`: authentication failed;
- `413`: request too large;
- `503`: queue full or collector shutting down.

The background worker inserts the validated log object into PostgreSQL JSONB without business parsing or field derivation:

```sql
CREATE TABLE gateway_http_logs (
    id BIGSERIAL PRIMARY KEY,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    payload JSONB NOT NULL
);
```

The collector does not parse SSE or derive business fields. It may inspect only what is necessary to validate an object or single-element array and insert the contained object as JSONB. This is semantic JSON storage, not byte-for-byte preservation.

## 9. Candidate deployment

1. Provision the independent PostgreSQL database and table under separate authorization.
2. Deploy the collector and verify `/healthz`.
3. POST one test JSON and verify one database row.
4. Start APISIX on alternate local ports.
5. Attach it only to the verified app and audit networks.
6. Configure new-api as upstream and enable `http-logger` on approved routes. Do not use a broad `/v1beta/models/*` audit route until a target-version matcher has been verified to include only the intended Gemini generation actions.
7. Mount gateway configuration and TLS material read-only.
8. Validate configuration and record its hash.
9. Test with `curl --resolve`, explicit Host/SNI, or a canary hostname without changing production DNS.
10. Do not stop or remove OpenResty during candidate validation.

## 10. Acceptance tests

Verify:

- direct new-api `/api/status`;
- APISIX-path `/api/status`;
- public pages and authentication;
- one non-streaming AI request;
- one streaming AI request;
- request body appears in `payload.request.body`;
- response body appears in `payload.response.body`;
- SSE still arrives progressively at the client;
- collector stores the raw SSE text without parsing;
- collector unavailable does not fail the AI request;
- PostgreSQL unavailable does not fail the AI request;
- queue full returns `503` to APISIX;
- body-limit truncation behavior is understood;
- APISIX config reload works;
- old OpenResty can be restored.

## 11. Cutover

Prefer a load balancer, port forward, or another immediately reversible entry switch. If APISIX must take 80/443 on the same host, use a bounded script:

```text
stop old gateway
  -> bind APISIX to 80/443
  -> run local health check
  -> on failure stop APISIX public listener and restore old gateway
```

Complete APISIX TLS, upstream, and candidate listener checks before releasing old ports. After cutover, verify public status, one ordinary AI request, one SSE request, and database insertion. Keep OpenResty configuration and restart instructions throughout the rollback window.

DNS changes require separate authorization and are not the default cutover mechanism.

## 12. Rollback

Use the smallest rollback:

1. Audit-only failure: remove `http-logger` from the affected routes.
2. APISIX configuration failure: restore the previous `apisix.yaml` snapshot.
3. APISIX/TLS/listener failure: release 80/443 and restore OpenResty.
4. Collector failure: restore the previous collector image or disable `http-logger`.
5. Audit PostgreSQL failure: restore that independent database or disable `http-logger`.
6. Application failure: use the existing new-api image rollback only if direct application checks fail.
7. DNS rollback: only if DNS was separately changed and listener rollback is insufficient.

Do not recreate or modify the new-api application PostgreSQL or Redis during gateway or audit rollback.

## 13. Cleanup gate

Do not remove the old gateway, its configuration, TLS integration, or rollback image until the observation period is complete and the user explicitly approves cleanup. Preserve sanitized deployment records containing image digest, configuration hash, cutover time, and validation request IDs; do not include captured request or response bodies in deployment records.
