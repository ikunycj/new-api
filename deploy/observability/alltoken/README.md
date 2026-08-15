# AllToken 可观测性组件

该目录在 AllToken ECS 上部署独立 Loki/Alloy，不修改或重启生产 `new-api`、PostgreSQL、Redis、OpenResty及现有 exporters。

## 边界

- Loki 与 Alloy UI 仅发布到 `127.0.0.1:3100`、`127.0.0.1:12345`。
- Loki retention 为 7 天，接受窗口为 10 分钟；Alloy从当前 Docker stream开始采集，不回灌历史容器日志。
- Docker discovery严格筛选容器名 `new-api`，不采集 PostgreSQL、Redis、OpenResty或 exporter日志。
- `request_id`、client trace、upstream ID、channel ID和model仅作为 structured metadata/JSON字段，不作为索引label。
- 磁盘守卫每5分钟检查：Loki数据超过15 GiB或根盘可用低于20 GiB时停止Alloy/Loki。

## 部署

目标目录固定为 `/opt/alltoken-observability/alltoken`：

```sh
cd /opt/alltoken-observability/alltoken
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
curl -fsS http://127.0.0.1:12345/-/ready
docker compose ps
systemctl start alltoken-observability-disk-guard.service
curl -G -fsS http://127.0.0.1:3100/loki/api/v1/series \
  --data-urlencode 'match[]={job="alltoken/new-api"}'
```

在新版本 `new-api` 部署前，Loki只能看到原有普通日志；部署后应出现 `observability_event=`并可按JSON字段下钻。

## 回滚

```sh
cd /opt/alltoken-observability/alltoken
docker compose down
systemctl disable --now alltoken-observability-disk-guard.timer
rm -f /etc/systemd/system/alltoken-observability-disk-guard.service \
      /etc/systemd/system/alltoken-observability-disk-guard.timer
systemctl daemon-reload
```

默认保留 `data/loki`，只有用户明确确认后才删除。
