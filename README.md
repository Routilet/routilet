# Routilet

轻量桌面应用管理器，为 Routilet 旗下独立应用提供统一的安装、更新、回退、启动与运行环境管理。

**当前状态：阶段 01 独立 Launcher v2 迁移审核中，未正式发布。** 现有 Accoding Launcher 及 Companion 正式版本继续可用。当前仅支持固定 Companion 根依赖和旧 TUI，不代表目标架构已经实现。

## 独立构建与测试

需要 Go 1.24.9、Node.js 22.12+（CI 使用 24）及 npm。无需 checkout Companion 或安装其 TypeScript 依赖。

```sh
cd launcher
go test ./...
go vet ./...
go build -o routilet .
```

从仓库根目录执行 npm 本地假包测试（不发布任何包）：

```sh
node --test tests/standalone/npm-dependency-probe.test.mjs
```

原生 CI 在 Windows amd64、macOS arm64 上额外设置 ACCODING_TEST_NODE_DOWNLOAD=1，下载并校验固定 Node 发行文件，在中文临时路径执行真实 npm 集成测试。原生打包：从仓库根目录执行 `bash scripts/package-launchers.sh dist`（Windows 使用 Git Bash）。macOS 产物为 tar.gz，保留执行权限及 ad-hoc 签名；Windows 产物为目录，由 Actions 打成 ZIP。

**测试安全：** 阶段 01 保留旧 accoding-launcher 安装目录及 Companion 稳定/预览 profile 协议。手动运行测试产物必须使用专用 OS 测试账户、空白配置和测试课程，不要使用正式账号或课程。自动化测试使用临时目录和本地假包，不接管现有安装。

## 结构

- launcher/：独立 Go module、原 TUI、运行时和兼容入口。
- launcher/internal/managed/：原事务安装、租约及清理内核。
- tests/standalone/：npm 依赖解析实验。
- docs/launcher/：原离线指南资源。
- scripts/：独立打包与指南生成。

[组织级架构](https://github.com/Routilet/.github/blob/main/docs/ecosystem.md) · [路线图](docs/roadmap.md) · [阶段 01](docs/phase-01.md) · [目标架构](docs/architecture.md) · [迁移报告](docs/phase-01-migration.md) · [测试报告](docs/phase-01-test-report.md)

源项目及 MIT 许可见迁移报告和 LICENSE；原产品历史说明保留在 docs/launcher-v2.md。
