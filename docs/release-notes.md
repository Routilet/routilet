# Routilet 0.1.6

- 接入独立可选组件 Accoding Adapter（`@routilet/accoding-adapter`），支持安装、更新、修复、回退、禁用和卸载。
- 首页常驻展开“可选组件”区域，与独立应用区分；安装不启动组件、不退出 Routilet。
- 新增 routilet-adapter-call/1：离线可用性检测及受保护的 stdio 调用。沿用快照、完整性校验及运行标记；安装前验证 Adapter 离线 capabilities。
- 保留 v0.1.5 Windows hoisted/copy 安装及独立缓存修正；用户已报告 Windows 实机验证正常。

当前 Adapter beta 在测试通道提供。Companion 正式调用路径需由 Companion 侧接入；本版不改变业务协议，不持久化 Adapter 的 stdio 或 stderr，不引入常驻服务。实际在线评测流程仍需联合验收。
