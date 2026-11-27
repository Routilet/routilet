# Routilet 0.1.0

首次独立发行：轻量应用启动台。支持 Windows 11 x64、Apple Silicon macOS。使用系统 WebView，无需安装 Node.js 或配置 PATH；固定 pnpm 12.22.0 随首次下载分发，后续使用 npm 官方仓库。

- 紧凑应用列表，点击安装或打开；可用更新才显示更新按钮。
- 安装/更新进度、取消、修复、卸载、回退、快捷方式与保守运行保护。
- 正式/测试通道；测试通道优先选择较新的兼容版本，避免过时 next 覆盖正式版本。
- Routilet 自更新及原入口回退，实际线上更新将在首次发行后验证。

**重要：Companion 和 Observer 新版仍在开发，不承诺本次能安装真实应用。** 默认正式通道；未提供兼容版本时显示不可用。没有旧 TUI 自动迁移。CI 的自更新使用原生测试候选，真实 GUI、快捷方式和运行保护仍需真机验收。本次为早期 v0.1.0，不建议替换现有考试工具安装。

下载对应系统归档、完整解压，打开 Routilet.exe 或 Routilet.app。macOS ad-hoc 签名，未公证；Windows 需系统 WebView2。不关闭系统安全保护。数据目录：macOS ~/Library/Application Support/Routilet，Windows %APPDATA%/Routilet。旧 Routilet Preview 保留。关闭相关程序后可手动清理，不会主动删除课程源码。

npm tarball 为管理器更新包，不是初始下载入口。初始入口须保留原位置；pnpm 不独立自更新。SHA256SUMS 用于核对下载文件。
