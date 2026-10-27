# Go-Remote 迁移与功能对齐方案

## 1. 目标

本仓库以 QuickDesk 为上游，目标是保留 QuickDesk 已经成熟的远程控制和媒体引擎，
把 Go-Remote 原有业务功能逐步迁移到 QuickDesk 的产品层、信令层和 Web 层。

核心原则：

- 远程控制、采集、编码、WebRTC、P2P/TURN 运行在 QuickDesk / Chromium Remoting 引擎内。
- 账号、设备、会话、审计、录制索引、权限、业务扩展和 Web 管理界面由本 fork 扩展。
- 默认使用 `quickdesk-remoting` 预编译产物，不要求日常开发编译 Chromium。
- 只有必须修改 Remoting 内核时，才进入 Chromium 编译流程。

## 2. QuickDesk 已经具备的能力

| 能力 | 当前实现 |
|---|---|
| 用户注册、登录、Token | Signaling Server：Go + PostgreSQL + Redis |
| 设备 ID、设备密钥、绑定 | Signaling Server + Host 自动 provisioning |
| 远程桌面和输入 | Chromium Remoting Host/Client |
| WebRTC P2P 和 TURN | 信令 + coturn |
| 音频、剪贴板、文件传输 | Remoting Host/Client + Qt/Web 客户端 |
| 多显示器、虚拟屏、隐私屏 | Host 能力 + Qt UI |
| 连接历史和收藏 | Signaling Server |
| 审计、Webhook、设备分组 | Signaling Server 管理后台 |
| Web 控制端 | `WebClient/` |
| MCP / AI 自动化 | `quickdesk-mcp` + Qt API |
| macOS / Windows 客户端 | Qt 6 + QML |

## 3. Go-Remote 原有能力差异

### 3.1 已完成对齐

- 账号认证、设备绑定、连接历史、收藏和管理员审计已经由 QuickDesk 覆盖。
- 远程桌面、输入、剪贴板、文件传输、音频和多显示器已经由 QuickDesk 覆盖。
- Web 控制端和自托管信令已经由 QuickDesk 覆盖。

### 3.2 当前分支开始补齐

- `session_recordings` 控制面模型和迁移。
- 用户录像索引 API：
  - `GET /v1/me/recordings`
  - `POST /v1/me/recordings`
  - `GET /v1/me/recordings/:id`
  - `DELETE /v1/me/recordings/:id`
- 该能力先提供录像发现、归属和生命周期元数据；媒体文件仍由 Host/Client 引擎生成。

### 3.3 仍需补齐

1. 屏幕录制和回放引擎：Host/Client 生成录像文件并上报 to `/v1/me/recordings`。
2. Web 端录像列表、播放、下载和删除页面。
3. 屏幕墙 / DeskHub：多设备画面集中查看和批量操作。
4. 更完整的远程文件管理器：目录浏览、重命名、删除、分片上传下载。
5. Go-Remote 业务策略：会话策略、通知、计划任务和自定义审计事件。
6. 文档、自动化测试和发布链统一。

## 4. 分阶段实施

### 阶段 A：控制面基础

- 新增录像索引和 API。
- 补充录像事件和审计。
- 为 Web 管理端和用户端提供查询基础。

### 阶段 B：媒体录制

- 在 Host/Client adapter 中增加录制启停控制。
- 生成媒体文件和缩略图。
- 上传/归档后回写录像元数据。

### 阶段 C：Web 回放与管理

- Web 用户端增加录像列表。
- 支持在线播放、下载、删除。
- 管理后台增加录像检索和使用统计。

### 阶段 D：屏幕墙和业务扩展

- 增加多设备屏幕墙。
- 补充设备分组、批量操作和业务策略。
- 接入通知、任务和自定义工作流。

## 5. 上游同步约定

- 上游仓库：`barry-ran/QuickDesk`。
- 开发基线：`upstream/dev`。
- 功能分支：`codex/*`。
- 不直接修改预编译 Remoting 包。
- 涉及 Remoting 内核的需求先评估是否可以通过 adapter、MCP 或 Qt API 完成。

## 6. 移动主控交互对齐

Web 手机模式与 Flutter Android 采用共同的触控板交互契约、显式点击/拖拽入口、全缩放级别双指滚动、焦点缩放、提交态 IME 输入及生命周期取消。详细设计和真机验收边界见[移动端远控交互规范](移动端远控交互规范.md)。共享回归数据位于 `test/fixtures/mobile-gestures.json`，由两端测试分别消费；Flutter 另外验证原生逐指采样到帧合并的路径，不能只依赖同时移动两指的理想化数据。桌面鼠标/键盘及预编译 Remoting 不变。

## 7. App 账户设备与三端排序

账户/设备/收藏的 App 补齐范围、源代码依据、安全边界及其他未对齐能力见 [App 账户设备功能对齐](App账户设备功能对齐.md)。三端排序以共享夹具为准，不将设备心跳时间误作最近使用时间，不因 App 登录自动转移设备归属。


### App 第二阶段：绑定、连接记录与密码重置

本机 Host 凭据现按服务器/API key 隔离；账户页显式确认后才可绑定/转移归属，登录与重连不自动绑定。主控连接记录复用既有 API，按账户/session epoch 隔离终态并禁止不确定 POST 重试。忘记密码复用 preset 的 WebClient URL 与 Qt hash 路由。完整协议依据、旧凭据升级行为与验证边界见 [App 账户设备功能对齐](App账户设备功能对齐.md)。画质设置与被控连接管理作为后续独立 PR 审核；不修改 Remoting 核心。
