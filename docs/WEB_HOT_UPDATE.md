# 前端热更新与核心版本升级说明

音屿安装助手严格区分两条升级轨道，不把前端热更新冒充为完整核心更新：

1. **核心服务升级（Docker 镜像 / 飞牛 FPK）**：Go 后端、API、配对/签名/续签内核（`plumesign`）、`usbmuxd2`、原生依赖、权限与生命周期脚本变更，均须通过升级完整 Docker 镜像或飞牛应用中心 FPK 完成。核心版本检查指向 `songdone/songlib-tv-installer` GitHub Releases，并在工作台与设置页按当前部署形态（飞牛 fnOS / 通用 Docker）给出对应升级指引。
2. **签名前端快照热更新（Web Hot Update）**：仅替换经 Ed25519 签名的完整前端构建快照，不影响后端逻辑、数据库、配对记录、Apple 账户或管理密码。

## 安全与事务机制

- **完整快照 + Ed25519 验签**：清单上限 `256 KiB`，最多 `512` 个文件，单文件上限 `8 MiB`，快照合计上限 `32 MiB`。签名覆盖 `payload` 的精确原始 JSON 字节（禁止重排或缩进），修订号为 `SHA256(payload)`。
- **后端精确匹配与防降级**：要求 `backend_id` 与当前运行二进制的 `SHA256(Version + "\0" + Commit + "\0" + BuildDate)` 完全一致，且满足 `min_backend_version`；单调递增 `sequence` 持久化于 `state.json`，回退不会降低最高序号，拒绝降级与同序号冲突载荷。
- **路径白名单与语法校验**：仅允许 `index.html`、根目录品牌图、`assets/`（JS/CSS/字体/图）及 `tv/`、`plan/`、`img/` 静态资源；下载与启动重验均使用 `O_NOFOLLOW` 目录描述符遍历，逐文件校验大小、SHA-256，并用进程内 `esbuild` 解析器校验全部 JS/CSS 语法。
- **原子切换与自动恢复**：下载至 `<DataDir>/web-updates/.staging-<rand>`，全部验证并 `fsync` 后改名为不可变版本目录 `versions/<revision>`，最后原子替换单一 `state.json`。启动时重验活动/上一版快照；活动版损坏自动回退上一版或随包 `go:embed` 原版；核心升级后 `backend_id` 变化自动复位到新包原版界面。
- **任务互斥与认证**：所有写操作受管理会话 Cookie、同源检查、`Sec-Fetch-Site` 与 `X-CSRF-Token` 保护，并与交互式 WebSocket 会话（安装/配对/登录/扫描）及后台安装/续签队列严格互斥。
- **独立恢复页**：`/management/web-update/recovery` 由后端 Go 模板直接输出，不依赖任何热更新前端脚本；若运行中活动 `index.html` 损坏，根路径直接返回 HTTP 503 并给出恢复页链接。

## 运行时 / 编译时公开配置

私钥绝不进入仓库、镜像、FPK、NAS 数据目录或日志。以下三项公开配置默认全空（保持纯离线安全默认）：

```text
SONGLIB_WEB_UPDATE_FEED=https://<受信任更新站点>/<backend_id>/manifest.json
SONGLIB_WEB_UPDATE_KEY_ID=<公开签名标识>
SONGLIB_WEB_UPDATE_PUBLIC_KEY=<Base64编码的32字节Ed25519公钥>
```

本机开发调试时如需通过 loopback HTTP 验证，可额外设置 `SONGLIB_WEB_UPDATE_ALLOW_HTTP=1`（仅允许 `localhost` / loopback IP，禁止重定向与压缩响应）。

## 离线生成签名快照

先在离线环境生成独立的音屿 Ed25519 PKCS8 私钥（切勿复用其他项目密钥）：

```bash
openssl genpkey -algorithm ED25519 -out /secure/offline/songlib-web-signing.pem
chmod 600 /secure/offline/songlib-web-signing.pem
```

构建前端后，使用离线工具打包并签名到全新输出目录：

```bash
cd web/static && npm ci && npm test && npm run build
cd ../..
go run ./cmd/songlib-web-release \
  -web web/static/dist \
  -out build/ui-release-1 \
  -version 1.1.0-ui.1 \
  -sequence 1 \
  -min-backend 1.1.0 \
  -backend-id <运行实例返回的backend_id> \
  -key-id songlib-web-2026 \
  -signing-key /secure/offline/songlib-web-signing.pem
```

工具仅在本地写出 `files/` 与 `manifest.json`，不联网、不覆盖已有目录，并在标准输出打印 `public_key`、`key_id`、`backend_id` 与 `revision`。
