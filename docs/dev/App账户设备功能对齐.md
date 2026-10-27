# App 账户与设备功能对齐

基线：`dev` / `c4c59ed592644b36d6bcdacd57ed82f1711bf25f`。本次仅扩展产品层，不修改服务端账户、归属、权限或 Remoting 协议。

## 源码依据与范围

- Qt：`QuickDesk/src/manager/AuthManager.cpp`、`CloudDeviceManager.cpp`、`QuickDesk/qml/quickdeskcomponent/LoginDialog.qml`、`QuickDesk/qml/pages/DeviceListPage.qml`。
- 服务端：`SignalingServer/docs/user-api-docs.md` 以及对应 handler；接口文档不是新增语义授权。
- 原 App：`quickdesk-android/lib/controller/home_page.dart` 只有主控/被控；`api/signaling_api.dart` 只提供访问码校验和 ICE；账户层缺失。
- Web：`WebClient/src/views/DevicesPage.vue`、`RecordingsPage.vue`、`ResetPasswordPage.vue`。

## 对齐矩阵

| 能力 | Qt / 后端现状 | 原 App | 本次目标 / 边界 |
|---|---|---|---|
| 用户名/手机/邮箱 + 密码登录 | 统一 identifier 接口 | 无 | 新增账户入口、失败重试、输入校验 |
| 注册并自动登录 | register 返回用户及令牌 | 无 | 按服务端 feature 显示注册/SMS，不改变注册政策 |
| SMS 登录与验证码 | sms_enabled 控制入口 | 无 | 按服务端开关显示，不发真实短信测试 |
| 会话恢复/轮转 | access + refresh；401 刷新后仅重试一次 | 无 | 安全存储、按服务器隔离、合并刷新、拒绝失效会话 |
| 临时网络故障 | 保留可恢复 session | 无 | 不把超时/5xx 当永久失效 |
| 退出/服务端撤销 | 当前 session 撤销；显式退出可清设备意图 | 无 | 清本地账户及缓存；不猜测未隔离的 Host 凭据所属服务器 |
| 我的设备 | 在线/登录状态、备注、解绑、连接 | 无 | 列表、刷新、动作确认及现有访问码连接路径 |
| 我的收藏 | 增删改、保存名称/访问密码、连接 | 仅本地历史 | 云端收藏，不迁移/覆盖原本地历史 |
| 实时同步 | 首帧认证、revision、snapshot、撤销事件 | 无 | 有序应用及断线恢复，令牌不放 URL |
| 列表排序 | 三端缺一致规则 | 无账户列表 | 三端共享规则及黄金测试夹具，见下文 |
| 设备绑定/归属转移 | POST /v1/me/devices 可转移设备归属 | Host 凭据为全局键，无服务器来源 | **不自动绑定**。需先做 Host 凭据来源隔离和明确归属确认 |
| 云端连接历史 | Qt 查询及记录；后端分页 | 本地历史 | 新增云端历史查询（显示最新 20 条）；连接完成上报仍待独立补齐，不伪造连接完成记录 |
| 忘记密码 | Qt 跳转 preset 指定 WebClient 重置页 | 无 | 后续独立补齐；不猜测密码重置站点 |
| 个人资料/会话管理 | 后端有修改资料、改密、会话列表/踢出接口 | 无 | 本次不扩大至账户管理中心 |

## 其他尚未对齐的相邻能力

1. **Host 账户关联**：先把 `HostCredentialStore` 的设备 ID/secret 与服务器来源绑定，再设计明确的绑定/切换账号确认流程；否则同名 ID 在不同服务器可能错误转移归属。
2. **密码重置、资料修改与活跃会话管理**：使用既有 API；改密会撤销会话，须单独覆盖中断/重新登录流程。
3. **录像索引/播放入口**：Web 已有 `RecordingsPage.vue`；App 没有。索引 API 不等于移动端已具备录像引擎。
4. **可信设备/审批策略**：Qt 设置有 trust confirm mode，App 现有 `SignalingApi` 仅访问码换票；不在本次引入持久信任或默认同意。
5. **多显示器/虚拟屏/隐私屏等高级远控选项**：Qt 已有产品入口，App 远控工具条尚未覆盖。与本次账户层无关，不改 Remoting。
6. **已有功能不重复重写**：App 已有剪贴板、文件传输、主控/被控、移动触控交互；保留原连接历史与全部已合并移动控制逻辑。

## 三端排序契约

- 我的设备：`online == true` 优先，其后按显示名称（非空 `remark` → `device_name` → `device_id`）自然排序，最后设备 ID 确定顺序。
- 我的收藏：名称自然排序，再按设备 ID。后端收藏行没有共同的在线/最近连接字段，不虚构状态。
- ASCII 大小写不敏感；数字片段自然比较（2 在 10 前）；非 ASCII 使用确定性 UTF-16 顺序，不依赖系统语言/区域。具体空白、前导零、同名行为以共享夹具为准。
- 不使用 `last_seen_at` 冒充最近连接时间，不使用当前时钟参与排序；输入数据不变则刷新不抖动。
- 排序只作用于展示副本，不改变实时状态的原始顺序/修订号、不写回服务端。

## 安全与验收边界

- 测试只用 mock/fixture；不创建真实账户、不发真实验证码、不建立真实设备归属或信任授权。
- 令牌不写日志、不写 SharedPreferences、不跨服务器重用；不放宽 TLS 校验。对沿用的明文服务器配置显示风险提示。
- 本地 Flutter bootstrap 和浏览器运行受环境限制，不能把代码审阅/格式化宣称为真机验证。最终 lint/test/APK 构建以 PR 指定 SHA 的 CI 为准。
- Android 真机需另验安全存储、输入法/大字/横屏、网络切换、后台恢复、扫码/Host 互通。CI 通过不替代这些检查。
