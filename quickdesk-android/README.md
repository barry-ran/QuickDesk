# quickdesk_android

QuickDesk Android 客户端（Flutter）：主控（控制桌面端/其它设备）与被控（被桌面端控制）双向。
代码覆盖 M1（主控跑通）、M2（主控完善）、M3（被控镜像）、M4（被控输入），待真机联调。

技术方案见 [`docs/dev/Android客户端技术方案.md`](../docs/dev/Android客户端技术方案.md)。

## 目录结构

```
lib/
├── main.dart                    # 入口（WebRTC field trial 注入 + 主题/语言）
├── core/                        # 公共工具：app_settings（服务器配置）/ rand_id / geometry
├── api/
│   ├── signaling_http.dart      # REST 公共层（ws→http、鉴权头、请求模板、ICE 解析）
│   ├── signaling_api.dart       # 主控 REST（access-code:verify / ice-config）
│   └── host_api.dart            # 被控 REST（provision / heartbeat / signal-tokens / access-code）
├── protocol/                    # 纯 Dart 协议栈（与 WebClient/js 对齐，可独立单测）
│   ├── auth/                    #   SPAKE2（edwards25519 + 认证状态机，Alice/Bob 双角色）
│   ├── proto/                   #   wire（protobuf 原语）+ Chromium Remoting 消息编解码
│   ├── signaling/               #   Jingle XML builder/parser + WS 传输（{client_id,payload} 封装）
│   ├── peer/                    #   会话公共构件：串行队列 / candidate 缓冲 / PC 工厂
│   ├── client_session.dart      #   主控端会话状态机（flutter_webrtc）
│   ├── host_session.dart        #   被控端会话状态机（Jingle responder + SPAKE2 Bob，两次协商）
│   ├── datachannel_handler.dart #   event/control DataChannel（client/host 角色共用）
│   ├── file_transfer.dart       #   文件传输（file_transfer.proto，独立 DataChannel）
│   └── host_input.dart          #   被控端输入事件模型
├── controller/                  # UI：连接页/被控页/首页 + 触控/键码/剪贴板/统计/历史
│   └── remote/                  #   远程桌面页（主页面/工具条/统计面板/传输弹层/光标）
├── host/                        # 被控端桥接：屏幕采集 / 输入注入 / 凭据 / 总控
├── l10n/                        # 轻量中英 i18n
└── theme/                       # Fluent 风格主题
```

被控端原生（Kotlin，`android/app/src/main/kotlin/...`）：屏幕采集前台服务、无障碍输入注入、
Shizuku 增强档（AIDL UserService + `injectInputEvent` 反射）、USB→Android 键码映射。

## 开发环境

- Flutter SDK stable 3.44+
- **JDK 17**：GraalVM / JDK 21 与 AGP 的 jlink 变换不兼容，需 `flutter config --jdk-dir "<jdk17>"` 指定
- Android SDK：platform-tools + platforms;android-36 + build-tools;36.0.0（NDK 由 Gradle 自动安装）
- 已内置 `compileSdk 36` / `minSdk 24`（插件依赖与被控输入 API 要求）

常用命令（在本目录执行）：

```bash
flutter pub get
flutter analyze            # 静态分析
flutter test               # 协议层单元测试（SPAKE2 / protobuf / Jingle）
flutter build apk --debug  # 构建 APK（产物在 build/app/outputs/flutter-apk/）
```

官方信令服务器 API Key 可在构建时注入，未注入时默认为空：

```bash
flutter build apk --release --dart-define=QUICKDESK_API_KEY=your-secret-key
```

> 注意：本工程不要放在网络映射盘（SMB/UNC 路径）上构建，Gradle/ninja/Kotlin
> 增量编译均无法在网络盘上正常工作。请将仓库克隆/复制到本地磁盘再构建。

## 使用

首页底部切换「主控 / 被控」：

- **主控**：填对端「设备 ID + 访问码」连接；服务器地址在「服务器设置」里配置。
  若服务器开启了 API-Key/Origin 保护，需在「API Key」填入服务器的 `QUICKDESK_API_KEY`
  （否则 REST 校验返回 FORBIDDEN）。明文 `ws://` 服务器已支持。
- **被控**：点「开启被控」，依次引导开启无障碍服务、授权屏幕采集，随后展示本机
  设备 ID + 访问码供其他端连入。输入默认走无障碍档；装并授权 Shizuku 后自动切增强档。

## 真机联调关注点（尚未验证）

- 被控端与 Qt 桌面端 / WebClient 的两次协商互通
- Android 14 MediaProjection 前台服务时序、锁屏终止投影
- Shizuku `injectInputEvent` 在不同 ROM/版本的兼容性
- 旋转后主控端坐标对齐

## 账户、设备与收藏

首页新增「账户」页：沿用服务器设置，支持密码/SMS 登录、注册、会话恢复、
我的设备、云端收藏及云端历史查询。修改服务器配置后需点「保存」；账户令牌
使用安全存储并按服务器配置隔离，不会转移到另一台服务器。

请优先配置 `wss://` / `https://`；明文服务器会显示安全提示。账户登录不等于
绑定本机：旧 Host 凭据缺少服务器来源信息，本次不会自动绑定或转移设备归属。
退出时本地账户缓存立即清空；网络不可达时会提示服务端撤销未确认。

三端设备/收藏排序规则及尚未对齐项见
[App 账户设备功能对齐](../docs/dev/App账户设备功能对齐.md) 和
[排序契约](../docs/dev/Device-list-ordering.md)。
