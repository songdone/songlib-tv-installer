# fnOS 验证依据 · 0.1.1

本轮重新读取官方文档并保存快照：
- 应用框架/生命周期：https://developer.fnnas.com/docs/core-concepts/framework/
- Docker应用：https://developer.fnnas.com/docs/examples/docker/
- 图标：https://developer.fnnas.com/docs/core-concepts/icon/

官方说明install_init在安装文件应用前执行，install_callback在应用后执行；同时对照已实测的听己prepare脚本与FrpDeck端口写入方法。初期不能向尚未授权的应用var目录写入。0.1.1删除Compose的env_file依赖；init只校验，callback生成私有数据与一次性密码文件，应用写出哈希后删除一次性明文。升级不重置既有密码/配对/端口。

图标来自既有音屿iOS应用1024源图，仅缩放、添加透明圆角，不重绘Logo、不改配色。用户要求的256与512素材分别为branding/ICON.PNG、branding/ICON_512.PNG；包内ICON_256.PNG与app/ui/images/icon_512.png均保留对应尺寸。官方ICON.PNG字段明确为64，build在临时目录另生成64兼容缩略图，不修改256/512母版。

官方fnpack1.2.3 build负责正式结构校验（该版本无独立check命令），check.py补充PNG尺寸、sRGB、圆角透明通道、JSON与最小权限检查。test_lifecycle.py验证var不存在时Compose校验、首次安装/配置写入、特殊字符密码、升级保留数据及端口。

仍需用户设备复测；本轮不直接操作NAS、不安装、不部署、不送审。当前为配套镜像预加载的本地验收包，不能声称已发布公共镜像。宿主缺少mDNS socket时不要求提高权限，使用上游手动IP路径。
