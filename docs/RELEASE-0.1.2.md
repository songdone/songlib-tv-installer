# 0.1.2 完整离线FPK

## 修复的问题
0.1.1 的两个 FPK 引用了只存在于开发机的镜像名，并配置 pull_policy=never。用户直接安装会遇到“No such image”。把预加载步骤留给用户，不符合直接安装交付的目标。

## 方案与权限
FPK在cmd/offline-image内携带完整镜像blobs/config/manifest，并包含对应架构的只读镜像源小程序。安装初始化阶段先在127.0.0.1启动临时镜像源，飞牛自身的Docker项目资源按固定manifest digest拉取。Compose改为missing：本机没有镜像时从包内载入，有时复用。

不执行sudo、不调用docker CLI/API、不加入docker组、不挂docker.sock、不改系统Docker配置、不增加privileged或root。镜像源仅包用户运行、仅绑定回环地址、只接受GET/HEAD、最多存续20分钟。服务容器继续非root与cap_drop全部。

按镜像digest派生本机端口，旧版与新版可短时并存。Open保留全部镜像blob的文件句柄，安装暂存目录被平台清理或移动时，正在进行的镜像读取仍然可完成；退出时释放句柄。init不向尚未授权的应用var目录写入，callback负责首次密码/数据准备。

该方案基于Docker官方Distribution V2协议的普通拉取路径。Docker文档说明127/8回环镜像源默认被识别为本地非TLS源，本轮在实际Docker Engine上验证；不擅自修改用户Docker daemon配置。

## 构建
先构建新版服务镜像与两种架构的cmd/offline-registry辅助程序。
`packaging/fnos/build.py --fnpack /path/to/fnpack --arch x86 --image songlib-tv-installer:0.1.2 --output output.fpk`
构建器docker save源镜像、核对OS/架构/每层diff_id，转换压缩blobs，锁定manifest digest，嵌入FPK，再使用官方fnpack build。ARM对应arch=arm与arm64源镜像。

用户只需要对应架构的一个FPK，不需要另下载镜像或运行docker load。包变大是因为包含了全部运行依赖。中文、8位密码、原音屿配色的256/512圆角图标保留。

## 验证边界
自动化覆盖只读API/错误路径/数据hash/暂存目录删除后读取、脚本无数据目录时的校验与密码规则、Go全套与lint。完整包验证须从最终FPK提取脚本和镜像，在缺少该manifest镜像引用的Docker环境中执行init→自动拉取→callback→启动，不能再先docker load后宣称安装通过。
未直接对用户NAS执行安装/部署；真实fnOS应用中心与Apple TV签装仍需设备复测。

参考：
- https://developer.fnnas.com/docs/core-concepts/framework/
- https://developer.fnnas.com/docs/core-concepts/privilege/
- https://developer.fnnas.com/docs/examples/docker/
- https://docs.docker.com/reference/cli/dockerd/#insecure-registries
