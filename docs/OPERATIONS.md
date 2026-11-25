# 运维、构建与打包说明

本项目一套源码同时产出两个维护目标：

1. **飞牛 fnOS 版（FPK）**：单包 `platform=all`（体积小于 2 MiB），运行由 DockerHub 拉取的多架构镜像 `666uos/songlib-tv-installer:<version>`；维护者与发布者均为 `6uos`，群链接为 `https://t.me/+hvpn0ClEZB4xYTQ1`。包内不含镜像层、离线源或商业播放器 IPA。
2. **通用 Docker / NAS 版**：同一多架构镜像 `666uos/songlib-tv-installer`（`linux/amd64` + `linux/arm64`），配套根目录 [docker-compose.yml](../docker-compose.yml) 与可选 Avahi 叠加配置 [docker-compose.avahi.yml](../docker-compose.avahi.yml)，详见 [DOCKER_GUIDE.md](./DOCKER_GUIDE.md)。

## 构建流程

1. 构建并测试前端：
   ```bash
   cd web/static && npm ci && npm test && npm run build
   ```
2. 生成对应源码归档（打入容器 `/opt/songlib-source.tar.gz`，供 `/source` 路由履行 AGPL 义务）：
   ```bash
   mkdir -p build && git archive --format=tar.gz -o build/songlib-source.tar.gz HEAD
   ```
3. 运行 Go 测试与跨平台静态编译（`CGO_ENABLED=0`，`linux/amd64` + `linux/arm64`），使用根目录 `Dockerfile` 构建并推送多架构镜像（或由 GitHub Actions `v*` 标签触发 `.github/workflows/release.yml`）。
4. 打包飞牛 FPK（先确认对应多架构镜像已发布并可匿名拉取）：
   ```bash
   python3 packaging/fnos/build.py \
     --fnpack /path/to/fnpack \
     --image 666uos/songlib-tv-installer:<version> \
     --output build/<build-id>/SongLib-TV-Installer-<version>.fpk
   ```

## 静态与生命周期校验

```bash
python3 packaging/fnos/check.py
python3 packaging/fnos/test_lifecycle.py
```

- 图标门禁：`ICON.PNG` 256×256、`ICON_256.PNG` 256×256、`ICON_512.PNG` 512×512，且 UI 目录提供 `icon_64.png`、`icon_256.png`、`icon_512.png`。
- 端口与默认密码：飞牛默认管理端口 `15533`（容器内部 `:80`），缺省向导初始密码 `songlib123`（可在飞牛应用设置或网页「设置 › 管理登录密码」修改）。

## 数据与诊断

- 飞牛数据目录：`${TRIM_PKGVAR}/data`；通用 Docker 数据目录：挂载至容器 `/data`。保存管理密码哈希、配对记录、Apple 账户、应用列表与前端热更新快照（`web-updates/`）。升级前先备份数据目录；升级保留原密码、配对与端口。
- 诊断顺序：核对容器运行状态与日志 → 检查宿主机映射端口（默认 `15533`）与 `GET /ready` 返回 200 → 登录页面核对运行状态。本地构建与打包校验不能替代目标设备实机验收。
- 热更新发布工具与恢复页见 [WEB_HOT_UPDATE.md](./WEB_HOT_UPDATE.md)。
