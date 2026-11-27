# Routilet 桌面启动台

启动台以应用展示与打开为主：紧凑窗口、应用列表，点击已安装应用直接启动，未安装且存在兼容版本时点击直接安装并打开，有更新才展示按钮。启动请求成功后退出启动台；启动失败保留窗口。更新不自动打开应用。应用独立进程；没有常驻服务。

Go + Wails v3.1.0，静态 Web 前端，固定 pnpm 12.22.0 随包分发；使用 npm 官方源，默认链接/导入方式，共享 store，禁用安装脚本。不使用 Node、镜像或 pnpm 版本管理。首次下载包含 pnpm。

## 使用与边界

- 从 GitHub Release 下载对应平台归档并完整解压，运行 Routilet.exe / Routilet.app。macOS ad-hoc 签名、未公证；可信来源按系统单应用授权。Windows 使用系统 WebView2。
- Companion 目前只展示“新版开发中”，不安装旧 TS/TUI。OJ 插件及 Exam Kit 不在首页；不做插件协议。
- Observer 接入现有 @routilet/c-observer 的兼容 GUI 版本（至少 0.2.0 系列），不降级到旧 TUI。默认正式通道；测试通道比较 latest 与 next。若尚未发行、网络失败或清单不匹配，提示具体原因。读取现有 manifest.json 的平台入口与 SHA-256；不要求修改 Observer。
- 安装数据隔离于 OS UserConfigDir/Routilet（macOS ~/Library/Application Support/Routilet；Windows %APPDATA%\\Routilet），当前版本通过每应用 activation.json 记录；更新新建快照、校验后激活。不接管已有 Launcher、账号或课程。保留当前、上一版以及有运行标记的快照；跨进程文件锁串行化操作。应用监督进程退出后清除标记，异常终止时保留标记，避免误删。未验证应用自身派生并脱离的子进程。应用自身仍可能读取其正常用户配置。
- 首页先读本地状态，再后台查版本；断网仍可打开已安装的应用。不检查 pnpm 新版本。应用更多菜单提供修复、卸载、回退、快捷方式（Windows .lnk / macOS .command，原始入口移动后需重建）。底部独立入口检查并显示自身最新版本；固定 pnpm 将新 Manager 安装至独立快照，校验协议，下一次打开由原入口启动，前端 Ready 超时或提前退出则回退。当前保留全部 Manager 快照，不清理可能运行中的版本。
- CI 测试 Go 单测、真实 esbuild 原生包安装；若 Observer next 已发布，再验证其实际包清单和哈希。CI 不启动 GUI，不等于真机验收；macOS open 成功代表系统接收启动请求，并非应用已完成初始化。Windows 同理仅确认进程创建。若真机发现启动异常，再完善握手机制。

旧 Routilet Preview 目录保留原状，不自动迁移或删除；可在关闭所有相关应用后手动清理。首页底部“数据目录”打开新目录。安装及更新使用 pnpm 实际进度输出，未知总量显示不定进度；取消在激活前有效，关闭窗口/退出在任务进行时阻止，请先取消或等待完成。取消后清除候选目录，旧版本不变；操作系统强制终止不保证清除残留候选。

## 真机验收

1. 双击，观察首页、系统深浅色、紧凑尺寸。
2. Companion 显示开发中，Observer 若无兼容发行版显示暂未发布；两者均不能触发无效安装。
3. Observer next 可用时安装并启动，确认 Routilet 退出、Observer 独立运行；再次打开 Routilet 直接启动。
4. 有新版时确认更新按钮出现；更新成功留在启动台，旧安装保留；断网仍可打开旧版。
5. 安装时检查真实进度、取消、关闭窗口阻止、旧版本保留及数据目录入口。
6. 检查 Windows/macOS 权限、执行方式、中文路径及程序退出。没有真实新版 Observer 时，第 3–4 项暂不可验收。

历史实验已结束；实际应用接入和线上升级仍需真机验收。

## 自更新发行协议

固定初始入口兼作备用 GUI。@routilet/routilet latest 包包含双平台预构建 Manager（manifest schema 1）；不包含 pnpm，使用初始入口同目录的固定 pnpm。`scripts/package-manager.py` 从已构建原生资产生成 npm tarball，不发布。macOS 候选 APP 需在打包前单独签名。CI 使用独立高版本测试 tarball，验证实际 pnpm 安装、清单和协议、激活记录、真实子进程 Ready 握手（无 WebView 的测试候选）及损坏后回退；用户已确认 v0.1.0 → v0.1.1 真实自更新通过；本版交互及后续升级仍需真机。正式 CI 生成同版本双平台 Manager npm 包，不含 pnpm，真实 npm 首发已完成；每次发行使用 SHA256 校验归档。

### 生命周期限制

保留由本启动台打开的应用的运行标记。异常退出的标记不按时间自动失效；因此可能积累旧快照，只有确认没有运行进程后才可人工删除标记及清理。直接绕过入口启动底层快照或应用自行脱离的子进程不在当前保护契约内。macOS 快捷方式为 .command（会短暂打开终端），Windows 为 .lnk；不覆盖已有快捷方式。初始入口必须保留原位置。自更新候选不清理历史版本，避免删除运行中的 Manager。真实 GUI 启动握手、运行监督及快捷方式需双平台实机确认。

### 发行通道

底部设置入口提供全局正式/测试通道，适用于 Routilet 自身和各独立应用，设置保存在数据根 release-channel.json。默认正式通道仅接受兼容的 latest 正式版；测试通道查询 latest 和 next，按版本优先级选择更高的兼容版本，平级优先 latest。next 缺失时使用 latest；latest 查询失败时不猜测 next 是否更新，保留现有安装。切换通道只改变后续检查和安装候选，不自动降级已安装版本，不撤销已准备好的自更新；回退仍是显式操作。如果已准备的 Manager 版本低于新发现的正式版，优先显示更新入口。旧 Observer TUI 不属于兼容候选。pnpm 不受通道影响。

## 图标与本地构建

设计源稿 `frontend/icons/*.svg` 与 [组织图标说明](https://github.com/Routilet/.github/blob/main/docs/icon-design.md) 同源。CI 使用固定版本 resvg 渲染 SVG，Windows 用 rsrc 嵌入 ICO，macOS 用 iconutil 生成 ICNS（签名前复制到 Resources）。生成的 PNG 也供 Wails 设置运行时图标；不需要用户安装 Node。开发时使用 Go 1.26，提交 go.mod/go.sum，`go test -mod=readonly ./...`。原生图标生成步骤见发布工作流。初始入口不会被自更新覆盖：v0.1.0 原始文件的 Finder/Explorer 图标不变；使用 v0.1.1 下载包才有新的初始入口图标。

## v0.1.2 交互验收

前端仍为原生 JS：state.js 为纯状态模型，app.js 负责协调，dialogs.js 和 menu.js 分别处理确认及菜单。运行 `node --test tests/interaction.test.mjs` 验证关键状态规则。CI 另用测试专用 Chromium 和模拟 Wails 桥执行浏览器交互测试，不将 Chromium 打入应用，也不等同于真实系统 WebView 交互。验收矩阵：

- 未发布/检查中/检查失败不可安装；已安装断网仍可打开；更新检查失败不显示“最新”。
- 点击安装/更新直接开始；取消在激活前有效；安装成功但打开失败时保留已安装状态。
- 应用菜单键盘方向键、Esc、点击外部关闭；回退目标显示具体版本，卸载明确确认。
- 设置内切换通道自动保存且不关闭；并行旧检查不覆盖新通道；清理结果显示实际移除、保护及失败。
- 自更新共享进度及取消；待重开状态独立于联网检查，按钮明确退出并应用更新（用户需重新打开）。
- 最小尺寸、深浅色、键盘焦点以及两平台原生显示须实测。
