# 贡献指南

感谢关注音屿安装助手。这是一个面向个人用户的开源工具（AGPL-3.0），用于在
NAS 上为 Apple TV 签名并安装 tvOS 应用。

## 提交内容边界

本仓库是公开仓库，请确认你的 PR **不包含**以下内容：

- 内部开发计划、交接文档、检查点或验收记录（这类资料保存在私有仓库）。
- API 密钥、Token、Apple 账户凭据、签名证书、`.p12`、数据库文件。
- 16688 等销售平台的后端代码、商户凭据或订单数据。
- 未发布的商业功能代码或私有播放器组件。
- 真实用户的设备信息、邮箱或日志。

允许并欢迎的内容：Bug 修复、兼容性改进、构建脚本改进、文档纠错、
国际化补充、安全加固。

## 流程

1. 从 `main` 拉出功能分支，一个 PR 聚焦一件事。
2. 提交前运行：
   - `cd web/static && npm test && npm run build`
   - `go test ./...`
3. 提交信息用英文，格式 `type(scope): summary`（feat / fix / docs / chore）。
4. CI 通过后等待 review；改动签名与授权逻辑时请说明影响面。

## 许可

提交即表示你同意以 **AGPL-3.0** 许可贡献这些代码。上游 ATVLoadly 的版权
信息必须保留，详见 [docs/UPSTREAM.md](./docs/UPSTREAM.md) 与
[docs/OPEN_SOURCE_COMPLIANCE.md](./docs/OPEN_SOURCE_COMPLIANCE.md)。
