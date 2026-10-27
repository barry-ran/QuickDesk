# App 画质与被控连接管理

本批位于账户绑定/云连接记录之后，PR 按前者、后者顺序审核合入。仅产品层，不修改 Chromium Remoting 内核。

## 主控画质与音频

远程工具栏“更多 → 画质与声音”提供：

- 目标帧率请求：5 / 15 / 30 / 60 FPS。
- 优先最低码率请求：自动（0）或 2 / 5 / 10 / 50 / 100 × 1024² bit/s。**这是 preferred minimum，不是带宽上限**，网络和远端编码器决定实际码率。
- 远端分辨率请求：720p 至 4K，96 DPI；有选中屏幕 ID 时带上该 ID。会请求修改远端桌面分辨率，不是本地缩放。
- 音频静音：同时禁用当前和随后到达的本地接收音轨，并通过已有 AudioControl 通知远端。Android 被控目前没有音频采集，因此不能承诺其系统音频。
- 持久化首选编码器：自动 / H264 / VP8 / VP9 / AV1，明确**下次连接**生效；使用本机 receiver capabilities 重排，不新增不支持的 codec，不删除 RTX/FEC 或其他回退。首次 offer 和后续 host offer 的 answer 均应用相同顺序。平台拒绝时保留默认协商，不重写 SDP。

### 协议证据与限制

字段与 `WebClient/js/protocol/protobuf-messages.js`、`WebClient/js/remote-main.js::_handleSettingChange` 一致：

| 请求 | ControlMessage 字段 | 内部字段 |
|---|---:|---|
| VideoControl | 3 | enable=1, target_framerate=5 |
| PeerConnectionParameters | 14 | preferred_min_bitrate_bps=1 |
| ClientResolution | 2 | width=1, height=2, x_dpi=5, y_dpi=6, screen_id=7 |
| AudioControl | 5 | enable=1 |

Qt 语义参照 `ClientManager::setTargetFramerate/setBitrate/setResolution` 和 `FloatingToolButton.qml`；codec 顺序参照 `WebClient/js/protocol/session.js::_applyCodecPreference`。Flutter 原生接口参照 [RTCRtpTransceiver 文档](https://flutter-webrtc.org/docs/flutter-webrtc/api-docs/rtc-rtp-transceiver/)。

现有协议没有 FPS/码率/分辨率的逐项能力标志，也没有成功确认。UI 不宣称设置已经生效。保守请求门槛：控制通道就绪，且远端有至少一个已知桌面行为标志 `sendAttentionSequenceAction`、`lockWorkstationAction`、`fileTransfer`、`privacyScreen` 或 `virtualDisplay`。这仅是**兼容性推断**，不是对画质能力的保证；界面明确提示可能被忽略。空/未知能力集合（包括目前 Android Host）禁用远端画质请求。音频本地静音和下次连接的 codec 首选项不依赖远端此推断。没有增加未经实现的能力声明或绕过 Android MediaProjection 权限。

## 被控连接管理

- 只列出已经认证且双数据通道就绪的客户端：显示信令 session 的 opaque client ID 与连接时间，不把它声称为用户名或可信设备名称。
- 单独断开需要确认；沿用已有 Jingle session-terminate 和该 peer 的通道/PeerConnection 关闭，其他客户端不受影响。关闭开始立即阻止排队输入，停止 Host 时先关闭新会话准入。异步确认绑定原 controller 和连接时间，不会误断后来重连的同 ID 会话。
- 单独断开不是拉黑：知道有效访问码的人仍可重连，确认框明确提示。
- 访问码只可在 Host 停止后刷新；界面提示下一次启动生效。请求期间按钮禁用，网络失败可重试，只有服务端成功才保存新码。Controller 也检查会话/阶段并互斥刷新与启动，避免界面之外调用让在线 SPAKE2 哈希与服务端访问码不一致。
- 未加入自动周期轮换、永久密码、封禁或绕过系统权限；这些不在当前 Android 已验证实现范围。

## 验证

`quality_controls_test.dart` 覆盖固定 wire bytes、发送通道就绪/关闭与范围边界、codec 回退顺序/存储校验、未知能力禁用、无匹配客户端时幂等断开、只读快照和禁用远端请求的 widget。`host_clients_test.dart` 通过生产使用的生命周期接口验证只关闭匹配 peer、异步关闭不误删替代会话、过期确认无效和重复断开幂等；不把 fake peer 测试声称为真实 WebRTC 多端集成测试。完整 `flutter analyze` / `flutter test` / APK 由 CI 执行；本地 SDK bootstrap 已被执行环境拒绝，不重试或绕过。

仍需真机互通：Qt Host 实际接受分辨率/码率请求的效果、Android 厂商 codec 支持、多个真实客户端分别断开、访问码停止后刷新与重连。未使用真实账户、短信或设备授权做测试。
