# 191 测试服务器（191.40.44.196）登录与部署流程

> 范围：**仅测试服务器 `191.40.44.196`**，公网端口 `3001`。部署模型与
> `development-deployment-aliyun-test.md` 完全一致——bind-mount 换二进制，**不重建镜像**。
> 不要把它用到生产（`ssh alltokenapi`）。

## 部署模型

宿主机 `/opt/new-api/` 下是 `docker-compose.yml` + `docker-compose.override.yml`。
override 把宿主机 `./new-api-current.bin` bind-mount 到容器 `/new-api`，镜像
`new-api:local` 只提供运行时（debian + ca-certificates）。真正在跑的二进制是
宿主机文件，通过软链指向：

```
new-api-current.bin -> new-api-<描述>-<shortsha>.bin
```

部署 = 上传新二进制 → `chmod +x` → 原子重指软链 → `docker compose ... force-recreate new-api`。
旧二进制留在磁盘上，回滚只需把软链指回去。

## 两条路：SSH 免密 / 手动 scp

### 前提（一次性）

- Windows 本机有 `~/.ssh/id_ed25519`，其公钥已加到 191 的 `root` 用户
  `~/.ssh/authorized_keys`。
- 免密自检：
  ```bash
  ssh -o BatchMode=yes root@191.40.44.196 true
  ```

### 路 A：SSH 免密（正常首选）

前置：先按 `development-deployment-aliyun-test.md` 的 Step 2/3 编出 `build/$NAME.bin`。

1. 上传：
   ```bash
   scp -o BatchMode=yes /d/new-api-master/build/$NAME.bin root@191.40.44.196:/opt/new-api/$NAME.bin
   ```

2. 落地 + 换链 + 重建（一条 ssh heredoc）：
   ```bash
   ssh -o BatchMode=yes root@191.40.44.196 bash -s <<'REMOTE'
   set -Eeuo pipefail
   cd /opt/new-api
   chmod +x $NAME.bin                       # scp 上来会丢执行位，必须补
   sha256sum $NAME.bin                      # 必须等于本地 $NAME.bin 的 SHA
   ln -sfn $NAME.bin new-api-current.bin.inflight
   mv -T new-api-current.bin.inflight new-api-current.bin
   docker compose --env-file .env up -d --no-build --no-deps --force-recreate new-api
   REMOTE
   ```

3. 验证：
   ```bash
   ssh -o BatchMode=yes root@191.40.44.196 bash -s <<'REMOTE'
   docker ps --filter name=new-api --format '{{.Names}} | {{.Status}}'
   curl -s -o /dev/null -w 'HTTP %{http_code}\n' http://127.0.0.1:3001/api/status   # 期望 200
   docker exec new-api sha256sum /new-api    # 必须等于本地 $NAME.bin 的 SHA
   ls -la new-api-current.bin
   REMOTE
   ```

### 路 B：手动 scp + 手动在线命令（SSH 被封 / 无免密时）

当 `ssh root@191.40.44.196` 报 `Connection reset by peer` 或一直卡住时，通常是本机
公网 IP 被 191 的 fail2ban 封了（本机公网 IP `219.143.23.203` 曾因此被封，高频 ssh
探测会触发）。

1. 解封（登录云控制台 → 打开 191 的网页终端 / VNC）：
   ```bash
   fail2ban-client set sshd unbanip 219.143.23.203
   # 不确定是不是 fail2ban：fail2ban-client status sshd
   ```

2. 上传二进制（能 scp 就用 scp，否则用云控制台网页上传到 `/opt/new-api/`）：
   ```bash
   scp D:\new-api-master\build\$NAME.bin root@191.40.44.196:/opt/new-api/$NAME.bin
   ```

3. 登录服务器（`ssh root@191.40.44.196` 或云网页终端），逐条执行：
   ```bash
   cd /opt/new-api
   chmod +x $NAME.bin
   sha256sum $NAME.bin
   ln -sfn $NAME.bin new-api-current.bin.inflight
   mv -T new-api-current.bin.inflight new-api-current.bin
   docker compose --env-file .env up -d --no-build --no-deps --force-recreate new-api
   ```

4. 手动验证：
   ```bash
   docker ps --filter name=new-api --format '{{.Names}} | {{.Status}}'
   curl -s -o /dev/null -w 'HTTP %{http_code}\n' http://127.0.0.1:3001/api/status
   docker exec new-api sha256sum /new-api
   ```

## 常见坑

- **scp 丢执行位**：Windows scp 上去的 `.bin` 是 644，容器执行 `/new-api` 会报
  `permission denied`。`chmod +x` 必做。
- **fail2ban**：高频 ssh 探测会封 Windows IP，症状是 `Connection reset by peer`。
  按路 B 第 1 步解封。
- **端口**：191 对公网是 `3001`，容器内仍是 `3000`。宿主机健康检查用
  `http://127.0.0.1:3001/api/status`。

## 安全红线

- 禁止 `docker compose up --build`、`docker system df`（内存受限，曾导致 dockerd OOM）。
- 禁止在环境之间复制 / 同步 `.env`。
- 软链用 `ln -sfn ... .inflight && mv -T` 原子换，不要直接写软链。
- 保留活跃 bin 以及被 `.rollback-*` 引用链上的所有 bin，不要随手删。
