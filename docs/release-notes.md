# Routilet 0.1.5

- Windows 新安装快照明确使用 pnpm hoisted 普通目录和 copy 导入，独立 store-native 缓存；不再复用旧 store 中的内容，不删除旧缓存/安装/课程数据。macOS 保持原状。
- 保留 manifest 路径范围和 SHA256 校验，不尝试绕过缺失文件或安全软件。
- 原生入口缺失时报告 Stat 与父目录内容，区分路径解析失败、目录缺失及文件缺失。
- CI 验证实际 Observer 包物理目录、新 store、重复修复及现有 ZIP/self-update 测试。

用户报告 Observer 0.2.0 清单可读但 Windows 原生入口不存在或无法解析；尚未取得本机缺失文件的直接证据。此版为针对性修正，需用户复验。
