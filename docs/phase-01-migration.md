# 阶段 01 迁移报告

## 基线与方法

- 来源：[Routilet/accoding-companion main](https://github.com/Routilet/accoding-companion/commit/95541ea4231d1ea53c514bf89ce52ebccd53d2ac)，实际 main commit 为 `95541ea4231d1ea53c514bf89ce52ebccd53d2ac`，API 返回 tree 为 `f244f2bcaad9f29204646c7184f6a6acdcbd7d3b`。规划文件将前者称为 tree SHA；以实际 branch API 为准。Launcher 0.3.0 / Companion 0.2.8。
- 采用独立导入，不重写目标仓库已有文档历史。执行容器无法访问 GitHub 网络，无法本地过滤历史；通过 GitHub Git API 按文件迁移，原作者历史可在上述源 commit 追溯。目标导入提交 c86c440、4444fb1、ebe424f。导入后核对 28 个迁移文件的 blob SHA 与来源完全一致。
- 保留原 MIT LICENSE（Better Accoding For BUAAers contributors）、原 Annouimo 作者信息、原指南及图片、Go module 依赖锁和全部 Go 测试。打包收集 Go runtime 和 Go module 第三方许可证。

## 独立化改动

- Go module 路径迁入 Routilet/routilet/launcher；TUI 默认 v2，旧 v1 代码和恢复逻辑保留，产品入口标注阶段 01。
- 打包不再读取 Companion package.json，仅固定原指南显示版本 0.2.8；不迁入 Companion 业务、TS 编译或 npm 发布脚本。
- 原 tests/standalone/launcher-smoke.mjs 仅调用 launch.cmd/launch.command（旧脚本启动器，并非 Go v2）；不迁移这些旧脚本，复用 Go v2 本地 registry 集成测试、npm 依赖 probe 及原 unit tests。增强真实 registry 的失败保留及关闭 registry 后的启动/回退。
- 安装模型、固定 Companion 根依赖、安装目录、渠道、旧包兼容、校验及运行租约保持不变。手动测试仅专用 OS 测试账户；自动测试临时目录。
- 原生 CI 只构建 Windows amd64 / macOS arm64，上传测试包，不触发任何正式发行。macOS ad-hoc 签名，Windows 无签名。

## 未进入范围

GUI、多应用模型、自更新、应用协作协议、仓库或包重命名、其他项目改动、正式签名及发布均未执行。旧 Launcher 保持可用。真机验收完成前不得进入阶段 02。
