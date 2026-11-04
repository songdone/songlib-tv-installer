# 安全策略

## 报告漏洞

如果你发现安全问题（可导致未授权访问、签名滥用、数据泄露等），请**不要**在
公开 Issue 中描述细节：

1. 在 GitHub 上新建一个标题为 `[security] <概述>` 的私密安全咨询
   （Security → Report a vulnerability），或
2. 发邮件至 support@playsong.cn，标题注明 `[security]`。

我们会在 72 小时内确认，修复后视情况发布补丁版本并在 Release 说明中致谢。

## 安全设计速览

完整说明见 [docs/SECURITY.md](./docs/SECURITY.md)。要点：

- 管理后台默认无密码，首次启动生成随机构建密码与 TOTP；bcrypt 哈希存储，
  会话仅内存保存（1 小时失效），全程 HttpOnly / SameSite=Strict / CSRF 校验。
- 下载白名单、IPA 结构校验、日志脱敏（邮箱 / IP / 授权码 / password 类字段）。
- 容器只读、非 root（UID 10001）、cap_drop ALL、no-new-privileges。

## 范围

本仓库只包含安装助手的开源代码与打包脚本；签名证书、Apple 账户、商户凭据
与播放器商业组件不在本仓库，也不应被提交。若发现疑似泄露，请按上述方式
立即报告。
