# 安全与数据

默认管理密码首次安装设置，不预置密码；bcrypt hash、0600权限，随机会话仅内存保存，1小时失效。HttpOnly/SameSite Strict，同源Origin和CSRF约束，WebSocket也校验Origin。首次密码保存在fnOS受保护runtime.env以完成初始化，初始化完成后可清空该值，仅保留password hash；不要丢失整个keys目录。

容器非root，去掉capabilities，禁止提权、只读根目录，不挂Docker socket。仅挂D-Bus/Avahi两个必要socket和应用私有var/data。宿主socket可访问性与局域网发现须真机证明，缺失即清晰失败，不自动增加privileged。

Apple账户、配对、证书私钥、IPA放持久目录，umask077；上游外部证书私钥使用部署密钥加密。账户登录态仍依赖上游及严格文件权限，不宣称全部账户数据已加密。

上传总512MiB、最多5个IPA、ZIP条目及展开体积限制；受限HTTPS主机白名单、DNS解析拒绝私网并固定连接IP、重定向重新校验、下载限512MiB。导入不会自动安装，需要明确点击签装。

禁止直接暴露公网。HTTP仅可信LAN；跨公网需要另行TLS和访问控制设计。日志只记录路径不记录query/body，不提供原始日志的无鉴权访问。真实Apple账户路径和诊断脱敏仍需真机复核。
