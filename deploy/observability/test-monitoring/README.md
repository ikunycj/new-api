# AllToken test monitoring

This lightweight stack is for the Aliyun test host only. It scrapes the
loopback-only new-api metrics endpoint and evaluates the profit margin warning
counter over a rolling two-hour window.

Before starting the stack, create `alertmanager.yml` from
`alertmanager.yml.template` and replace `__ALERT_EMAIL__` with the authorized
test recipient. Keep Prometheus and Alertmanager bound to loopback.

The new-api test environment must set:

```text
FAILOVER_PROMETHEUS_URL=http://alltoken-test-prometheus:9090
FAILOVER_ALERTMANAGER_URL=http://alltoken-test-alertmanager:9093
```

Validate the configuration before startup:

```bash
docker compose config --quiet
docker run --rm --entrypoint /bin/promtool \
  -v "$PWD/prometheus.yml:/etc/prometheus/prometheus.yml:ro" \
  -v "$PWD/alerts.yml:/etc/prometheus/alerts.yml:ro" \
  prom/prometheus:v3.2.1 check config /etc/prometheus/prometheus.yml
docker run --rm --entrypoint /bin/amtool \
  -v "$PWD/alertmanager.yml:/config/alertmanager.yml:ro" \
  prom/alertmanager:v0.28.1 check-config /config/alertmanager.yml
```
