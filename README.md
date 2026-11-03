# 音屿-Apple TV音乐播放器安装助手

**你的音乐 自成一屿** —— 这是 [音屿 SongLib Amp](https://songlib.playsong.cn/)（Apple TV 私人音乐播放器）的官方安装助手：一个免费、开源的 Apple TV IPA 自签与自动续签工具，运行在飞牛 fnOS 或 Docker 上。

由 **6uos** 维护，基于 [ATVLoadly](https://github.com/bitxeno/atvloadly)（AGPL-3.0）改造；签名与续签内核来自上游项目，上游版权与对应源码义务完整保留。

> 音屿播放器是独立的商业产品，与安装助手分开维护：播放器客户端、授权服务与官网在私有仓库开发。安装助手永久免费，安装其他合法 tvOS IPA 不需要购买音屿。

## 功能

- 自动发现同一局域网中处于配对模式的 Apple TV，支持手动 IP 与 Tailscale
- 引导式配对流程，tvOS IPA 签名与安装
- 免费 Apple 账户签名约 7 天有效，到期前自动尝试续签
- 支持多 Apple 账户、外部证书（P12 + 描述文件）签名
- 跟踪 GitHub Releases / AltStore 来源，手动或自动更新
- 简体中文为默认界面语言，支持多语言与浅色/深色主题

## 界面

<p>
  <img width="420" src="./marketing/tv/demo-cinematic-tv-playing.jpg" alt="音屿 Apple TV 播放页（官方截图）">
  <img width="420" src="./marketing/tv/demo-cinematic-tv-lyrics.jpg" alt="音屿 Apple TV 全屏歌词（官方截图）">
</p>

## 安装

- **飞牛 fnOS**：安装 `SongLib-TV-Installer` FPK（当前版本见 [docs/RELEASE-0.1.3.md](./docs/RELEASE-0.1.3.md)），镜像从 DockerHub 多架构自动匹配，无需预加载。
- **Docker**：`666uos/songlib-tv-installer`，x86_64 / ARM64。Linux 主机需要 `avahi-daemon`。

构建、运行与生命周期管理见 [docs/OPERATIONS.md](./docs/OPERATIONS.md)；安全与数据说明见 [docs/SECURITY.md](./docs/SECURITY.md)。

## 从源码构建

```bash
cd web/static && npm ci && npm run build   # 前端
cd ../.. && go test ./...                  # Go 服务
python3 packaging/fnos/check.py            # FPK 校验
```

每个发布版本提供完整对应源码（应用内 `/source` 入口与构建归档），可在 AGPL-3.0 下重新构建。

## 使用须知

- 建议注册专用的 Apple 账户小号用于自签，开启双重认证；不要使用存有照片与支付资料的主账号。
- 免费签名通常 7 天有效；自动续签依赖 NAS 在线、Apple TV 可达与 Apple 账户状态，不保证百分之百成功。
- 安装对象为 Apple TV。iPhone/iPad 路径仅为保留的底层兼容代码，未通过端到端验证，不作为已支持功能宣传。

## 许可与致谢

- 本项目遵循 **AGPL-3.0**；上游 [ATVLoadly（bitxeno）](https://github.com/bitxeno/atvloadly) 的版权与历史提交完整保留，完整上游说明见 [docs/upstream/README.md](./docs/upstream/README.md)。
- 第三方组件声明见 [THIRD_PARTY_NOTICES.md](./THIRD_PARTY_NOTICES.md)。
- 音屿播放器截图为官方营销素材，来源见 [marketing/tv/PROVENANCE.md](./marketing/tv/PROVENANCE.md)。
