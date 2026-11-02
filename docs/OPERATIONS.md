# 开发与运维

## 构建
1. Go1.25+，Node，npm ci（web/static），npm test，npm run build。
2. CGO_ENABLED=0 GOOS=linux GOARCH=amd64/arm64 go build -trimpath -o build/atvloadly-linux-<arch> .
3. git archive HEAD | gzip > build/songlib-source.tar.gz。不得混入商业项目。
4. docker build --platform linux/<arch> -f packaging/Dockerfile -t songlib-tv-installer:0.1.1 .（跨架构须对应参数）。
5. python3 packaging/fnos/check.py；python3 packaging/fnos/build.py --fnpack /path/to/fnpack --arch x86 --image songlib-tv-installer:0.1.1 --output /path/to/SongLib-TV-Installer-0.1.1.fpk。

## 本地验收包安装
先docker load对应架构镜像，记录image ID。用fnOS手动安装FPK，向导选择空闲端口、8至64位管理密码。
本地包不从不存在的公开registry拉取，不能直接当作应用中心版本。
数据目录 `/var/apps/songlibtvinstaller/var/data`，运行配置在私有目录内的密码哈希与应用配置。此目录不可分享给普通NAS用户。

## 配对与续签
见 APPLE_ID_GUIDE.md。首页同时显示产品卡、原上游设备与应用状态。使用独立Apple小号。导入合法IPA后手动确认签装。P12+描述文件、手动IP仍沿用上游。

## 健康/诊断
`/ready`只表示进程响应，与某IPA是否到期分开。`/healthcheck`需管理登录，保留上游应用过期含义。不要用503直接循环重启容器。
配对失败先查同LAN/mDNS/socket；签名失败查Apple配额、2FA、账户风控和tvOS描述文件；NAS本地没有PC下载的IPA，需手动上传。

## 备份与升级
先停任务和应用服务，再备份完整var（包括data/keys/PlumeImpactor/config；历史runtime.env若存在也一并备份），校验SHA256。升级不删除数据并沿用原端口、UID、密码hash；停机时会话失效后重新登录。
恢复必须成套恢复数据+密钥+配置，并先在隔离目录演练；丢失私钥无法凭IPA找回。卸载前导出备份，fnOS可能删除数据。

## 发布边界
没有生产部署/16688交易/应用中心提交权限。只有本地构建与浏览器验证；fnOS+AppleTV与7天续签未验收。真实设备验证完成前，包名及manifest保持开发验收说明。
