# 第一阶段：迁移 Launcher v2 并建立独立基线

状态：待执行（本文件不是开工指令）。日期：2026-11-27。

## 目标与来源

从 [Routilet/accoding-companion](https://github.com/Routilet/accoding-companion) 迁移已有 Go Launcher，独立构建和测试；不增加新产品功能。规划时检查的源码树为 `95541ea4231d1ea53c514bf89ce52ebccd53d2ac`（tree SHA，非 commit SHA）；开工时记录实际选用的 main commit 与差异。现有 Launcher v0.3.0、Companion v0.2.8 为参考基线，以实际源码为准。

- [launcher/](https://github.com/Routilet/accoding-companion/tree/main/launcher)：Go module、managed 安装内核、运行时管理、菜单与测试。
- [设计历史](https://github.com/Routilet/accoding-companion/blob/main/docs/launcher-v2.md)
- [原 CI](https://github.com/Routilet/accoding-companion/blob/main/.github/workflows/verify-native-launcher.yml)
- 原打包脚本 scripts/package-launchers.sh、相关 guide、tests/standalone/launcher-smoke.mjs、npm-dependency-probe.mjs 及其测试：开工时核实哪些是必要的跨目录依赖。

## 执行步骤

### A. 确定基线并迁移

- [ ] 在新仓库开独立开发分支；记录来源 commit、迁移方法、作者与许可证。
- [ ] 尽可能保留 Launcher 相关 Git 历史；如过滤历史会破坏新仓库已有文档或需要重写主分支，则保留文档历史，使用独立导入并记录来源及理由，不强推 main。
- [ ] 迁移 launcher/go.mod、go.sum、VERSION、runtime.go、installation.go、main.go、managed_v2.go、guide 平台实现及必要资源，迁移全部相关 Go 测试。
- [ ] 完整保留 internal/managed 下的 manager、runner、cleanup、Windows/Unix 锁及测试。
- [ ] 更新 module 路径及必要引用；排查对 Companion 源码、打包脚本和说明文件的隐式相对路径依赖。保留许可、第三方声明与作者信息。
- [ ] v2 为主要开发路径；旧版迁移与恢复逻辑先梳理，不擅自删减。暂保留固定 Companion 根依赖、现有安装协议、版本化目录与 TUI 测试入口。
- [ ] 对新仓库的测试使用隔离临时数据目录，不接管现有正式安装、账号、课程配置。

### B. 独立验证与打包

- [ ] 从新仓库独立执行 go test ./...、go vet ./...；无需 checkout Companion 源码。
- [ ] 迁移本地 npm registry/假包集成测试，验证安装、更新、候选校验、失败保留旧版、回退、租约及清理；不得向真实 registry 发布测试包。
- [ ] CI 仅以 Windows amd64 与 macOS arm64 为正式桌面构建目标；分别使用原生 runner，上传测试 artifact，不发布 Release。
- [ ] 原有 TUI 暂作验证入口；不添加 Wails 或 Web 前端。macOS 若打包原生 APP，保留基础 ad-hoc 签名；不新增正式签名体系。
- [ ] 在 Windows 11 和 Apple Silicon Mac 实机验收，记录系统、产物、结果与未测试项。

### C. 文档及交接

- [ ] 更新 README 的实际状态和构建步骤，记录新仓库结构、来源、兼容约定和已知限制。
- [ ] 增加迁移报告及测试报告，包含 CI 链接、测试产物、提交号、真机结果。
- [ ] 提交 PR 供维护者审核；验收前不发布新版，不删除 Companion 旧 Launcher，不修改现有发行流程。

## 验收矩阵

| 项目 | Windows 11 amd64 | macOS arm64 |
| --- | --- | --- |
| 独立构建与启动 | 必测 | 必测 |
| Node 发现与安装（含校验） | 必测 | 必测 |
| npm 安装与依赖解析 | 必测 | 必测 |
| 更新候选差异确认 | 必测 | 必测 |
| 失败保留当前安装 | 必测 | 必测 |
| 回退 | 必测 | 必测 |
| 运行租约与安全清理 | 必测 | 必测 |
| 离线启动已安装程序 | 必测 | 必测 |
| 中文路径 | 必测 | 必测 |

尽量复用已有自动化测试；区分自动测试、CI 构建与实机操作证据。不能验证的项目明确记录，不虚报通过。

## 完成条件

独立仓库包含完整 Go 安装内核与必要测试；无原仓库源码依赖；原 v2 核心能力无退化；两平台 CI 产物与真机验收完成；文档与迁移来源齐全。维护者验收后方进入阶段 02。

## 明确不做

多应用安装模型、应用目录协议、Manager npm 自更新、GUI、Companion 或 Observer 业务修改、仓库/npm 包重命名、公共 UI 库、正式签名、公证及正式发布均不在本阶段。旧 Launcher 保留用于现有发行，待 Routilet 正式替代时再退役。仅在阶段 01 验收后冻结旧 Launcher 的功能开发（必要修复仍可进行）。
