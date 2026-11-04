# 0.1.3 DockerHub通用版

取代0.1.0–0.1.2的预加载/离线方案。交付方式：一个platform=all的轻量FPK，引用DockerHub版本化多架构镜像666uos/songlib-tv-installer:0.1.3，由Docker自动选择amd64/arm64。
发布者与维护者均为6uos。固定规范在packaging/fnos/release-policy.json，check.py自动断言。

包内无镜像层、无临时registry/native helper。安装时自动从DockerHub拉取镜像；无需docker load。默认中文、8–64位密码与256/512圆角图标保留。

启动配置改用官方TRIM_UID/TRIM_GID；入口脚本等待密码初始化文件后才访问私有数据，避免资源启动早于callback。未获取192.168.31.185容器日志，不能把此项认定为用户现场的唯一故障原因。该地址5533和SSH22均拒绝连接，现有fnOS OAuth授权连接中没有安装助手容器；未修改NAS。

DockerHub匿名HTTP验证：公共tag存在、index含两种架构、每个远端配置digest与本地构建一致。具体index digest与两架构同包运行/登录结果存于内部验收归档（私有仓）。测试明确使用同一个最终FPK与同一镜像名。

源码镜像构建基线80a5ce7；后续提交只补文档与测试回执。未提交应用中心审核，未对用户NAS执行安装/升级。
