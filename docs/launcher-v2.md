# Launcher v2：统一应用安装与更新

维护者确认：用户只有一个 Accoding 更新入口，更新确认页展示组件变化。开发者仍独立发布各 npm 包。仓库使用 Routilet/accoding-companion、Routilet/c-observer、Routilet/c-exam-kit；正式包使用 @routilet/accoding-companion、@routilet/c-exam-kit、@routilet/c-observer；旧包仅供历史兼容。组织迁移后另行核验发行元数据及 Trusted Publisher。

## 阶段一：安装内核

- tests/standalone/npm-dependency-probe.mjs：回环 HTTP registry，测试作用域假包，不向真实 registry 发布。验证精确根版本、间接依赖兼容更新、精确预览依赖、缺失可选依赖、禁用安装脚本、候选锁文件和 npm ci。
- launcher/internal/managed：独立安装内核。仅声明 Companion 根依赖。候选锁文件生成、版本差异（包括 npm 路径；可选项不是安装保证）、锁文件 SHA-256、npm ci、CLI 校验、激活记录一次替换、写锁、过期候选拒绝、回退。稳定版和预览版隔离，保留历史安装，直到运行租约及安全清理实现。
- 使用现有 Node 运行时，不配置 PATH，不安装编译器。机器命令有超时，禁用 npm 生命周期脚本。

安装目录：旧 packages/current 暂不改变。v2/channels/{latest,next}/activation.json 保存 current/previous；releases/release-*/ 保存 package.json、package-lock.json、installation.json、node_modules；write.lock 用于并发写操作。

## 接口

accoding --installation-info 在构造业务对象之前执行，无网络、无登录、无 TUI。schemaVersion=1、version、ready、components(name/package/version/required/status)。必要依赖缺失应失败，可选依赖不可用不阻断。必须使用与实际程序一致的 npm 模块解析。没有新接口的旧 Companion 保留旧安装启动路径，不认作新版受管理安装。

## 计划

1. 迁入已验证的本地安装内核，真实 npm/三平台 CI 复测。
2. Companion 安装信息、配置隔离；与 Observer 集成协调。
3. 新版启动器入口、统一候选确认、渠道切换、迁移及离线启动、回退。
4. 运行租约、安全清理、故障及 Windows 文件替换验证。
5. HTML 定稿审核、预览发行、实机及断网验收。Observer Windows/GDB 暂缓，启动器 Windows 支持保留。

本地原型已验证 Node 22.16.0/npm 10.9.2、Go 1.23.2 Linux amd64；Windows amd64、Darwin arm64 交叉编译。提交后的 CI 需独立确认，不等于实机验收。阶段一不改变现有入口、正式版本或发行工作流。

## 安装信息接口实现

新增 --installation-info，复用严格本地 C Exam Kit 解析，不访问全局 npm。Observer 使用 manifest / --capabilities 本地检查；缺失或不支持平台不阻断。其运行入口仍由 Observer 集成分支维护，汇合时将其 resolvePackageFile / observerBinaryFromManifest / probeObserverBinary 改为共用接口。C Exam Kit 依赖同 Observer 分支，旧启动器兼容保留。ACCODING_MANAGED_V2=1 禁用运行时全局包发现。仅安装信息接口与依赖调整，不提前发布 Observer。

## 开发入口与配置隔离

运行 launcher --managed-v2 可进入开发中的新版入口；默认仍运行旧版，待配套 Companion 预览版可用并验收后再切换。首页安装/启动/统一更新、候选锁差异确认、渠道切换、离线旧安装启动和回退已接入。预览配置及数据使用独立 profile，macOS Keychain 使用独立服务名，Windows DPAPI 使用独立配置目录。用户可以主动选择其他课程目录；不是安全沙箱。当前不提供删除历史安装、卸载或自动清理，直到运行租约和 Windows 文件替换验证完成。轻量后台只检查 Companion 标签，手动更新会完整解析间接依赖。

## 运行保护和卸载

已添加跨平台共享/独占安装租约：启动时锁定实际 release，允许另一个实例更新但不允许删除正在运行的旧安装。更新后保留 current 与 previous，清理更旧且无租约的 release。卸载当前 v2 渠道前要求所有 release 无运行租约；保留另一渠道、旧安装、Node、账号配置和课程。失败暂存目录在正常失败时清理；异常退出遗留 stage 的安全清理尚未接入。原 npm 技术实验已接入 npm test，需核对真实 CI。

补充：后台通过 npm outdated --all --json 的 wanted/current 提示可能的兼容更新，实际确认仍以完整锁文件为准；超时不阻塞启动。候选 stage 持租约，超过 24h 且无租约的残留目录可在安全清理时删除。缺失旧锁文件可显式重新解析修复，不先删除旧安装。回退和卸载仍不改课程。

新版整体候选已改为实际暂存安装再确认；旧的 lock-only/npm ci 说明仅为历史方案。实际组件状态在确认页面展示，确认后不再重新解析依赖。

## 正式发行决定（最新）

Launcher 0.3.0 通过 --managed-v2 打包为默认入口，配套 Companion 0.2.8 与 Observer 0.1.0。仍保留旧安装启动路径。实际安装使用 staged npm install/update，不采用旧 npm ci 方案。安装状态校验不可跳过；旧 Companion 缺少 --installation-info 时明确报版本和命令。运行环境有效性包含 npm --version，三平台 CI 新增真实 npm 本地 registry 的安装/更新/回退及不兼容版本拒绝；发布 ZIP 前从实际公开 registry 验证配套版本。用户将使用正常正式 npm 流程做实机验收。
