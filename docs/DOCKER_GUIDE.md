# 通用 Docker / NAS 部署指南

本文档面向 **群晖（Synology）、极空间（ZSpace）、威联通（QNAP）、绿联（UGREEN）、Unraid 以及通用 Linux 主机** 的 Docker 部署场景。

> 如使用飞牛 fnOS，优先推荐直接通过飞牛应用中心安装 FPK（`SongLib-TV-Installer`），也可按本文档使用通用 Docker Compose 方式独立部署。

## 1. 快速启动（推荐 Docker Compose）

多架构镜像 `666uos/songlib-tv-installer:latest` 同时包含 `linux/amd64` 与 `linux/arm64`，Docker 会自动拉取匹配架构。

使用仓库根目录的 [docker-compose.yml](../docker-compose.yml)：

```bash
# 可在启动前自定义端口与初始管理密码（8–64 位）
SONGLIB_PORT=15533 SONGLIB_ADMIN_PASSWORD=your-strong-password docker compose up -d
```

启动后在浏览器打开 `http://<NAS局域网IP>:15533` 登录。

- 默认端口 `15533`（避开上游 ATVLoadly 常用的 `5533`）。
- 首次启动如未指定 `SONGLIB_ADMIN_PASSWORD`，Compose 默认使用 `songlib123` 初始化密码哈希（落盘于 `./data/keys/management-password.hash`）。
- **强烈建议**：首次登录后立即进入网页 **「设置 › 管理登录密码」** 修改为个人密码；密码哈希落盘后，环境变量不再覆盖已保存密码。

## 2. 局域网自动发现 vs 手动 IP 配对

- **默认通用模式（零宿主机依赖）**：`docker-compose.yml` 默认设置 `ZEROCONF_ENABLED=false`，不挂载 `/var/run/dbus` 或 `/var/run/avahi-daemon`，避免在群晖、极空间、macOS 等未开放宿主机 Avahi socket 的设备上启动失败。在网页「Apple TV 设备」页直接输入 Apple TV 局域网 IP（或 Tailscale IP）即可完成配对与安装，已配对的手动设备在容器重启后会自动恢复加载。
- **Linux 宿主机启用 mDNS 自动发现（可选）**：若宿主机已运行 `avahi-daemon` 且存在 `/var/run/dbus/system_bus_socket` 与 `/var/run/avahi-daemon/socket`，可叠加 [docker-compose.avahi.yml](../docker-compose.avahi.yml)：

```bash
docker compose -f docker-compose.yml -f docker-compose.avahi.yml up -d
```

## 3. 各品牌 NAS 面板配置要点

如果不使用命令行，在群晖 Container Manager / 极空间 Docker / 威联通 Container Station / 绿联 Docker 界面创建容器时，只需填以下 4 项：

| 配置项 | 推荐值 | 说明 |
| --- | --- | --- |
| 镜像 | `666uos/songlib-tv-installer:latest` | 自动匹配 x86_64 / ARM64；也可固定为具体版本标签（如 `:1.1.0`） |
| 端口映射 | 宿主机 `15533` → 容器 `80`（TCP） | 宿主机端口可改为任意未占用端口 |
| 存储卷挂载 | 宿主机持久化目录 → 容器 `/data` | 保存管理密码哈希、配对记录、Apple 账户、已安装应用与热更新快照 |
| 环境变量 | `SONGLIB_DEPLOY_MODE=docker`<br>`SONGLIB_ADMIN_PASSWORD=<8-64位初始密码>`<br>`ZEROCONF_ENABLED=false`<br>`TZ=Asia/Shanghai` | 不需要开启特权模式（Privileged）或 `seccomp:unconfined` |

## 4. 升级与备份

### 升级核心版本（镜像更新）

后端服务、签名内核（`plumesign`）、配对协议与依赖更新属于核心升级，无法通过前端热更新覆盖。备份 `./data` 后执行：

```bash
docker compose pull
docker compose up -d
```

升级不会重置管理密码、已配对设备和已安装应用列表。若之前激活过前端热更新快照，核心升级后系统会自动检测 `backend_id` 变化并切换为新镜像自带的原版界面。

### 界面热更新与独立恢复

若运维环境配置了签名发布源（详见 [WEB_HOT_UPDATE.md](./WEB_HOT_UPDATE.md)），可在网页 **「设置 › 核心版本与界面热更新」** 检查并原子切换经过 Ed25519 签名的界面快照。异常时可直接访问 `/management/web-update/recovery` 独立恢复页回退上一版或恢复原版界面。

### 反向代理与域名访问

通过 Nginx / Caddy / Lucky / frp 等反向代理以 HTTPS 暴露给自用设备时，请确保透传 `Host` 与 `X-Forwarded-Proto: https` 请求头（供同源校验与 `Secure` Cookie 判断），并可按需设置 `SONGLIB_ALLOWED_HOSTS=你的域名` 开启额外的 Host 白名单加固。
