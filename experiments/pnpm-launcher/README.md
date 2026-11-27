# Routilet 独立 pnpm 启动器实验

实验不修改 launcher/、正式发行或组织决策。Go + pnpm 12.22.0 原生客户端，使用 npm 官方仓库 https://registry.npmjs.org/，不引入镜像、不覆盖 nodeLinker 或 packageImportMethod、不配置系统 PATH。仅用独立 store/PNPM_HOME 与禁用安装脚本；候选目录不移动、不修改已安装包文件。

## 使用

下载实验分支 Actions 对应平台 artifact 并完整解压内部 ZIP / tar.gz。Windows 双击 routilet-pnpm-probe.exe；macOS 双击 launch.command（或在终端运行 ./routilet-pnpm-probe）。macOS 实验入口 ad-hoc 签名，不公证。不要关闭系统安全机制。pnpm 与启动器同目录，首次下载包含 pnpm，不在用户机器访问 GitHub 下载运行时。后续安装访问官方 npm。

菜单：安装 esbuild 0.25.11 → 更新 0.25.12 → 运行原生程序 --version → 回退。也可在新解压目录使用 --self-test 自动验证以上流程与离线 frozen-lockfile 恢复。关闭网络后选择启动，预期直接运行已安装原生程序；启动器自检 pnpm --version 不联网。数据仅在相邻 probe-data/；删除此实验目录即可卸载（先关闭实验程序）。不触碰正式安装或课程文件。

esbuild 是真实 npm 原生 Go 程序，使用平台可选依赖，故可验证平台包解析、实际运行和无 Node 的安装链路，而不必发布假包或修改 Observer/Companion。此实验不是最终产品，也不启动真实 GUI 应用。

## CI

固定 pnpm 官方 Release 12.22.0 的双平台归档和 SHA-256，CI 获取并校验；产物随附 pnpm 和上游许可证（如果发行归档中未附则从固定 tag 获取 MIT LICENSE）。Go 只用于构建，用户无需 Go 或 Node。CI 在原生 Windows x64 / macOS arm64 runner 测试从打包内容启动，进程 PATH 排除常见 Node 安装位置并检查无法发现 node。CI 网络连接不是国内网络实测。安装数据不进入 artifact。

## 验收及限制

- [ ] Windows amd64 CI：构建、原生 pnpm、真实 npm 安装、更新、失败不改变指针、回退、离线锁定恢复。
- [ ] macOS arm64 CI：同上；签名与执行权限。
- [ ] 双平台真机：双击启动，官方 npm 下载，断网启动，重复更新与回退。
- [ ] 后续专门验证签名 APP、现有 Observer/Exam Kit 包、资源定位及运行租约。
- [ ] 实际 artifact/可执行文件尺寸与安装耗时由 CI 日志记录。

单实例顺序操作；不实现多应用模型、Manager 自更新、GUI、自动清理、运行租约或完整 Windows 原子性/崩溃恢复。版本使用确切旧版本，保留 pnpm 默认安全策略；不测试新发布版本的冷却策略。默认链接/共享存储不保证可写文件隔离；禁止应用修改安装目录，后续验证。pnpm 12 为 Rust 重写，实验结果不等同于正式产品验收。

参考：[pnpm 安装文档](https://pnpm.io/installation)、[固定上游发行](https://github.com/pnpm/pnpm/releases/tag/v12.22.0)、[esbuild 平台包](https://github.com/evanw/esbuild/blob/master/npm/esbuild/package.json)。
