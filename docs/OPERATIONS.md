# 当前安装方式：0.1.3 DockerHub通用版

只安装一个SongLib-TV-Installer-0.1.3.fpk，支持x86/ARM；镜像由DockerHub自动选择架构，无需手动导入。
发布者6uos；镜像666uos/songlib-tv-installer:0.1.3。包内不放镜像、临时镜像源或播放器IPA。

构建：先构建服务两架构镜像并发布多架构index，验证匿名可拉取；再执行packaging/fnos/build.py --fnpack /path/to/fnpack --image 666uos/songlib-tv-installer:0.1.3 --output output.fpk。
检查：packaging/fnos/check.py与test_lifecycle.py；FPK必须小于2MiB，manifest platform=all，维护者和发布者均为6uos。

数据：TRIM_PKGVAR/data保存密码哈希、配对、账户和应用数据。升级前备份完整var，升级保留原密码和端口。卸载前自行导出备份。
诊断：应用中心运行状态、Docker容器状态/日志、5533端口映射、/ready返回200后再检查页面。不要仅用打包或本地构建替代用户设备验收。

0.1.2离线方案已停用，历史文件仅作回溯，不继续交付。
