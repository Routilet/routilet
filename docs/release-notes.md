# Routilet 0.1.1

首次发行后的工程收尾，不增加产品功能。

- 新的 Routilet 归并图标；Windows EXE 资源、macOS ICNS、运行时及应用列表统一矢量源。
- 正式源码迁移至 desktop/，Go 依赖锁定，历史 Launcher v2 归档，PR 双平台测试与正式发行统一。
- 更新架构、路线图、品牌说明；保持现有数据根与自更新协议。

已安装 v0.1.0 可用底部 Routilet 入口检查更新并重新打开。初始入口不会被覆盖，原始 Finder/Explorer 图标不会自动更改；如需新初始入口图标，请在完成自更新验收后另行下载新版完整归档。Windows 11 x64、Apple Silicon macOS；固定 pnpm 随包。真实在线自更新、GUI、快捷方式和生命周期仍需真机验证。Companion、Observer 尚未完成接入，不自动迁移旧版工具及课程数据。
