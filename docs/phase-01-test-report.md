# 阶段 01 测试报告

状态：双平台原生核心测试、回归测试、打包及产物上传均通过；维护者确认 Mac 真机验收成功，Windows 真机验收待完成。

| 检查 | 自动化证据 | Windows 11 amd64 实机 | macOS arm64 实机 |
| --- | --- | --- | --- |
| 独立构建、TUI 启动 | CI 构建；菜单测试及假包 startV2 | 待验证 | 待验证 |
| Node 发现、安装和校验 | TestManagedRuntimeFromChinesePath | 待验证 | 待验证 |
| npm 安装及依赖解析 | 本地 registry 集成、npm probe | 待验证 | 待验证 |
| 候选差异及确认 | TestTransactionAndOfflineRollback、TestV2SettingsAndMenu | 待验证 | 待验证 |
| 失败保留当前安装 | unit / 本地 registry 不兼容版本 | 待验证 | 待验证 |
| 回退 | unit / 关闭本地 registry 后 rollback | 待验证 | 待验证 |
| 租约、安全清理 | TestRunningReleaseRetainedAndUninstallProtected、TestStageLeaseAndRepair | 待验证 | 待验证 |
| 离线启动 | registry 关闭后假包 startV2；旧安装兼容 unit | 待验证 | 待验证 |
| 中文路径 | 固定 Node 下载、发现及真实 npm 集成 | 待验证 | 待验证 |

CI：[Phase 01 native verification](https://github.com/Routilet/routilet/actions/workflows/phase-01.yml)。每个平台独立运行 go test（含实际下载固定 Node）、go vet、npm probe、打包，上传日志及测试产物；以 PR 对应 run/commit 为准。CI runner 不等于用户真机。真实公开 registry 安装测试默认不运行，不发布任何假包。

## 真机验收

使用专用 OS 测试账户，不复用正式账号、课程及安装。记录操作系统版本、架构、CI run/commit、产物名、安装和离线启动实际结果。中文路径、取消候选、失败后旧版、回退、正在运行时清理、渠道隔离须分别实测；没有第二个可用版本时记录阻塞，不声称通过。保留失败日志，不关闭全局安全机制。macOS 下载 tar.gz 后解压，双击 Routilet.command；如 Gatekeeper 阻止，仅按可信来源单应用授权方式处理。Windows 解压测试包并启动 Routilet.exe。

本次执行容器为 Linux Go 1.23.2，网络无法解析 GitHub，不能在该容器独立下载所需 Go module/toolchain；原生 CI 是自动验证入口。Windows 维护者实机验收未完成。Mac 总体验收已由维护者确认，详见下方记录。

初次 [CI run 45944334693](https://github.com/Routilet/routilet/actions/runs/45944334693) 的 macos-14 runner 未分配即取消；该镜像已于 2026-11-02 [退役](https://github.com/actions/runner-images/issues/13518)。改用 macos-15（arm64），仍在 CI 中强制检查宿主架构。

## CI run #2 实际结果（d5e4baf）

[运行 45944392044](https://github.com/Routilet/routilet/actions/runs/45944392044)：两平台 go test（含 Node 下载校验、中文路径、本地 registry）、go vet、npm probe 均通过。macOS arm64 构建、签名校验及上传通过，[Mac 产物](https://github.com/Routilet/routilet/actions/runs/45944392044/artifacts/15174082237)。Windows 在生成 HTML 指南时因多行模板匹配失败（疑为 CRLF）中断，尚未执行最终 go build；[Windows 日志](https://github.com/Routilet/routilet/actions/runs/45944392044/job/140325658938)。修复读取模板时归一化 CRLF，并增加 LF/CRLF 输出一致性测试；修复后的 CI 结果见下方。

## CI run #3 与 Mac 真机验收（2026-11-27）

- [CI 45944766397](https://github.com/Routilet/routilet/actions/runs/45944766397)，源码提交 `8495d44c0162d540e6870920a54b184bbdeba368`：两平台 Go 测试、go vet、npm probe、LF/CRLF 回归测试、原生打包及产物上传均通过。 工作流最终状态：success。
- [Windows amd64 验收包](https://github.com/Routilet/routilet/actions/runs/45944766397/artifacts/15174536249)；[Mac arm64 验收包](https://github.com/Routilet/routilet/actions/runs/45944766397/artifacts/15173724070)。
- [维护者 Mac 真机验收确认](https://github.com/Routilet/routilet/pull/1#issuecomment-7210513104)：总体成功；具体系统版本、产物编号及逐项实测记录未提供，上表的逐项 Mac 证据仍待补充，不推定全部故障场景均已执行。
- Windows 真机验收仍待维护者确认；不发布、不合并，等待审核。
