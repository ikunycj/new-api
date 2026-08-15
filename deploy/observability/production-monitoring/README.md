# AllToken production monitoring

This stack runs Prometheus, Alertmanager, and Grafana on the AllToken ECS using
host networking. Every service listens on `127.0.0.1`; Grafana is exposed only
through the authenticated `/grafana/` location on `alltokenapi.com`.

The AllToken application container reaches Prometheus and Alertmanager through
`host.docker.internal`, which is provided by `docker-compose.1panel.yml`.

Deploy to `/opt/alltoken-production-monitoring`, validate with
`docker compose config --quiet`, then start with `docker compose up -d`.
