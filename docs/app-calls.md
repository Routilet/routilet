# 本地应用调用协议 v1

v0.1.3 新增无界面入口：

```sh
routilet --app-call-probe
routilet --open-app observer
routilet --open-app observer --wait -- --session /absolute/path/session.json
routilet --query-app observer -- --capabilities
routilet --query-app observer -- --version
```

probe 输出 routilet-app-call/1。应用 ID 是 catalog 稳定 ID，不接受任意可执行路径或 routilet 自身。-- 后原样传递参数，不执行 shell。调用目标自行解析业务数据；调用者应传绝对文件路径。无 --wait 时仅确认启动；--wait 等待实际进程结束，传递退出状态（信号/管理错误归一为 1）。查询仅允许 --capabilities / --version，10 秒期限、stdout/stderr 各 1 MiB；成功 stdout 原样返回，不混入管理器日志，失败 stderr 和非零状态。查询不自动安装、不联网、不启动管理器 GUI。

所有调用使用激活记录、manifest v1、平台与 SHA256 校验；在跨进程锁内创建运行标记，再释放锁等待，安装/卸载/更新可并行，清理保留运行版本。每次调用独立选择激活版本，查询与之后启动之间可能发生更新，应用协议兼容由调用方确认。异常杀死监督进程时保留标记，不按时间自动清除；不能保护目标自行脱离的子进程。

Routilet 向原生应用注入 ROUTILET_EXECUTABLE（本次运行的支持协议的管理器绝对路径）、ROUTILET_BOOTSTRAP（初始入口路径）。**不是永久稳定入口**：已发行旧 bootstrap 无法识别新增参数，因此不把旧入口冒充新协议入口。当前 manager 快照不自动清理，支持调用期间使用；应用每次启动重新读取，不持久化。未来若实现管理器快照清理须保护调用者所持路径。旧 --open-app ID、原有快捷方式语法保留；旧 bootstrap 不会自动获得新 CLI 能力，旧入口创建的旧快捷方式行为不变；当前为开发期，正式分发使用新版初始入口。旧入口不提供新增调用协议，不做额外迁移。

macOS 监督进程改为直接执行经校验的 APP 内部原生入口，而不是 open -n -W，以确保参数、环境和真实退出状态；Wails 窗口激活、Dock 行为和退出需真机验证。Windows 查询隐藏控制台窗口。GUI 调用者在后台执行 --wait，只有明确退出才清理会话；Routilet 不读取、删除或解释业务文件。退出码非零不表示被观察程序运行失败，后者由 Observer UI 管理。

本阶段实现启动协议并启用 Companion catalog（新包 @routilet/companion，拒绝 0.0.0 占位版本），未管理 Exam Kit、未实现 OJ 插件。自动测试使用原生 fixture 验证参数、Unicode 路径、查询、退出状态、卸载期间保护、清理；现有 CI self-test 在安装实际 Observer 后查询其能力。真机跨应用 GUI 联调另行验收。

自 v0.1.4 起，禁用应用拒绝新的启动和查询请求，返回非零状态；已启动进程不被终止。此为用户偏好，不是安全沙箱或旧管理器的访问控制。

## 可选 Adapter 组件调用协议

Accoding Adapter（catalog ID `accoding-adapter`、包 `@routilet/accoding-adapter`）独立安装、更新、回退、禁用、卸载；不自动随 Companion 安装，不可直接打开，不提供快捷方式。当前 beta.1 只在测试通道可安装。沿用版本快照和清理保护。采用已发行 Adapter manifest v1 的 adapter/protocol/platforms.path，复用哈希校验，安装前另做离线 capabilities 握手。

```sh
routilet --adapter-call-probe
routilet --adapter-available accoding-adapter
routilet --adapter-stdio accoding-adapter
```

probe 返回 `routilet-adapter-call/1`；available 只读验证当前激活版本、启用状态、身份/协议、平台和 SHA256（退出 0 表示可启动，不联网，不启动子进程）。stdio 在操作锁内重新校验并放置运行标记，释放锁后原样转发 stdin/stdout 到固定 --stdio 子进程；不解析业务、不转发 Adapter stderr、不接受任意参数。正常 EOF/退出回收运行标记；异常杀死监督进程保守保留标记。调用者正常关闭 stdin、等待 supervisor 退出；超时再终止，可能留下保守标记。不要暴露安装二进制路径或绕过 manager。旧 manager 不支持时仅禁用在线功能。Companion 对接需另行实现，仍可保留显式开发路径作为测试覆盖。

首页将可选组件与独立应用分区显示，当前固定展开；安装组件不自动启动或退出 Routilet。业务 stdio 和 stderr 不持久化、不展示，避免账号、请求和响应数据泄露；启动失败、进程退出状态和离线握手错误提供结构化阶段提示。
