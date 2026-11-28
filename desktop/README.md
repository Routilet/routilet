# Routilet 桌面启动台

启动台以应用展示与打开为主：紧凑窗口、应用列表，点击已安装应用直接启动，未安装且存在兼容版本时点击直接安装并打开，有更新才展示按钮。启动请求成功后退出启动台；启动失败保留窗口。更新不自动打开应用。应用独立进程；没有常驻服务。

Go + Wails v3.1.0，静态 Web 前端，固定 pnpm 12.22.0 随包分发；使用 npm 官方源，默认链接/导入方式，共享 store，禁用安装脚本。不使用 Node、镜像或 pnpm 版本管理。首次下载包含 pnpm。

## 使用与边界

- 从 GitHub Release 下载对应平台 ZIP 并完整解压。Windows 打开 Routilet 文件夹并双击 Routilet.exe，不单独移动 EXE；macOS 解压后直接使用 Routilet.app，可移入 Applications。macOS ad-hoc 签名、未公证；可信来源按系统单应用授权。Windows 使用系统 WebView2。
- Companion 使用新包 `@routilet/companion`；占位版本 0.0.0 不可安装，当前通道无可用发行时显示暂未发布；不安装旧 TS/TUI。OJ 插件及 Exam Kit 不在首页；不做插件协议。
- Observer 接入现有 @routilet/c-observer 的兼容 GUI 版本（至少 0.2.0 系列），不降级到旧 TUI。默认正式通道；测试通道比较 latest 与 next。若尚未发行、网络失败或清单不匹配，提示具体原因。读取现有 manifest.json 的平台入口与 SHA-256；不要求修改 Observer。
- 安装数据隔离于 OS UserConfigDir/Routilet（macOS ~/Library/Application Support/Routilet；Windows %APPDATA%\\Routilet），当前版本通过每应用 activation.json 记录；更新新建快照、校验后激活。不接管已有 Launcher、账号或课程。保留当前、上一版以及有运行标记的快照；跨进程文件锁串行化操作。应用监督进程退出后清除标记，异常终止时保留标记，避免误删。未验证应用自身派生并脱离的子进程。应用自身仍可能读取其正常用户配置。
- 首页先读本地状态，再后台查版本；断网仍可打开已安装的应用。不检查 pnpm 新版本。应用更多菜单提供修复、卸载、回退、快捷方式（Windows .lnk / macOS .command，原始入口移动后需重建）。底部独立入口检查并显示自身最新版本；固定 pnpm 将新 Manager 安装至独立快照，校验协议，下一次打开由原入口启动，前端 Ready 超时或提前退出则回退。当前保留全部 Manager 快照，不清理可能运行中的版本。
- CI 测试 Go 单测、真实 esbuild 原生包安装；若 Observer next 已发布，再验证其实际包清单和哈希。CI 不启动 GUI，不等于真机验收；两平台直接执行经校验的原生入口；仅确认进程创建，不等于应用已完成初始化。若真机发现启动异常，再完善握手机制。

旧 Routilet Preview 目录保留原状，不自动迁移或删除；可在关闭所有相关应用后手动清理。设置中的“打开数据目录”打开新目录。安装及更新使用 pnpm 实际进度输出，未知总量显示不定进度；取消在激活前有效，关闭窗口/退出在任务进行时阻止，请先取消或等待完成。取消后清除候选目录，旧版本不变；操作系统强制终止不保证清除残留候选。

## 真机验收

1. 双击，观察首页、系统深浅色、紧凑尺寸。
2. Companion、Observer 若无兼容发行版显示暂未发布，不能触发无效安装；正式发行可用后分别测试安装及打开。
3. Observer next 可用时安装并启动，确认 Routilet 退出、Observer 独立运行；再次打开 Routilet 直接启动。
4. 有新版时确认更新按钮出现；更新成功留在启动台，旧安装保留；断网仍可打开旧版。
5. 安装时检查真实进度、取消、关闭窗口阻止、旧版本保留及数据目录入口。
6. 检查 Windows/macOS 权限、执行方式、中文路径及程序退出。没有真实新版 Observer 时，第 3–4 项暂不可验收。

历史实验已结束；实际应用接入和线上升级仍需真机验收。

## 自更新发行协议

固定初始入口兼作备用 GUI。@routilet/routilet latest 包包含双平台预构建 Manager（manifest schema 1）；不包含 pnpm，使用初始入口附带的固定 pnpm（Windows runtime/pnpm.exe，兼容旧同级 pnpm.exe；macOS Contents/MacOS/pnpm）。`scripts/package-manager.py` 从已构建原生资产生成 npm tarball，不发布。macOS 候选 APP 需在打包前单独签名。CI 使用独立高版本测试 tarball，验证实际 pnpm 安装、清单和协议、激活记录、真实子进程 Ready 握手（无 WebView 的测试候选）及损坏后回退；用户已确认 v0.1.0 → v0.1.1 真实自更新通过；本版交互及后续升级仍需真机。正式 CI 生成同版本双平台 Manager npm 包，不含 pnpm，真实 npm 首发已完成；每次发行使用 SHA256 校验归档。

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

## v0.1.3 跨应用调用

详见 [本地调用协议 v1](../docs/app-calls.md)。Companion 使用新包 `@routilet/companion`，通过标准原生 manifest 安装；实际发行候选存在时 CI 安装并校验。macOS 直接启动 APP 内入口，窗口激活、Dock 和应用间真实联调须实机验证。旧初始入口无需迁移，正式分发请使用新版初始入口。

## v0.1.4 发行与应用偏好

Windows ZIP 内只有顶层 Routilet/，根目录包含 Routilet.exe、使用说明.txt 和 runtime/；pnpm 与许可证集中到 runtime/。macOS ZIP 使用 ditto 保留 APP 结构、权限和签名；只有 Routilet.app 是用户入口。npm Manager 包仍不包含 pnpm。新旧 Windows 入口均可定位各自附带的 pnpm。CI 从最终归档解压到含中文、空格的深层路径，使用实际发布二进制运行安装/自更新测试，更新候选额外验证原始入口的 pnpm 定位。CI 不等于 Windows 实机安装问题已复现。

应用菜单提供禁用/启用（包括尚未安装的应用）。每个应用的 disabled.json 与 activation.json 分离，禁用后不检查远程版本、不安装/更新/修复/回退、不接受经新版 Routilet 的新启动、能力查询和跨应用调用。保留安装快照、课程数据及正在运行的实例，仍可卸载；卸载保留禁用偏好。启用后重新检查。Routilet 自身不可禁用；旧管理器、绕过 Routilet 直接执行底层文件不受这一用户偏好约束，这不是安全访问控制。数据目录入口仅在设置中。

安装失败时查看错误弹窗技术详情：包含准备/下载安装/校验/激活等阶段，以及 pnpm、工作目录、包目录、入口等相关路径。用户实际报告的 Windows 路径错误尚无完整日志，未确定根因；请用新版复验并提供技术详情。

## Windows 原生安装布局（v0.1.5）

Windows 新快照通过 pnpm-workspace.yaml 固定 hoisted/copy、store 完整性验证和禁用全局 virtual store；使用数据根 store-native，保留原 store 和已激活旧快照。macOS 不改。原生清单与 SHA256 仍为激活条件。缺失入口额外记录 Stat 及有限父目录内容。CI 实际安装 Observer 并检查普通物理目录和缓存重复修复；用户机器仍需复验。不要为了缺失文件关闭安全软件。

## v0.1.6 可选组件

首页“可选组件”固定展开，与独立应用分区。Accoding Adapter 采用 `@routilet/accoding-adapter`，通过现有全局发行通道独立管理，安装后不打开或退出启动台。详见 [调用协议](../docs/app-calls.md)。用户已确认 v0.1.5 Windows 实机安装正常；本版尚需 Companion 联调与实际在线评测验收。

## v0.2.0-beta.2 下载加速

设置中的“下载源”独立于更新通道，默认自动选择（npmmirror 优先，可退回官方源），也可固定使用 npm 官方源。版本发现始终使用 npm 官方源的 latest/next；精确版本的官方 SHA-512 完整性信息是镜像校验基准。只有官方元数据表明无运行依赖的独立软件包才尝试镜像，镜像须提供相同身份、版本与 SHA-512，实际压缩包须经过重新计算 SHA-512 验证。仅允许 HTTPS 的预定镜像主机和重定向；未同步、哈希不一致、下载超时等均回退原官方 pnpm 安装路径；用户取消不继续回退。镜像元数据检查最多 3 秒，下载最多 2 分钟、512 MiB。设置损坏时使用官方源。

已校验的压缩包临时存于新快照，通过相同 pnpm 和原官方 registry 安装，安装后清除临时压缩包；不改变已有激活、回退、原生入口验证及安装脚本禁用逻辑。镜像下载的进度来自实际字节数（已知 Content-Length 时），未知大小时只显示阶段。镜像路径可能重新下载已存在于 pnpm store 中的压缩包，不提供自动测速、多个镜像竞速或按平台拆包。本版需在国内真实网络验证同步延迟和加速效果。CI 单测模拟镜像和错误响应，原生自更新测试验证实际 pnpm 的相对本地压缩包安装与删除后运行。
