# Observability shutdown plan

## Scope

This change disables the Grafana monitoring chain without disabling gateway
traffic logs or channel health probes.

Disabled components:

- Prometheus scraping and the application metrics listener
- Alertmanager queries and alert delivery
- Grafana dashboards and the authenticated Grafana proxy
- Loki/Alloy log collection and the monitoring-only exporters
- Structured observability event records emitted by the application

Retained components:

- Nginx `access.log`, `error.log`, 502/504 records, and `gateway-5xx.log`
- Scheduled channel monitor probes and their PostgreSQL history
- Manual channel tests, probe refresh SSE/Redis Pub/Sub, and circuit half-open probes
- Normal gateway request logs and billing/audit logs

## Runtime switches

The application reads these variables at startup:

```text
MONITORING_ENABLED=false
OBSERVABILITY_EVENT_LOG_ENABLED=false
ENABLE_METRICS=false
```

`MONITORING_ENABLED=false` is the primary switch. It bypasses the metrics
middleware, prevents the metrics listener from starting, skips Prometheus and
Alertmanager requests, and makes `/api/channel/failover/monitoring` return an
explicit `disabled` status. The channel probe runner does not read this flag.

`OBSERVABILITY_EVENT_LOG_ENABLED=false` is an independent guard for structured
event records. Keep it false when the complete monitoring chain is disabled.

`ENABLE_METRICS=false` remains a compatibility guard for deployments that
already use it. Both switches must allow metrics before the listener starts.

## Deployment sequence

1. Build and test the application locally.
2. Deploy the application to the Aliyun test host with the three variables set
   to `false`; do not copy local or production `.env` files.
3. Stop only the monitoring stack and monitoring-only systemd proxy/socket units.
   Never run `docker compose down -v`; data volumes are retained for rollback.
4. Verify the application health endpoint, authenticated API traffic, channel
   probes, and Nginx log writes.
5. Verify that Grafana, Prometheus, Alertmanager, Loki, Alloy, and their proxy
   ports are stopped or unreachable, and that the UI reports every disabled
   component.
6. Keep the previous compose files and image available for rollback. Re-enable
   the variables and restart the monitoring stack to restore the old behavior.

## Test plan

### Automated

- Go unit tests assert the disabled monitoring snapshot does not contact any
  configured monitoring source.
- Go tests assert disabled HTTP middleware does not create route metrics.
- Go tests assert structured observability events are not written when either
  shutdown guard is false.
- Existing channel routing, circuit, and channel monitor tests remain enabled.
- Frontend typecheck, lint, i18n sync, and production client build must pass.

### Aliyun test host

- Check container and systemd states before and after the change.
- Check ports `3001`, `3100`, `9090`, `9093`, `12345`, `8006`, `9100`, `9187`,
  and `9121`; no monitoring listener should accept connections.
- Open the admin monitoring page. It must show a disabled status for
  Prometheus, Alertmanager, Grafana, the metrics listener, and structured event
  logs; it must not poll or render empty metric cards.
- Open `/channel-monitors`, run a manual test, and wait for one scheduled probe.
  `last_checked_at`, history rows, and the refresh event must continue updating.
- Send one authenticated gateway request and confirm Nginx access/error logs
  still receive entries.
- Re-enable monitoring in a disposable test window and verify metrics and the
  monitoring page recover, then disable it again.

## User-visible verification

The monitoring tab now returns a stable status card instead of blank charts.
It explicitly shows which telemetry components are disabled, confirms Nginx
logs are retained, and confirms channel probes are still running. The channel
probe page remains the source of truth for probe results; the monitoring tab is
not used for probe scheduling or history.
