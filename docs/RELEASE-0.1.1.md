# 0.1.1 中文、安装与图标修正

用户反馈：0.1.0默认界面跟随英文浏览器，许可页直接英文，管理密码强制16位且字符限制过窄；安装期Compose读取尚未生成的runtime.env报错；图标应使用既有音屿应用的深色金色配色、圆角及256/512规格。

修正：
- 默认zh-CN，回退同为中文，网页lang同步；忽略旧i18next英语缓存，保留显式语言切换。服务端默认中文，补齐Bark/Telegram字段中文。
- 安装许可先显示中文阅读说明，原AGPL英文原文不改动并完整保留。
- 密码8–64字符，无强制字符组合；支持常用符号、空格、中文。仅保留禁止换行的单行输入规则。新哈希先SHA-256再bcrypt以支持中文长度；既有bcrypt密码验证/升级保留。
- Compose不再引用runtime.env；install_init只验证，不写尚未交付的var目录。install_callback/upgrade_callback调用prepare，在正确阶段保存数据、端口与一次性initial-password文件。首次启动生成哈希后删除一次性明文。原runtime.env仅作旧数据迁移读取，不执行文件内容。
- 参考听己实际生命周期与FrpDeck端口渲染脚本，使用TRIM_PKGVAR等官方变量。缺失Avahi/D-Bus时回退手动IP流程，不增加privileged或Docker socket权限。
- 图标来自既有iOS1024源图，保留背景/图形/颜色，仅缩放和圆角透明处理。256/512主素材有嵌入sRGB profile；64为官方ICON.PNG字段兼容图，由build临时生成。

验证：Go全套、密码与旧哈希专项、前端测试、golangci-lint、脚本生命周期测试；fnpack两架构build。本机原生服务的设置页因缺少plumesign报错，不将其作为最终浏览器通过证据；最终浏览器验收使用配套Docker镜像。

未直接操作用户NAS；安装报错修复已覆盖本地“先校验Compose再生成数据”的复现流程，仍需用户飞牛设备复测。Apple TV签装/续签不在本次本地回归通过项。
