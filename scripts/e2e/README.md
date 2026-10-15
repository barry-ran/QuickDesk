# 端到端联调脚本（自托管信令 + 真实被控端）

这套脚本用来在没有构建本仓库 Windows 产物的情况下，验证
「WebClient → 自托管 SignalingServer → Windows 被控端」的完整链路：
画面、鼠标、键盘三条路都要拿到证据，不允许只看截图就下结论。

## 组成

| 文件 | 作用 |
|---|---|
| `remote-smoke.mjs` | 用真实 Chrome 打开 WebClient，连接设备，断言解码后的视频帧在推进；可选向远端打字 |
| `record.mjs` | 连上之后点录制按钮，录一段再停止，断言上传日志出现（录 → 存 → 索引） |
| `record-failure.mjs` | 录制时拦截上传请求制造失败，断言索引行最终是 `failed` 且没有产物（不允许"假完成"） |
| `record-retry.mjs` | 中断前两次产物上传与第一次预览上传，断言退避重试后仍然落地，并且最终行是 `completed` **且带 `thumbnail_url`** |
| `record-concurrent.mjs` | 同时开 N 个会话录同一个设备（`QD_SESSIONS`，默认 2），断言每路都独立落库为 `completed` 且各自有产物与预览图 |
| `playback.mjs` | 在录像页点「播放」，断言签名 URL 打开的播放器 `currentTime` 在推进 |
| `guest-input-target.ps1` | 被控端侧的单窗口输入靶子（WinForms 文本框，内容变化即落盘），比 Win11 记事本可靠 |
| `guest-qt-ui.ps1` | 用 UI Automation 驱动 Windows 上的 QuickDesk 桌面客户端（填设备号/接入码、点连接、发快捷键），用于验证 Qt 侧录制 |
| `guest-notepad.ps1` | 备选靶子：用 UI Automation 读写记事本，适合快速人工核对 |

## 依赖

- Node 18+，`npm i playwright`（用本机已安装的 Chrome，`channel: 'chrome'`）
- 一个跑起来的 SignalingServer，以及一个已经绑定到该用户的在线设备
- WebClient 构建产物用静态服务器暴露，例如：

```bash
cd WebClient && npm run build
python3 -m http.server 5173 --bind 127.0.0.1 --directory dist
```

## 跑一次

```bash
QD_TOKEN=<access_token> \
QD_DEVICE=<device_id> \
QD_SERVER=ws://10.211.55.2:8000 \
QD_APP=http://127.0.0.1:5173 \
node remote-smoke.mjs
```

可选环境变量：

- `QD_TYPE_TEXT`：连上之后向远端输入的字符串
- `QD_CLEAR_FIRST=1`：输入前先 Ctrl+A / Delete
- `QD_CLICK_FRAC=x,y`：点击远端画面的相对坐标（默认 `0.12,0.20`，避开 Windows
  首次连接时的防火墙弹窗）
- `QD_OUT`：截图与 `e2e-result.json` 的输出目录

退出码 0 表示视频帧真实到达（`videoWidth > 0` 且 `currentTime` 在推进）。

## 被控端准备（Parallels Windows 虚拟机）

```bash
# 1. 安装上游产物；SYSTEM 上下文可以免 UAC 静默安装
prlctl exec <uuid> --current-user cmd /c "copy /Y \\Mac\Home\...\QuickDesk-win-x64-*-setup.exe C:\QuickDeskTest\setup.exe"
prlctl exec <uuid> cmd /c "C:\QuickDeskTest\setup.exe /VERYSILENT /SUPPRESSMSGBOXES /NORESTART /SP-"

# 2. 把被控端指向自托管信令：写 %APPDATA%\QuickDesk\db\appconfig.db
#    signalingServerUrl=ws://<host>:8000 且 apiKey=<服务端 API_KEY>

# 3. 输入靶子
prlctl exec <uuid> --current-user powershell -File C:\QuickDeskTest\guest-input-target.ps1 -Action show
prlctl exec <uuid> --current-user powershell -File C:\QuickDeskTest\guest-input-target.ps1 -Action focus
```

实测结论与两个容易踩的坑见 `docs/dev/自托管端到端联调记录.md`。
