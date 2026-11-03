# FrpDeck → 音屿安装助手 组件映射表（P0 产出）

> 2026-11-04。依据 `docs/FRPDECK_GLOBAL_UI_SYSTEM_HANDOFF_20261104.md` §1 要求产出。
> FrpDeck 源码：`/Users/playsong/Mirasim/frp-switch/frpdeck/web/assets/`（css/base|ui|pages|minimal.css、tpl/shell|overview|license.js、js/motion.js）。
> 结论：**布局方法、栅格、控件风格、留白、动效、交互规律全部继承；配色、图形、内容全部换为音屿品牌。**

## 1. 设计 Token 映射（FrpDeck 变量 → 音屿 --sl-*）

| FrpDeck | 音屿 Token | 说明 |
| --- | --- | --- |
| `--bg` / `--bg-2` | `--sl-bg` / `--sl-bg-soft` | 浅色暖白纸感 #FAF8F4；深色取 Logo 墨底 #0D0F11 |
| `--surface`（rgba .94/.74 微透明） | `--sl-surface` | 卡片微透明 + backdrop-blur，同样双层 |
| `--surface-solid` | `--sl-surface-raised` | 弹窗/菜单实底 |
| `--surface-2` / `--hover` | `--sl-surface-2` / `--sl-hover` | 次级底与 hover |
| `--line` / `--line-2` | `--sl-border` / `--sl-border-strong` | 精细内描边两级 |
| `--text` / `--muted` / `--faint` | `--sl-text` / `--sl-muted` / `--sl-subtle` | 文字三级（暖灰） |
| `--pri` / `--pri-2` / `--pri-3` / `--on-pri` | `--sl-accent` / `--sl-accent-strong` / `--sl-accent-soft`(亮金) / `--sl-accent-on` | **从 Logo 采样**：亮金 #E3B459、深金铜 #9A7420（浅色 AA）；不复制 FrpDeck 金 #E6BE7A |
| `--btn`（渐变）+ `--btn-shadow` | `--sl-btn` / `--sl-btn-shadow` | 主按钮金色渐变，音屿采样值 |
| `--heading`（渐变大标题） | `--sl-heading` | h1 渐变文字，暖墨色系 |
| `--good/warn/crit/off/sec` | `--sl-good/warning/danger/info/muted-strong` | 语义色，深浅色两套 |
| `--glow1/2/3` | `--sl-glow-1/2/3` | 背景三团光晕：金 + 苔绿 + 暖橙（音乐氛围） |
| `--shadow` / `--topbar` | `--sl-shadow` / `--sl-topbar` | 大柔和投影 + 顶栏毛玻璃底 |
| `--r-xs/sm/r/lg/xl`（8/10/14/18/22） | `--sl-radius-*` | 同梯度 |
| `--ease`/`--spring`/`--sp-*` | `--sl-motion-*` | 同缓动族（弹簧 linear() 采样可选） |
| `--top-h` 64px | `--sl-top-h` | 顶栏 64px；平板两行 112px；手机 bnav 底栏 |

## 2. 布局 / Shell 映射（tpl/shell.js → layout.vue）

| FrpDeck | 音屿实现 | 备注 |
| --- | --- | --- |
| `.topbar` + `.topbar-in`（1480 版心） | `layout.vue` 顶栏（毛玻璃、内描边） | 同结构：品牌区 + 居中胶囊导航 + 右侧图标区 |
| `.brand` + `.ver` 版本小徽标 | 音屿 Logo(PNG) + 应用名（窄屏简称） | 用 songlib-mark.png，非 FrpDeck lockup |
| `.nav` 胶囊导航 + 弹簧 slider | 同款胶囊导航；active 用内衬底色胶囊（不移植 JS slider，改纯 CSS，效果一致且更稳） | 6 项：总览/Apple TV/应用与续签/签名账户/音屿播放器/购买与授权 |
| 平板 861–1320 两行顶栏（112px） | 同断点同布局 | css 网格实现 |
| `.top-actions` icon-btn 组 | 语言/主题/关于/GitHub + 设置齿轮 | FrpDeck 的安全/退出按钮不适用（管理密码在 Go 侧） |
| `.pop` 弹出菜单 + `.pop-i` + `.swd` | 主题/语言菜单改用此样式（弃 daisyUI dropdown） | 行高、圆角、动画一致 |
| 手机 `.bnav` 底部导航 | `BottomNav`：总览/设备/应用/音屿 + 「更多」弹层（购买/账户/设置/关于） | safe-area、blur、on=accent |
| `.bgfx` 三团漂移光晕 | 同结构光晕（音屿金色系），工作页减淡 | reduced-motion 全停 |
| `.toasts` | 保留 vue3-toastify，但用 CSS 对齐 .toast 视觉 | 不引入 FrpDeck JS |
| `.auth`/`.auth-card`（登录卡+演示区） | Go 模板登录页重绘：居中 auth-card（Logo、ifield 图标输入、btn pri block、form-err），不做演示区 | 管理 8–64 位密码逻辑不变 |

## 3. 组件映射（ui.css/pages.css → components.css + Vue 组件）

| FrpDeck 组件 | 音屿组件/类 | 采纳点 |
| --- | --- | --- |
| `.page-head`（eyebrow/h1 渐变/desc/head-acts） | `.sl-page-head` `.sl-eyebrow` `.sl-h1` `.sl-desc` `.sl-head-acts` | 每页统一页头 |
| `.card`（微透明+blur+内描边+glowy） | `.sl-card`（+ `.glowy` 鼠标柔光） | 全站卡片 |
| `.card-head/.card-title/.card-sub/.ico-t` | 同名 `.sl-*` | 卡头图标砖 |
| `.btn`（胶囊 38px；pri 渐变+shine；ghost/danger/sm/xs/block） | `.sl-btn` 及变体 | 按钮体系（主按钮每区最多一个） |
| `.field/.ifield/hint/.grid2/.sec-t` | `.sl-field` `.sl-ifield` `.sl-hint` `.sl-grid2` `.sl-sec-t` | 表单 |
| `.seg` 分段选择 | `.sl-seg` | 设备页筛选 |
| `.pill/.chip/.tag/.kbd` | `.sl-pill`（dot 动画）/`.sl-chip`/`.sl-tag` | 状态三重编码（icon+文字+色） |
| `.stats/.stat/.stack` | `.sl-stats` `.sl-stat` | 应用页状态摘要 |
| `.banner` info/crit/pro | `.sl-banner` | 续签风险、离线提示 |
| `table.t`（sticky th、tr.bad 左红条） | `.sl-table` | 应用 >8 条时表格 |
| `.modal/.drawer`（移动端底部 sheet） | `.sl-modal`（关于弹窗迁移） | 圆角 20、pop/sheet 动画 |
| `.empty` + `.art` | `.sl-empty` | 设备/应用空态 |
| `details.faq` | `.sl-faq` | 购买页 FAQ |
| `.steps`（编号圆圈） | `.sl-steps` | 购买/激活三步、QuickStart |
| `.lic-plans/.lic-plan/.lic-plan-art/.lic-price/.lic-feats` | 购买页 `PlanCard`：左图右文（200px 官方截图）、金色 hover 边 | 价格不写死，CTA 门控 |
| `table.cmp`（三列对比、gold 列） | `PlanComparison` | 基础版/Pro 功能对比（官网已验证功能） |
| `.kv/.bar-track/.data-tip/.code` | 同类 | 按需 |
| `.page > *` 依次 fadeUp / `.card` cardIn | `tokens.css` 同名动画 | reduced-motion 全停 |
| `motion.js` 弹簧 | 仅 CSS 缓动近似（linear() 采样原样搬） | 不引入其 JS |

## 4. 页面映射（§4 表格落地）

| 音屿页面 | 路由 | FrpDeck 参照 | 状态 |
| --- | --- | --- | --- |
| 登录 | /management/login（Go 模板） | auth 卡 | 重绘模板，逻辑不动 |
| 总览 | /#/home | overview.js page-head+hero+分组卡 | hero=系统状态卡 + 音屿宣传卡 + 设备/续签摘要 |
| Apple TV | /#/devices | 设备列表卡+筛选 | seg 筛选、空态、手动 IP |
| 应用与续签 | /#/apps | 服务列表+状态 | stats 摘要、行卡/表格、banner |
| 签名账户 | /#/account | 表单/安全页 | 旧页经 token 映射自动换肤 + 页头补齐 |
| 音屿播放器 | /#/explore | showcase 轻模块 | 三章官方截图 |
| **购买与授权** | /#/purchase（新增） | license.js | 套餐卡+对比+步骤+FAQ；**无机器码、无价格硬编码**，CTA 门控 |
| 设置 | /#/settings | system 帮助页 | 页头统一；高级工具入口保留 |

## 5. 旧样式清理清单

1. 删除侧栏布局（layout.css 的 .sl-sidebar/.sl-navitem 体系）→ 换 topbar/bnav。
2. `app.css`：`--atv-*` 全量保留但仅作为映射层指向 `--sl-*`；删除与 `.sl-*` 冲突的旧按钮/卡片覆盖（如 `.app-container .btn`、`.modal-box` 定制随迁移移除）。
3. daisyUI 仅作为旧页面（install/pair/account/settings/tools）过渡依赖；`--p/--pf/--pc` 指向新 accent，主色统一为音屿金铜色。
4. 移除上一版遗留的 `.sl-hero-ctas` 大按钮 / `.atv-onboard` 系样式（QuickStart 重写为 `.sl-steps` 卡片）。
5. 不新增任何 `!important`。

## 6. 业务红线复核（映射时已排除的 FrpDeck 内容）

- 不移植：机器码/Docker 绑定、授权码激活输入框、¥8.8/¥18 价格、NAS 产品图、线路/隧道指标、FrpDeck logo 与字体文件（Inter woff2 不复制，用系统字体栈）。
- 购买页价格与 CTA 只来自 `/songlib/config`（verified 16688 链接）；未配置时展示功能对比与购买说明，不渲染假按钮。
- 「30 天 Pro 试用」「6 台跨端」等宣称在私有后端上线前不出现；FAQ 单独解释 7 天自签与商业授权两条时间轴。
