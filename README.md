# Routilet

轻量桌面应用管理器，为 Routilet 旗下独立应用提供统一的安装、更新、回退、启动与运行环境管理。

**当前状态：规划完成，尚未开始实现。** 本仓库目前仅包含决策与执行文档，不是可下载的正式软件。现有 Accoding Launcher 及 Companion 正式版本继续可用。

## 产品目标

用户首次下载 Routilet，然后按需安装 Companion、Observer 或未来应用。各应用可独立启动、卸载及协作，不要求管理器一直运行。Exam Kit 等资源作为依赖分发，而非强制安装的独立应用。

目标平台为 Windows 11 amd64 和 macOS Apple Silicon。使用 Go，管理界面采用已验证的 Wails v3 + Web UI；保留 npm 作为内部发行基础设施。

## 开发入口

- [组织级架构与共同决策](https://github.com/Routilet/.github/blob/main/docs/ecosystem.md)
- [总体路线图](docs/roadmap.md)
- [第一阶段执行说明](docs/phase-01.md)
- [架构与待定接口](docs/architecture.md)

迁移来源：[accoding-companion/launcher](https://github.com/Routilet/accoding-companion/tree/main/launcher)。

第一阶段只迁移现有安装内核并建立独立测试基线，不提前开发 GUI、多应用安装或自更新，不发布正式版本。待维护者分配任务后开始执行。
