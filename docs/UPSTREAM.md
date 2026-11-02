# Source provenance

2026-11-02 本地衍生自 bitxeno/atvloadly，upstream commit `6029780292123ef6d0ee74e304c0f5d56e7330eb`，AGPL-3.0。
保留原 Git 历史和 LICENSE，不修改商业播放器，不附带 IPA。
运行依赖采用上游已构建多架构镜像固定 digest：
`bitxeno/atvloadly@sha256:55a3a594dbe345d1b9d76b54aa44226cec25dfb1f3d67e6d7537303bcb01b310`。
该基础镜像的上游依赖下载可追溯至其 Dockerfile，但 Apple 动态依赖与DeveloperDiskImages并非内容锁定；不能声称全链可复现。发布前须保留基础镜像、对应源码和外部二进制许可。

修改：管理身份认证、CSRF/Origin、下载约束、IPA结构限制、只读非root容器、品牌产品卡、fnOS生命周期与图标。完整代码在 FPK 对应源码目录；运行界面 `/source` 提供本构建源码压缩包，`/license` 提供许可。远端公开仓尚未发布，不虚构URL；对外分发前须公开对应源码与许可复核。

更新策略：保留 upstream remote，在独立分支 merge/rebase 后重新跑Go、前端、容器、fnpack与设备矩阵，不自动追最新tag。
