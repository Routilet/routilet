# pnpm 启动器实验结果

日期：2026-11-27；实验提交 [b776d0e](https://github.com/Routilet/routilet/commit/b776d0e3ea92865936a595257cfdf3730fc1a9d8)，[PR #2](https://github.com/Routilet/routilet/pull/2)。

## CI

[实验 Actions #45947142149](https://github.com/Routilet/routilet/actions/runs/45947142149) 双平台成功。Go tests/vet、上游 pnpm 摘要校验、原生打包、中文及空格路径解包、排除 Node PATH、官方 npm 安装、更新、旧版本验证、失败候选不改变激活、回退及 offline frozen-lockfile 恢复通过。另有原阶段 01 回归 CI #45947154518 成功；不等于阶段 01 真机验收。

| 平台 | Actions artifact 压缩字节数 | CI | 真机 |
|---|---:|---|---|
| Windows amd64 | 28773804 | 通过 | 待验收 |
| macOS arm64 | 26221928 | 通过 | 部分通过 |

大小是 Actions artifact 外层归档，不是解压后占用或启动器单文件大小。

## macOS 维护者实测

维护者提供终端日志：首次 pnpm --version 出现 signal: killed，之后使用 bash 执行 launch.command 成功，pnpm 12.22.0，Node not on PATH。安装 esbuild 0.25.11 用时 10.754s，更新 0.25.12 用时 7.414s，原生启动返回 0.25.12；回退至旧快照并再次切换回新版均成功。路径位于 iCloud Drive，含空格；pnpm 报告从内容寻址 store clone 文件至虚拟 store。运行时出现 12.23.0 自更新提示，未执行自更新。

首次 SIGKILL 原因未确定（不能据此断定签名、Gatekeeper、iCloud 或其他因素），复现频率未知。双击直接启动、断网真机启动、Windows 真机、真实 Observer APP/Exam Kit、运行租约与自更新尚未验收。没有证据证明系统完全未安装 Node，仅证明实验进程 PATH 无 Node。

## 结论

初步技术可行性通过；尚不宣称正式发行验收完成。采用官方 npm 源（维护者此前确认国内访问可用）、pnpm 默认链接与存储策略，不自动更新 pnpm。后续评估首次启动异常以及实际应用包兼容。未经维护者确认不合并、不发布、不调整组织决策。
