# Routilet

![Routilet](desktop/frontend/icons/routilet.svg)

Routilet 源于 routine：减少重复性工作，使人专注于真正值得思考和创造的事情。本项目是系列工具的独立桌面启动台。

- [下载](https://github.com/Routilet/routilet/releases) · [npm](https://www.npmjs.com/package/@routilet/routilet)
- Windows 11 x64 / Apple Silicon macOS（12+），系统 WebView，无需用户安装 Node.js 或配置 PATH。
- 固定原生 pnpm 12.22.0；官方 npm 仓库、版本化快照、回退及离线启动。
- Observer 与 Companion 使用独立原生 npm 包（`@routilet/c-observer`、`@routilet/companion`），当前通道无兼容版本时不可安装。

完整解压后运行 Routilet.exe / Routilet.app。首次入口及随附 pnpm 不要拆开。macOS ad-hoc 签名、未公证；Windows 使用系统 WebView2。底部“数据目录”打开安装数据；不自动迁移课程或旧启动台数据。

## 开发与文档
- [正式桌面源码](desktop/) · [技术说明及真机验收](desktop/README.md)
- [架构](docs/architecture.md) · [路线图](docs/roadmap.md) · [发行说明](docs/release-notes.md)
- [共同生态决策](https://github.com/Routilet/.github/blob/main/docs/ecosystem.md) · [图标设计说明](https://github.com/Routilet/.github/blob/main/docs/icon-design.md)
- [正式 CI 与发行](.github/workflows/publish-npm.yml)：PR 双平台构建，不发布；main VERSION 变更触发发行；已有 Release 时复用原始校验包。npm Trusted Publisher 使用此文件名。
- [旧 Launcher v2 归档](https://github.com/Routilet/routilet/tree/archive/launcher-v2) · [固定 pnpm 验证](https://github.com/Routilet/routilet/pull/2)

MIT · Routilet.
