# Repository Guidance

## 项目定位

- 本仓库是 `barry-ran/QuickDesk` 的 fork，目标是逐步加入 Go-Remote 的业务能力。
- 远程控制和媒体核心优先复用 QuickDesk / Chromium Remoting，不重复实现 Windows 采集、输入、编码和 WebRTC。
- 默认开发不需要编译 Chromium；`quickdesk-remoting` 通过预编译 release 下载。
- 只有明确修改 Remoting 内核时才进入 Chromium 编译流程。

## 上游与分支

- `upstream` 指向 `barry-ran/QuickDesk`。
- 功能开发基于 `upstream/dev`。
- 使用 `codex/` 前缀创建本地或远端功能分支。

## 功能开发原则

- 控制面、业务能力、UI 和扩展优先修改 Go、Vue、QML/C++ 产品层。
- 不要为了一个小功能修改 `quickdesk-remoting`。
- 涉及 Remoting 的需求先考虑：
  1. 是否能通过现有 Qt WebSocket API 完成。
  2. 是否能通过 MCP 或 adapter 完成。
  3. 是否只需要增加 Host/Client 状态和控制消息。
  4. 最后才考虑 fork 并编译 Remoting。
- 功能差异和维护计划记录在 `docs/dev/Go-Remote迁移与功能对齐方案.md`。

## 验证

- Signaling Server：`cd SignalingServer && go test ./...`
- Signaling Server 构建需要先生成 `SignalingServer/web/dist`：
  `cd SignalingServer/web && npm install && npm run build`
- 修改 API 路由时同步更新 `SignalingServer/docs/user-api-docs.md`。
- 修改数据库模型时同步更新 `SignalingServer/migrations/001_init.sql`。
