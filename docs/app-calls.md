# 本地应用调用协议 v1

新增无界面入口（拟随后续 Routilet 发行，不改变当前 VERSION）：

```sh
routilet --app-call-probe
routilet --open-app observer
routilet --open-app observer --wait -- --session /absolute/path/session.json
routilet --query-app observer -- --capabilities
routilet --query-app observer -- --version
```

probe 输出 routilet-app-call/1。应用 ID 是 catalog 稳定 ID，不接受任意可执行路径或 routilet 自身。-- 后原样传递参数，不执行 shell。调用目标自行解析业务数据；调用者应传绝对文件路径。无 --wait 时仅确认启动；--wait 等待实际进程结束，传递退出状态（信号/管理错误归一为 1）。查询仅允许 --capabilities / --version，10 秒期限、stdout/stderr 各 1 MiB；成功 stdout 原样返回，不混入管理器日志，失败 stderr 和非零状态。查询不自动安装、不联网、不启动管理器 GUI。

所有调用使用激活记录、manifest v1、平台与 SHA256 校验；在跨进程锁内创建运行标记，再释放锁等待，安装/卸载/更新可并行，清理保留运行版本。每次调用独立选择激活版本，查询与之后启动之间可能发生更新，应用协议兼容由调用方确认。异常杀死监督进程时保留标记，不按时间自动清除；不能保护目标自行脱离的子进程。

Routilet 向原生应用注入 ROUTILET_EXECUTABLE（本次运行的支持协议的管理器绝对路径）、ROUTILET_BOOTSTRAP（初始入口路径）。**不是永久稳定入口**：已发行旧 bootstrap 无法识别新增参数，因此不把旧入口冒充新协议入口。当前 manager 快照不自动清理，支持调用期间使用；应用每次启动重新读取，不持久化。未来若实现管理器快照清理须保护调用者所持路径。旧 --open-app ID、原有快捷方式语法保留；旧 bootstrap 不会自动获得新 CLI 能力，旧入口创建的旧快捷方式行为不变，新应用启用前另行验证。

macOS 监督进程改为直接执行经校验的 APP 内部原生入口，而不是 open -n -W，以确保参数、环境和真实退出状态；Wails 窗口激活、Dock 行为和退出需真机验证。Windows 查询隐藏控制台窗口。GUI 调用者在后台执行 --wait，只有明确退出才清理会话；Routilet 不读取、删除或解释业务文件。退出码非零不表示被观察程序运行失败，后者由 Observer UI 管理。

本阶段只实现启动协议，未启用 Companion catalog 项（占位包不安装）、未管理 Exam Kit、未实现 OJ 插件。自动测试使用原生 fixture 验证参数、Unicode 路径、查询、退出状态、卸载期间保护、清理；现有 CI self-test 在安装实际 Observer 后查询其能力。真机跨应用 GUI 联调另行验收。
