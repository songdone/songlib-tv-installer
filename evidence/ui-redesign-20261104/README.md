# UI 重构验收截图 · 2026-11-04

依据 `docs/UI_PRODUCT_REDESIGN_HANDOFF_20261104.md` 与 `docs/VISUAL_SYSTEM_AND_CODEX_HANDOFF_20261104.md` 完成的页面重构验收截图。

## 采集方式

- 本地运行真实 Go 服务（临时数据目录、测试密码），浏览器登录后截图。
- 「mock-connected / mock-cards」两张来自只读 mock API 服务（静态 dist + 假 `/api` 响应），
  用于演示已连接设备、安装中与续签失败状态；**mock 截图不代表真机验收**。
- 溢出检查：360/390/768/1024/1440 五种宽度 × 浅色/深色 × 4 个页面，
  `document.scrollWidth > innerWidth` 全部通过（40/40；修复记录见提交历史）。

## 文件清单

| 文件 | 内容 |
| --- | --- |
| workbench-light-1440.png | 工作台浅色：Hero（正式 Slogan 双行、官方 TV 截图轮播、价值点、分层 CTA）+ 三步 QuickStart + 安全须知 |
| workbench-dark-1440.png | 工作台深色：墨绿底金色点缀，主按钮浅底深字，未做成整片金色 |
| workbench-light-390.png | 移动端 390px：56px 紧凑顶栏（汉堡菜单+品牌简称）、单列、通栏主按钮 |
| workbench-mock-connected-1440.png | 已连接状态：QuickStart 折叠为摘要卡、真实设备卡（已连接/待配对状态色） |
| devices-light-1440.png | Apple TV 设备页：筛选、单一空态引导卡、手动 IP 入口 |
| apps-empty-light-1440.png | 应用与续签空态：一条可行动引导，无空白表格 |
| apps-mock-cards-1440.png | 应用与续签（mock）：状态摘要、行卡列表、警告 badge、到期时间 |
| explore-light-1440.png | 探索音屿：三章官方截图、支持的服务器、功能模块 |

## 仍需真机验收（未完成，不得写成已通过）

- 真实 fnOS 安装启动、真实 Apple TV 发现/配对、tvOS IPA 签装、7 天续签观察。
- iPhone/iPad 路径保持隐藏（`showIphoneTab=false`），开启前需端到端验证。
- 应用中心 FPK 展示、多机型 Safari/Chrome 兼容性抽样。
