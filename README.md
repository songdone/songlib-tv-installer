# 音屿-Apple TV音乐播放器安装助手

**你的音乐 自成一屿** —— 这是 [音屿 SongLib Amp](https://songlib.playsong.cn/)（Apple TV 私人音乐播放器）的官方安装助手：一个免费、开源的 Apple TV IPA 自签与自动续签工具，提供 **飞牛 fnOS 应用中心版** 与 **通用 Docker / NAS 版**（支持群晖、极空间、威联通、绿联、Unraid 及通用 Linux）。

基于 [ATVLoadly](https://github.com/bitxeno/atvloadly)（AGPL-3.0）改造；签名与续签内核来自上游项目，上游版权与对应源码义务完整保留。

> 音屿播放器是独立的商业产品，与安装助手分开维护：播放器客户端、授权服务与官网在私有仓库开发。安装助手永久免费，安装其他合法 tvOS IPA 不需要购买音屿。

## 功能

- 自动发现同一局域网中处于配对模式的 Apple TV，支持手动 IP 与 Tailscale（无 Avahi 环境下重启亦可自动恢复已配对手动设备）
- 引导式配对流程、设备管理（支持移除任意已连接设备）、tvOS IPA 签名与安装
- 免费 Apple 账户签名约 7 天有效，到期前自动尝试续签；支持在续签证书满额时按需选择注销证书重试
- 支持多 Apple 账户、外部证书（P12 + 描述文件）签名及长上传保活
- 跟踪 GitHub Releases / AltStore 来源，手动或自动更新已安装应用
- 独立管理密码保护（支持网页内直接修改密码）、核心版本更新提醒与 Ed25519 签名前端热更新（含独立恢复页 `/management/web-update/recovery`）
- 简体中文为默认界面语言，保留用户手动切换的多语言偏好，支持浅色 / 深色 / 跟随系统主题

## 界面

<p>
  <img width="420" src="./marketing/tv/demo-cinematic-tv-playing.jpg" alt="音屿 Apple TV 播放页（官方截图）">
  <img width="420" src="./marketing/tv/demo-cinematic-tv-lyrics.jpg" alt="音屿 Apple TV 全屏歌词（官方截图）">
</p>

## 安装

- **飞牛 fnOS 版**：在飞牛应用中心搜索安装，或下载最新 `SongLib-TV-Installer-*.fpk`（单包支持 x86_64 / ARM64，镜像由 DockerHub 自动匹配）。管理端口默认 `15533`，初始密码默认 `songlib123`。
- **通用 Docker / NAS 版**：使用 [docker-compose.yml](./docker-compose.yml) 拉取多架构镜像 `666uos/songlib-tv-installer:latest`（x86_64 / ARM64），完整部署与升级步骤见 [docs/DOCKER_GUIDE.md](./docs/DOCKER_GUIDE.md)。

运维、打包与生命周期管理见 [docs/OPERATIONS.md](./docs/OPERATIONS.md)；前端热更新与核心升级边界见 [docs/WEB_HOT_UPDATE.md](./docs/WEB_HOT_UPDATE.md)；安全与数据说明见 [docs/SECURITY.md](./docs/SECURITY.md)。

## 从源码构建

```bash
cd web/static && npm ci && npm test && npm run build   # 前端测试与构建
cd ../.. && go test ./...                              # Go 服务测试
python3 packaging/fnos/check.py                        # 飞牛 FPK 静态校验
python3 packaging/fnos/test_lifecycle.py               # 飞牛生命周期脚本测试
```

每个发布版本提供完整对应源码（应用内 `/source` 入口、`/license` 许可页与 Release 源码归档），可在 AGPL-3.0 下重新构建。

## 使用须知

- 建议注册专用的 Apple 账户小号用于自签，开启双重认证；不要使用存有照片与支付资料的主账号。
- 免费签名通常 7 天有效；自动续签依赖 NAS 在线、Apple TV 可达与 Apple 账户状态，不保证百分之百成功。
- 安装对象为 Apple TV。iPhone/iPad 路径仅为保留的底层兼容代码，未通过端到端验证，不作为已支持功能宣传。
- 交流与反馈：音屿用户 Telegram 群 https://t.me/+hvpn0ClEZB4xYTQ1

## 许可与致谢

- 本项目遵循 **AGPL-3.0**；上游 [ATVLoadly（bitxeno）](https://github.com/bitxeno/atvloadly) 的版权与历史提交完整保留，完整上游说明见 [docs/upstream/README.md](./docs/upstream/README.md)。
- 第三方组件声明见 [THIRD_PARTY_NOTICES.md](./THIRD_PARTY_NOTICES.md)。
- 音屿播放器截图为官方营销素材，来源见 [marketing/tv/PROVENANCE.md](./marketing/tv/PROVENANCE.md)。
