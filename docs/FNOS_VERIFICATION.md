# fnOS 验证依据

2026-11-02 HTTP读取 https://developer.fnnas.com/docs/examples/docker/ ，使用官方fnpack 1.2.3 Darwin arm64 `create --template docker`与`build`。该版本无独立check命令；build执行官方结构校验，另以packaging/fnos/check.py做补充检查。
官方模板ICON.PNG为64px，故安装包主文件64，256设计母版为branding/ICON.PNG，ICON_256.PNG为256，marketing/icon_512.png为512。
本地验收包pull_policy=never，需要先load对应镜像，不假装已发布registry镜像。fnOS检测向导类型、socket、UID映射与升级保留数据尚待设备实测。
