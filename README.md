# Routilet

轻量应用启动台：展示、打开、安装与更新 Routilet 旗下独立应用。Go + Wails，固定原生 pnpm，npm 官方仓库。无需用户安装 Node.js。

当前首发版本：0.1.0。以 [GitHub Releases](https://github.com/Routilet/routilet/releases) 实际发行资产和 [npm](https://www.npmjs.com/package/@routilet/routilet) 状态为准。Companion、Observer 新版仍在开发，实际应用接入在首次发行之后验收；本版不替代现有课程工具。

## 使用
Windows 11 x64 下载 ZIP；Apple Silicon macOS 下载 TAR.GZ，完整解压后打开 Routilet.exe / Routilet.app。系统 WebView2 / WKWebView；Mac ad-hoc 签名、未公证。固定初始入口与随附 pnpm 不要拆开或移动。首页底部“数据目录”可直接打开安装数据；macOS ~/Library/Application Support/Routilet，Windows %APPDATA%/Routilet。旧安装与课程配置不自动迁移。

## 开发
新桌面启动台源码暂保留在 [experiments/launcher-gui](experiments/launcher-gui)，避免在首次发行时移动已验证的构建路径。旧 launcher/ 保留为历史兼容基线，不参与新发行。

- [新路线图](docs/roadmap.md)
- [启动台技术说明](experiments/launcher-gui/README.md)
- [发行说明](docs/release-notes.md)
- [正式发布工作流](.github/workflows/publish-npm.yml)

发行由 main 的 VERSION 变更触发；手动 dispatch 可重试。npm 与 GitHub Release 独立发布，使用同一构建的校验产物。npm trusted publisher 配置：GitHub Routilet/routilet，publish-npm.yml，允许 npm publish；首次若尚未建立授权需维护者完成 npm 设置或首次手动发布。不要在仓库保存令牌。

MIT · Routilet.
