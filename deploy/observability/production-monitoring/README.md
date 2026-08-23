# AllToken production monitoring

Grafana, Prometheus, and Alertmanager run on the dedicated monitoring host.
The production host keeps Loki, Alloy, and the exporters close to the services
they observe. The hosts communicate only through WireGuard:

- production: `10.66.0.1`
- monitoring: `10.66.0.2`

Grafana listens on `10.66.0.2:3001`, Prometheus on `10.66.0.2:9090`, and
Alertmanager on `10.66.0.2:9093`. Production OpenResty proxies the authenticated
`/grafana/` route to Grafana. No monitoring listener is bound to the monitoring
host's public address.

Prometheus scrapes production metrics through the systemd socket proxies in
`systemd/`. The AllToken application still reaches Prometheus and Alertmanager
through `host.docker.internal`; the failover socket proxies forward those
Docker-bridge connections to the monitoring host.

## Monitoring host

Deploy `compose.remote-monitoring.yml` as `/opt/alltoken-grafana/compose.yml`,
along with `alerts.yml`, `alertmanager.yml`, `prometheus.remote.yml`, and the
`grafana` directory. The following external volumes must exist:

- `alltoken-remote-grafana_grafana_data`
- `alltoken-remote-monitoring_prometheus_data`
- `alltoken-remote-monitoring_alertmanager_data`

Validate the deployed stack with `docker compose config --quiet`, then start it
with `docker compose up -d`.

## Production host

Install the socket and service units from `systemd/`, then enable the socket
units. `compose.yml` deliberately contains no services so a routine Compose
operation cannot restart the migrated collectors. `compose.legacy.yml` retains
the old definitions for rollback, and the old data volumes remain in place.

Verify that the Docker bridge gateway is `172.17.0.1` before installing the
failover sockets. Keep all bridge and WireGuard listeners closed to public
traffic.
