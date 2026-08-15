# AllToken production monitoring

This stack runs Prometheus, Alertmanager, and Grafana on the AllToken ECS using
host networking. Prometheus and Alertmanager listen on the Docker bridge address
(`172.17.0.1` by default), so the AllToken application container can query them
without exposing them on the public interface. Grafana listens on `127.0.0.1`
and is exposed only through the authenticated `/grafana/` location on
`alltokenapi.com`.

The AllToken application container reaches Prometheus and Alertmanager through
`host.docker.internal`, which is provided by `docker-compose.1panel.yml`.

Deploy to `/opt/alltoken-production-monitoring`, validate with
`docker compose config --quiet`, then start with `docker compose up -d`.

Verify that the host Docker bridge is `172.17.0.1` before deploying. The
Prometheus alert target and Grafana data sources use the same bridge address.
Keep host firewall rules closed for TCP ports `9090` and `9093`; these listeners
are intended only for container-to-host traffic.
