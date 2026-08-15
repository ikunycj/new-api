# ikun 可观测性组件

该目录用于在 ikun 主机部署只读观测组件，不修改或重启 `sub2api`、OpenResty及其数据库。

## 安全与容量边界

- Loki、Alloy UI、node-exporter 仅映射至宿主机 loopback：`127.0.0.1:3100`、`127.0.0.1:12345`、`127.0.0.1:9100`。
- Loki retention 为 7 天（`168h`）。
- Alloy journald `max_age` 为 5 分钟，OpenResty 文件采集使用 `tail_from_end = true`，部署时不回灌历史日志。
- 仅采集 `sub2api.service` journald 与 `/opt/1panel/www/sites/ikun.love/log/error.log`；不采集 access log。
- `request_id`、`client_request_id`、模型和状态等字段进入 structured metadata/JSON，不作为 Loki 索引 label。
- 磁盘守卫每 5 分钟检查一次：Loki 数据超过 15 GiB，或根盘可用空间低于 20 GiB时，停止 `alloy` 和 `loki`，保留 `node-exporter`。

## 部署

目标目录固定为 `/opt/alltoken-observability/ikun`。部署前记录：

```sh
df -h /
du -sh /var/log/journal /opt/1panel/www/sites/ikun.love/log/error.log
ss -lntp | grep -E ':(3100|9100|12345)\b' || true
```

上传本目录文件后执行：

```sh
cd /opt/alltoken-observability/ikun
mkdir -p data/alloy
install -d -o 10001 -g 10001 -m 0750 data/loki
chmod 0755 disk-guard.sh

docker compose config --quiet
docker compose pull
docker compose up -d

install -m 0644 alltoken-observability-disk-guard.service /etc/systemd/system/
install -m 0644 alltoken-observability-disk-guard.timer /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now alltoken-observability-disk-guard.timer
```

## 验证

```sh
curl -fsS http://127.0.0.1:3100/ready
curl -fsS http://127.0.0.1:9100/metrics | grep -m1 '^node_exporter_build_info'
curl -fsS http://127.0.0.1:12345/-/ready

docker compose ps
docker compose logs --tail=100 loki alloy
systemctl status alltoken-observability-disk-guard.timer --no-pager
systemctl start alltoken-observability-disk-guard.service
```

验证新日志而非历史回灌：

```sh
curl -G -fsS http://127.0.0.1:3100/loki/api/v1/query_range \
  --data-urlencode 'query={job="ikun/sub2api"}' \
  --data-urlencode 'limit=5'
```

## 回滚

日志采集可独立回滚，不影响 `sub2api`、OpenResty或 node-exporter：

```sh
cd /opt/alltoken-observability/ikun
docker compose stop alloy loki
systemctl disable --now alltoken-observability-disk-guard.timer
```

完整回滚：

```sh
cd /opt/alltoken-observability/ikun
docker compose down
rm -f /etc/systemd/system/alltoken-observability-disk-guard.service \
      /etc/systemd/system/alltoken-observability-disk-guard.timer
systemctl daemon-reload
```

默认保留 `data/loki` 以便调查；只有用户明确确认后才可删除数据目录。
