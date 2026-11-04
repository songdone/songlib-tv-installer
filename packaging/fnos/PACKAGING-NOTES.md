# fnOS（飞牛）FPK 打包要点

维护本目录（`packaging/fnos/songlibtvinstaller/`）前先读一遍。这里每一条都是真机踩过坑换来的，与 fnpack 官方模板不一致的地方以本文件为准——官方模板是最简示例，不是最佳实践。

## 图标（易错：照抄官方模板会糊）

fnpack 官方模板的 `ICON.PNG` 是 **64×64**，而飞牛应用**详情页会放大读取根目录 `ICON.PNG`**——照抄官方尺寸必然模糊。

本应用的规格（构建脚本 `build.py` 会从圆角母版自动重建全套，不要手工散放图片）：

| 文件 | 尺寸 | 用途 |
| --- | --- | --- |
| 根目录 `ICON.PNG` | **512×512** | 应用详情页（放大读取的就是它） |
| 根目录 `ICON_256.PNG` | **256×256** | 应用中心网格 |
| `app/ui/images/icon_64.png` | 64×64 | 桌面快捷方式（`ui/config` 里 `images/icon_{0}.png` 按尺寸取用） |
| `app/ui/images/icon_256.png` | 256×256 | 同上 |
| `app/ui/images/icon_512.png` | 512×512 | 同上 |

注意：

- `fnpack build` 只会把根目录 `ICON.PNG` 和 `ICON_256.PNG` 收进包，其他根目录图标写了也白写。
- 圆角母版是 `ICON_512.PNG`（512×512，已抠圆角）。`branding/ios-app-icon-original.png` 是 1024 方角原图且画风不同，不要混用。
- 验包必须实际解包读像素尺寸（fpk 是两层 tar：外层 → `app.tgz` → `ui/`），不能只看文件存在。

## 管理端口（易错：装机脚本里渲染桌面入口无效）

- 桌面入口端口写在 `app/ui/config` 的 `"port"` 字段，值必须是**字面占位符 `${wizard_port}`**（与 `wizard/install` 里的字段名一致）。飞牛在注册桌面入口时自行替换；在 `cmd/prepare` 里 sed 渲染是**无效的**——那时入口早已注册，还会破坏占位符导致后续改动失效。
- 容器端口映射在 `app/docker/docker-compose.yaml.template`（`# songlib:port` 标记行），由 `cmd/prepare` 用 awk 按向导值渲染。不要指望平台把向导变量传给 compose：真机实测「修改配置」后 compose 收到的 `wizard_*` 可能仍是旧值，以 prepare 渲染为准。
- 端口持久化在 `$TRIM_PKGVAR/port`，`cmd/main status` 按它做健康检查（并兜底解析 compose 现值）。
- 改端口必须重建容器才生效：`cmd/config_callback` 在端口变化时重渲染并 `docker compose up -d --force-recreate`。
- 「域名+端口」是飞牛第三方应用桌面入口的通用打开方式。不带端口的域名访问（路径或子域名路由）由用户自己的反向代理/隧道解决，FPK 不负责。

## 其他

- `cmd/*` 权限必须是 0755（跨平台打包可能丢失执行位，发布前 `tar -tvzf` 检查）。
- `manifest.desc` 支持 HTML（`<br>`/`<b>`/`<a>`），应用介绍完整文案随包发布。
- `wizard/config` 定义「应用设置」页的可改项（当前为管理端口，留空表示不修改）。
- 改动打包内容后**版本号必须递增**（`manifest.version`），否则飞牛不允许覆盖升级。
