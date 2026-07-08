# Android 客户端技术方案（主控 + 被控）

> 状态：**M1–M4 代码完成，`flutter analyze` 零问题，待真机联调**（M5 未开始）
> 工程位置：仓库根目录 `quickdesk-android/`
> 参考项目：RustDesk（Flutter 移动端架构）、Chromium Remoting（协议）、scrcpy（Shizuku/ADB 注入路线）

> 进度速览：
> - 主控端（M1/M2）：REST + 信令 WS + Jingle + SPAKE2 + WebRTC 收流渲染 + 触控/键盘；剪贴板、多显示器、连接历史/收藏、性能统计、文件传输、中英 i18n、Fluent UI 均已实现
> - 被控端（M3/M4）：Host 角色协议（两次协商 + SPAKE2 Bob）、屏幕采集前台服务、无障碍输入注入、Shizuku 增强档、自动选后端、权限引导、旋转同步
> - 协议栈（SPAKE2 / protobuf / Jingle）有单元测试（`test/protocol_test.dart`），与 WebClient JS 逐字节对齐

## 1. 需求描述

为 QuickDesk 增加 Android 客户端，具备双角色能力：

1. **主控端（Client 角色）**：手机连接并控制任意 QuickDesk 被控设备（Windows / macOS / 未来的 Android 被控），看画面、触控操作、键盘输入、剪贴板同步
2. **被控端（Host 角色）**：手机屏幕被其他 QuickDesk 主控端（桌面客户端 / WebClient / 手机）查看与控制，并可被 MCP 接入的 AI Agent 操控

约束：

- **协议必须兼容现有 Chromium Remoting 体系**（Jingle 信令 + SPAKE2 认证 + WebRTC 传输 + protobuf 消息），不引入第二套协议，服务端零改动或最小改动
- 复用现有信令服务器（`/v1/*` API + `/v1/realtime/signal`）与 TURN 基础设施

## 2. 现状资产盘点

### 2.1 WebClient：可移植的纯实现协议栈（最大资产）

`WebClient/js/` 是一份**不依赖 Chromium 二进制**的完整 Client 角色协议实现，浏览器能跑通说明协议栈可以在任何有 WebRTC 能力的平台复刻：

| 模块 | 文件 | 说明 |
|------|------|------|
| 信令传输 | `signaling/websocket-transport.js` | WS 首帧 auth（signal_token 一次性令牌） |
| Jingle 编解码 | `signaling/jingle-builder.js` / `jingle-parser.js` | session-initiate/accept/terminate、transport-info XML |
| SPAKE2 认证 | `auth/spake2.js` / `auth-util.js` | 严格对照 BoringSSL spake25519 + Chromium spake2_authenticator |
| protobuf 消息 | `protocol/protobuf-messages.js` | 手写 wire-format 编解码（event.proto / control.proto 子集） |
| 会话状态机 | `protocol/session.js` | RTCPeerConnection + DataChannel（event/control/文件传输/actions） |
| 触控输入 | `input/touch-handler.js` | 触屏 → 鼠标/触摸事件映射（移动端手势经验可直接复用） |

### 2.2 信令服务器：设备无关，无需改动

Android 端按现有 API 走完整生命周期即可：

- Host 角色：`POST /v1/devices:provision` 注册 → `heartbeat` → `PUT access-code` → `signal-tokens` → WS `role:host`
- Client 角色：`POST access-code:verify` 换 signal_token → WS `role:client` → Jingle 会话
- `GET /v1/ice-config` 获取 STUN/TURN

### 2.3 缺口（均已在本期填补）

| 缺口 | 说明 | 现状 |
|------|------|------|
| Host 角色协议实现 | WebClient 只实现了 initiator（Client）。Host 角色需要 Jingle responder、SPAKE2 Bob、WebRTC 协商、host 侧 control 消息 | ✅ `protocol/host_session.dart`（两次协商）+ `spake2_authenticator.dart` 的 Bob 侧 |
| Android 屏幕采集/输入注入 | Chromium Remoting host 无 Android 版，需用平台 API 原生实现 | ✅ Kotlin 前台服务 + 无障碍 + Shizuku |
| 移动 UI | 全新 Flutter 工程 | ✅ `quickdesk-android/`（Fluent 主题 + 中英 i18n） |

## 3. 参考 RustDesk 的架构分析

RustDesk Android 端架构（已验证的成熟路线）：

```
Flutter UI (Dart)
   ↕ FFI (flutter_rust_bridge)
Rust 核心（协议、编解码、网络）
   ↕ JNI / 系统服务
Kotlin 原生服务：
  - MainService：MediaProjection 抓屏 + MediaCodec 硬编，前台服务保活
  - InputService：AccessibilityService，dispatchGesture 注入手势，
    Android 13+ 用 FLAG_INPUT_METHOD_EDITOR 处理文本输入
```

**QuickDesk 采纳与差异**：

| 维度 | RustDesk 做法 | QuickDesk 方案 | 原因 |
|------|--------------|----------------|------|
| UI | Flutter | Flutter（同） | 跨平台，未来覆盖 iOS |
| 传输 | 自研协议 + 自管 socket | **WebRTC（flutter_webrtc）** | 必须兼容 Chromium Remoting；flutter_webrtc 底层就是 Google libwebrtc，硬件编解码齐全 |
| 协议核心 | Rust + FFI | **纯 Dart 移植 WebClient** | 协议逻辑（Jingle/SPAKE2/protobuf/状态机）是纯计算，JS→Dart 几乎 1:1 翻译；省掉 FFI 工具链复杂度 |
| 抓屏 | Kotlin MediaProjection + 自管 MediaCodec | Kotlin 前台服务 + flutter_webrtc `getDisplayMedia`（内部走 MediaProjection + libwebrtc 硬编） | 编码、拥塞控制交给 libwebrtc，不重复造轮子 |
| 输入注入 | AccessibilityService | AccessibilityService 起步 + **Shizuku/ADB 增强档** | 无障碍免 root 受众广；Shizuku 提供完整按键注入（scrcpy 路线） |

不采用 Rust FFI 的补充说明：QuickDesk 桌面端的 Rust 组件（quickdesk-mcp、skill-host）与远程协议无关；协议参照物是 JS 而非 Rust，Dart 移植路径最短。若未来性能瓶颈出现在 protobuf 编解码等热点，再考虑局部下沉。

## 4. 总体架构

单 Flutter 工程双角色，仓库新增 `quickdesk-android/`（Android 优先，结构上不排斥未来 iOS 主控）。**实际落地结构**：

```
quickdesk-android/
├── lib/
│   ├── main.dart               # 入口（Fluent 主题 + 语言切换）
│   ├── api/
│   │   ├── signaling_api.dart  #   主控 REST（access-code:verify / ice-config）
│   │   └── host_api.dart       #   被控 REST（provision/heartbeat/signal-tokens/access-code）
│   ├── protocol/               # WebClient JS → Dart 移植层
│   │   ├── signaling/          #   websocket_transport（{client_id,payload} 封装）/ jingle
│   │   ├── auth/               #   edwards25519 / spake2 / spake2_authenticator（Alice+Bob）
│   │   ├── proto/              #   protobuf_messages 手写编解码（event/control 子集）
│   │   ├── client_session.dart #   主控会话状态机（对照 session.js）
│   │   ├── host_session.dart   #   被控会话状态机（Jingle responder，两次协商）
│   │   ├── datachannel_handler.dart # event/control 通道
│   │   ├── file_transfer.dart  #   文件传输（file_transfer.proto，独立通道）
│   │   └── host_input.dart     #   被控端输入事件模型
│   ├── controller/             # UI：connect/remote/host/home + touch_input/keycode_mapper
│   │                           #      + clipboard_sync/video_stats/connection_store
│   ├── host/                   # 被控桥接：screen_capture/input_injector/host_credential_store/host_controller
│   ├── l10n/                   # app_strings（中英）+ locale_controller
│   └── theme/                  # app_theme（Fluent 风格）
├── android/app/src/main/
│   ├── aidl/.../IShizukuInputService.aidl
│   ├── kotlin/.../
│   │   ├── MainActivity.kt              # 通道注册 + 输入路由 + DisplayListener + Shizuku 权限
│   │   ├── ScreenCaptureService.kt      # 前台服务 foregroundServiceType="mediaProjection"
│   │   ├── InputAccessibilityService.kt # dispatchGesture / 全局动作 / 文本注入
│   │   ├── ShizukuInputInjector.kt      # 增强档管理（权限/绑定/手势状态机）
│   │   ├── ShizukuInputUserService.kt   # shell 进程内 injectInputEvent 反射注入
│   │   └── UsbKeycodeMap.kt             # USB HID → Android 键码
│   └── res/xml/{accessibility_service_config,network_security_config}.xml
├── test/protocol_test.dart     # SPAKE2 / protobuf / Jingle 单元测试
└── pubspec.yaml                # flutter_webrtc、crypto、xml、http、shared_preferences、
                                #   flutter_secure_storage、file_picker、path_provider、wakelock_plus
```

### 4.1 主控端数据流（M1 先行）

```
Flutter UI（触控/键盘）
  → protocol/client_session.dart
      1. REST: access-code:verify → signal_token
      2. WS /v1/realtime/signal 首帧 auth (role:client)
      3. Jingle session-initiate ↔ accept，SPAKE2 (Alice) 认证
      4. RTCPeerConnection：ontrack 收视频 → RTCVideoRenderer 渲染
      5. DataChannel 'event'：MouseEvent/KeyEvent/TouchEvent protobuf
      6. DataChannel 'control'：剪贴板、VideoLayout、capabilities
```

触控交互对齐 WebClient `touch-handler.js` 与 RustDesk 移动端经验：单指=移动+点按、双指=滚动/缩放、长按=右键、浮动鼠标模式可选；文本输入走系统软键盘 + KeyEvent/文本注入消息。

### 4.2 被控端数据流（M3/M4，已实现）

```
先决：无障碍已开启 → 设备 provision/心跳/访问码 → 先起前台服务再 getDisplayMedia 采集 → host signal_token → WS role:host
对端 session-initiate
  → host_session.dart（Jingle responder + SPAKE2 Bob，两次协商见 §5.2）
      基础连接(answer#1) → SPAKE2 → 加视频轨+control 通道 → 签名 offer#2 → answer#2
  → DataChannel 'event' 收 protobuf 输入事件，归一化为 HostInputEvent → 注入路由：
      → 优先 Shizuku 增强档：ShizukuInputUserService.injectInputEvent 真实 MotionEvent/KeyEvent
      → 回退无障碍标准档：dispatchGesture（点按/拖拽/长按/滚动）、全局动作（返回/主页/最近任务）、
        ACTION_SET_TEXT 文本注入
  → control channel 下发：capabilities、VideoLayout（单屏，含真实分辨率）
```

### 4.3 与 MCP 的关系（M5）

被控端跑通后，现有 AI 链路自动获益：AI Agent → quickdesk-mcp → 桌面 QuickDesk（主控）→ Android 被控设备。截图/点击/输入等 MCP 工具无需感知被控端是手机；仅坐标系（竖屏分辨率、旋转）与键盘语义需在工具描述中补充说明。

## 5. 关键技术点与对策

### 5.1 SPAKE2 的 Dart 实现（已验证 ✅）

`spake2.js` 依赖 Ed25519 点运算（非 RFC 9382，M/N 点来自 BoringSSL）。已完成纯 Dart 移植（`edwards25519.dart` 点运算 + `spake2.dart` + `spake2_authenticator.dart`，仅依赖 `package:crypto`，无 FFI），与 WebClient JS 金标准**逐字节兼容**：

- 曾用独立对拍工具（Dart↔JS 双向角色 + 负向用例）验证 auth key 一致、verification hash 双向互验通过；结论沉淀后对拍工具已移除
- 现由 `test/protocol_test.dart` 覆盖 Alice/Bob 握手、错误访问码拒绝等回归
- Alice 与 **Bob（Host 角色）两侧均已实现**，被控端（M3）直接复用

剩余注意点：Dart 实现为 BigInt 非常量时间，会话级一次性随机数场景可接受；真机对 Chromium Host 的最终确认在联调时完成。

### 5.2 Host 角色 Jingle responder（两次协商）

WebClient 只有 initiator。`host_session.dart` 实现 responder，采用与 Chromium Host 一致的**两次协商**流程（对齐 Qt/Web 各端，不依赖客户端 rollback）：

1. **第一次协商（基础连接）**：收 `session-initiate`(offer#1) → `setRemote` → `createAnswer`（此时无视频轨，媒体 inactive）→ 回 `session-accept`(answer#1 + method + SPAKE2 消息)。answer#1 不签名（auth_key 未就绪，客户端不校验 session-accept 签名）
2. `session-info` 往返完成 SPAKE2（client 发 spake+hash，Host 回 hash）
3. **第二次协商（视频）**：认证后加屏幕视频轨 + 建 `control` 通道 → `createOffer`(offer#2) → 用 auth_key 对 SDP 签名 → `transport-info(offer)`；client 回签名 answer#2，Host 校验签名后 `setRemote`
4. 双向 `transport-info` 交换 ICE candidate（认证前缓冲，认证后 flush）；client 建 `event` 通道，Host 收输入事件转注入层

多客户端按信令 `client_id` 区分，各自独立 sid/PeerConnection/SPAKE2，共享同一路屏幕采集流。JID 格式 `<device_id>@quickdesk.local/chromoting_ftl_*`（remote JID 取 session-initiate 的 initiator）。信令服务器只做转发，无需改动。

> 信令帧封装：与 Chromium/WebClient 一致，收发都用 `{client_id, payload:"<jingle xml>"}` JSON 信封（`websocket_transport.dart` 处理封包/解包，host 角色据 client_id 路由）。

### 5.3 Android 14+ MediaProjection 合规链路

系统硬性要求（缺一必崩）：

1. Manifest 声明 `FOREGROUND_SERVICE_MEDIA_PROJECTION` 权限 + service `foregroundServiceType="mediaProjection"`；Android 13+ 另需运行时请求 `POST_NOTIFICATIONS`
2. 顺序：**先** `ScreenCaptureService.start()` 拉起前台服务 → **再** `getDisplayMedia`（flutter_webrtc 内部请求 MediaProjection 授权并建采集）
3. 授权 token 一次性，**每个新会话都要用户点一次授权**
4. Android 15 锁屏会终止投影，需提示用户并自动重协商（待真机验证）

flutter_webrtc 的 `getDisplayMedia` 不自带前台服务，`ScreenCaptureService.kt` 自研并在 Dart 调用前经 `quickdesk/screen_capture` 通道启动。**采集尺寸兜底**：`getSettings()` 取不到宽高时，回退到原生 `WindowManager` 真实屏幕尺寸（`getScreenSize` 通道），避免下发 0×0 的 VideoLayout 导致主控端坐标映射失效。

### 5.4 输入注入两档

| 档位 | 能力 | 限制 |
|------|------|------|
| 无障碍（默认） | 点按/滑动/长按/多指手势（`dispatchGesture`）、全局动作（返回/主页/最近任务）、文本注入（IME） | 无法注入任意 KeyEvent 到其他 App；部分厂商 ROM 需手动保活无障碍服务 |
| Shizuku（增强） | `input`/uinput 级真实事件注入，完整按键、组合键、游戏触控 | 用户需 ADB 激活一次 Shizuku（无线调试或电脑），重启后可能需重新激活 |

运行时探测：Shizuku 可用且已授权 → 增强档；否则回退无障碍档，并在 UI 明示当前档位与差异。

### 5.5 视频与旋转

- 编码/拥塞控制交给 libwebrtc screencast 模式（降分辨率优先于降帧）
- 竖屏/横屏切换：`MainActivity` 注册 `DisplayManager.DisplayListener`，默认屏幕变化时取真实尺寸经通道通知 Dart → `HostSession.updateScreenSize` 向所有在线客户端**重发 VideoLayout**。画面本身由 flutter_webrtc 的 `OrientationAwareScreenCapturer` 自动跟随旋转，故只需同步 VideoLayout 宽高纠正主控端坐标映射，**无需 WebRTC 重协商**
- 桌面 Qt 客户端 / WebClient 连 Android 被控时为单流单屏，`VideoLayout` 的 `mediaStreamId` 取采集流真实 id（与 SDP msid 对齐），置单显示器即兼容现有多显示器逻辑

### 5.6 保活与厂商适配

会话中 `WakelockPlus` 保持屏幕常亮；被控前台服务常驻。电池优化白名单、厂商（MIUI/HarmonyOS/ColorOS）自启动引导为后续项。

### 5.7 构建与安全配置（本期落地）

- **compileSdk 36 / minSdk 24**：`flutter_plugin_android_lifecycle` 等插件要求 compileSdk ≥ 36；无障碍手势与事件注入要求 API 24+。根 `build.gradle.kts` 用 `subprojects` 统一把插件模块 compileSdk 提到 36
- **JDK 17**：GraalVM/JDK 21 与 AGP 的 jlink 变换不兼容，需 `flutter config --jdk-dir` 指向 JDK 17
- **明文流量**：信令服务器地址由用户填写，可能是自建明文（ws/http）或局域网 IP，故 `network_security_config.xml` 开启 `cleartextTrafficPermitted`；TLS（wss/https）部署不受影响
- **device_secret 加密存储**：被控端 provision 拿到的 device_secret 存入 `flutter_secure_storage`（Android Keystore 加密）；device_id/access_code 等低敏感项走 shared_preferences
- **Shizuku 增强档**：`dev.rikka.shizuku:api/provider` + 自定义 AIDL UserService，运行在 shell 进程用反射调 `InputManager/InputManagerGlobal.injectInputEvent` 注入真实事件；API 34 前后自动切换注入通道

## 6. 里程碑划分

| 里程碑 | 内容 | 验收标准 |
|--------|------|----------|
| **M1 主控端跑通**（代码完成，待真机联调） | Flutter 工程骨架；REST + 信令 WS + Jingle initiator + SPAKE2 Alice 的 Dart 移植；flutter_webrtc 收流渲染；event channel 触控/键盘 | 手机成功控制 Windows/macOS 被控端：看到画面、点击、打字 |
| **M2 主控端完善**（代码完成，待真机联调） | 剪贴板同步、多显示器切换、连接历史/收藏（当前本地存储，登录后可迁移 `/v1/me/*`）、性能统计面板、文件传输、中英 i18n、Fluent 风格 UI | 功能对齐 WebClient 现状 |
| **M3 被控端镜像**（代码完成，待真机联调） | Host 角色协议（responder + SPAKE2 Bob）；ScreenCaptureService + getDisplayMedia 发流；设备注册/心跳/访问码 | 桌面端输入手机设备 ID + 访问码可看到手机画面 |
| **M4 被控端输入**（代码完成，待真机联调） | 无障碍服务注入（手势/全局动作/IME 文本）；Shizuku 增强档（UserService 注入真实事件）；自动选后端 + 权限引导 | 桌面端可完整操作手机；两档位可切换 |
| **M5 AI 场景打磨** | MCP 工具对 Android 被控的适配说明（坐标/旋转/软键盘语义）、典型场景 Demo 文档 | AI Agent 经桌面 QuickDesk 自动化操作手机 |

## 7. 风险清单

代码已实现，以下为**待真机联调验证**的风险（已消除项标注）：

| 风险 | 等级 | 对策 / 现状 |
|------|------|------|
| ~~SPAKE2 Dart 移植与 BoringSSL 不互通~~ | ~~高~~ | **已消除**：对拍全过 + 单测覆盖，详见 §5.1 |
| ~~被控端 SDP 协商仅对 WebClient 的 rollback 路径有效，Qt 端可能不兼容~~ | ~~高~~ | **已消除**：改为标准两次协商（§5.2），Qt/Web/自家客户端都走标准 answer→重协商路径 |
| ~~被控端采集尺寸缺失导致 VideoLayout 为 0×0、主控端无法映射坐标~~ | ~~中~~ | **已消除**：回退真实屏幕尺寸 + 主控端帧尺寸兜底（§5.3） |
| ~~旋转后 VideoLayout 未同步，坐标漂移~~ | ~~中~~ | **已消除**：DisplayListener 重发 VideoLayout（§5.5） |
| MediaProjection 前台服务时序（Android 14）/ 锁屏终止投影 | 中 | 已按"先起前台服务再 getDisplayMedia"实现；真机验证时序与锁屏重协商 |
| Shizuku `injectInputEvent` 反射在不同 ROM/版本的兼容性 | 中 | 已按 scrcpy 路线 + API34 前后双通道；需真机矩阵验证 |
| 多客户端共享同一采集 track 挂到多个 PeerConnection | 中 | libwebrtc 一般支持；未测，多控场景需验证 |
| 厂商 ROM 杀后台/无障碍失活 | 中 | 前台服务 + WakelockPlus；白名单引导为后续项 |
| MediaProjection 每会话授权打断无人值守场景 | 低 | 系统限制，如实告知；Shizuku 档位未来可探索 `screenrecord` 替代 |

## 8. 不做的事（本期范围外）

- iOS 端（架构预留，Flutter 代码可复用，但 iOS 无被控可能性且主控需另行适配）
- Android 被控多屏/虚拟屏
- 音频采集转发（Android 10+ 才支持 AudioPlaybackCapture，且需 App 逐个允许，收益低）
- 手机端跑 MCP Server / skill-host
