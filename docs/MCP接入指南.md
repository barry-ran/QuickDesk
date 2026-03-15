# QuickDesk MCP 接入指南

QuickDesk 内置 [MCP (Model Context Protocol)](https://modelcontextprotocol.io/) Server，让 AI Agent 能够以编程方式查看和控制远程桌面。

## 架构

QuickDesk MCP 支持两种传输模式：

### stdio 模式（默认）

```
AI Agent (Claude / Cursor / GPT)
    │  stdio (JSON-RPC 2.0)
    ▼
quickdesk-mcp (Rust 桥接程序)          ← 由 AI 客户端启动
    │  WebSocket
    ▼
QuickDesk GUI (Qt 6)
    │  Native Messaging + 共享内存
    ▼
远程桌面 (Chromium Remoting / WebRTC)
```

AI 客户端将 `quickdesk-mcp` 作为子进程启动，通过 stdin/stdout 通信。最简单的配置方式——粘贴一段 JSON 即可。

### HTTP/SSE 模式

```
AI Agent (Claude / Cursor / GPT)
    │  HTTP (Streamable HTTP / SSE)
    ▼
quickdesk-mcp (Rust HTTP 服务器)       ← 由 QuickDesk 启动和管理
    │  WebSocket
    ▼
QuickDesk GUI (Qt 6)
    │  Native Messaging + 共享内存
    ▼
远程桌面 (Chromium Remoting / WebRTC)
```

QuickDesk 启动并管理 `quickdesk-mcp` HTTP 服务器进程。AI 客户端通过 `http://127.0.0.1:18080/mcp` 连接。此模式支持多个 AI 客户端同时连接，也支持网络远程访问。

## 快速开始

### 1. 启动 QuickDesk

正常启动 QuickDesk，WebSocket API 服务器会自动在 `ws://127.0.0.1:9600` 上运行。

### 2. 配置 AI 客户端

#### 方式 A：stdio 模式（推荐单客户端使用）

##### Cursor IDE

在项目根目录创建或编辑 `.cursor/mcp.json`：

```json
{
  "mcpServers": {
    "quickdesk": {
      "command": "/path/to/quickdesk-mcp",
      "args": [],
      "env": {}
    }
  }
}
```

##### Claude Desktop

编辑 `claude_desktop_config.json`：

- **Windows**：`%APPDATA%\Claude\claude_desktop_config.json`
- **macOS**：`~/Library/Application Support/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "quickdesk": {
      "command": "C:\\path\\to\\quickdesk-mcp.exe",
      "args": []
    }
  }
}
```

##### VS Code

在项目根目录创建或编辑 `.vscode/mcp.json`：

```json
{
  "servers": {
    "quickdesk": {
      "type": "stdio",
      "command": "/path/to/quickdesk-mcp",
      "args": []
    }
  }
}
```

#### 方式 B：HTTP/SSE 模式（支持多客户端同时连接）

在 QuickDesk 中打开 MCP 设置 → 切换到 **HTTP/SSE** 模式 → 打开 MCP HTTP 服务开关。QuickDesk 会自动启动 `quickdesk-mcp` HTTP 服务器。

##### Cursor IDE

```json
{
  "mcpServers": {
    "quickdesk": {
      "type": "sse",
      "url": "http://127.0.0.1:18080/mcp"
    }
  }
}
```

##### Claude Desktop

```json
{
  "mcpServers": {
    "quickdesk": {
      "type": "sse",
      "url": "http://127.0.0.1:18080/mcp"
    }
  }
}
```

##### VS Code

```json
{
  "servers": {
    "quickdesk": {
      "type": "http",
      "url": "http://127.0.0.1:18080/mcp"
    }
  }
}
```

> **注意：** VS Code 使用 `"type": "http"` 和顶层键 `"servers"`。其他客户端（Cursor、Claude Desktop、Windsurf）使用 `"type": "sse"` 和 `"mcpServers"`。

#### 通用 MCP 客户端（stdio）

`quickdesk-mcp` 是标准的 MCP stdio 服务器，任何支持 `stdio` 传输的客户端都可以使用：

```bash
quickdesk-mcp [--ws-url ws://127.0.0.1:9600] [--token YOUR_TOKEN]
```

#### 命令行参数

| 参数 | 默认值 | 环境变量 | 说明 |
|------|--------|----------|------|
| `--ws-url` | `ws://127.0.0.1:9600` | `QUICKDESK_WS_URL` | QuickDesk WebSocket API 地址 |
| `--token` | （空） | `QUICKDESK_TOKEN` | 完全控制权限的认证 Token |
| `--readonly-token` | （空） | `QUICKDESK_READONLY_TOKEN` | 只读权限的认证 Token |
| `--allowed-devices` | （空） | `QUICKDESK_ALLOWED_DEVICES` | 允许连接的设备 ID 白名单（逗号分隔） |
| `--rate-limit` | `0` | `QUICKDESK_RATE_LIMIT` | 每分钟最大 API 请求数（0 = 不限制） |
| `--session-timeout` | `0` | `QUICKDESK_SESSION_TIMEOUT` | 会话空闲超时（秒，0 = 不超时） |
| `--transport` | `stdio` | `QUICKDESK_TRANSPORT` | 传输模式：`stdio` 或 `http` |
| `--port` | `8080` | `QUICKDESK_HTTP_PORT` | HTTP 服务器端口（仅 HTTP 模式） |
| `--host` | `127.0.0.1` | `QUICKDESK_HTTP_HOST` | HTTP 服务器绑定地址（仅 HTTP 模式） |
| `--cors-origin` | （空） | `QUICKDESK_CORS_ORIGIN` | 允许的 CORS 来源（仅 HTTP 模式） |
| `--stateless` | `false` | — | 使用无状态 HTTP 会话（仅 HTTP 模式） |

### 3. 开始使用

配置完成后，AI Agent 可以直接使用 QuickDesk 工具。示例对话：

> **你**：连接我的远程服务器（设备 ID：123456789，访问码：888888），看看屏幕上有什么。
>
> **AI Agent**：*（调用 `connect_device` → `screenshot` → 分析图像）* 我看到 Windows 桌面，文件管理器正在打开...

## MCP 工具参考

### 连接管理

| 工具 | 说明 |
|------|------|
| `connect_device` | 通过设备 ID + 访问码连接远程设备。返回 `connection_id`。设置 `show_window=false` 可进行后台无界面自动化。 |
| `disconnect_device` | 断开指定远程连接。 |
| `disconnect_all` | 断开所有活跃连接。 |
| `list_connections` | 列出所有活跃连接及其 ID 和状态。 |
| `get_connection_info` | 获取指定连接的详细信息。 |

### 视觉与屏幕

| 工具 | 说明 |
|------|------|
| `screenshot` | 截取远程屏幕。返回 base64 图像。使用 `max_width` 缩小图片以加速传输和 AI 处理。 |
| `get_screen_size` | 获取远程桌面分辨率（宽 × 高）。 |
| `get_screen_text` | 对当前远程桌面帧执行 OCR（PP-OCRv4）。返回所有识别到的文字块，含边界框、中心坐标和置信度。结果按帧哈希缓存——对同一帧多次调用无额外开销。 |
| `find_element` | 通过可见文字查找 UI 元素。返回所有匹配的文字块及其边界框和中心坐标。支持部分匹配（默认）和精确匹配。 |
| `click_text` | 在远程桌面上查找文字并一步点击。等价于 `find_element` + 在文字中心执行 `mouse_click`。若有多个匹配，点击第一个。 |

#### `get_screen_text`

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `connection_id` | string | ✅ | — | 远程桌面的连接 ID |

**返回：** `{ blocks, frameHash, width, height, connectionId }`

`blocks` 中每个元素的字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `text` | string | 识别到的文字内容 |
| `bbox` | object | 边界框 `{ x, y, w, h }`（像素） |
| `center` | object | 中心点 `{ x, y }`，可直接用于点击坐标 |
| `confidence` | number | OCR 置信度（0–1） |

#### `find_element`

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `connection_id` | string | ✅ | — | 远程桌面的连接 ID |
| `text` | string | ✅ | — | 要在屏幕上查找的文字 |
| `exact` | boolean | — | `false` | 为 true 时要求精确匹配；false 为部分/子串匹配 |
| `ignore_case` | boolean | — | `true` | 是否忽略大小写 |

**返回：** `{ found, matches, query, frameHash, connectionId }`

#### `click_text`

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `connection_id` | string | ✅ | — | 远程桌面的连接 ID |
| `text` | string | ✅ | — | 要查找并点击的文字 |
| `exact` | boolean | — | `false` | 精确匹配还是部分匹配 |
| `ignore_case` | boolean | — | `true` | 是否忽略大小写 |
| `button` | string | — | `"left"` | 鼠标按钮：`"left"`、`"right"` 或 `"middle"` |

**返回：** `{ success, clickedText, x, y, confidence }`，文字未找到时返回 `{ success: false, error }`。

### 基于 OCR 的屏幕理解

相比截图后交给视觉模型分析，这些工具在本地直接对原始视频帧执行 OCR（PP-OCRv4）——无 JPEG 压缩损耗、响应即时，且按帧哈希自动缓存。

#### OCR 工具 vs `screenshot` 的选择

| 场景 | 推荐方式 |
|------|----------|
| 读取菜单项、按钮标签、对话框文字 | `get_screen_text` 或 `find_element` |
| 通过标签点击按钮 | `click_text` |
| 理解整体 UI 布局 | `screenshot` → 视觉模型 |
| 在屏幕上查找特定词语 | `find_element` |
| 验证操作后文字是否出现 | `get_screen_text`（缓存，极快） |

#### 使用模式：通过标签点击按钮

```
click_text(connection_id=conn_id, text="确定")
         → 在屏幕上找到"确定"按钮并点击
```

#### 使用模式：读取文字后执行操作

```
get_screen_text(connection_id=conn_id)
    → 返回所有文字块及坐标

find_element(connection_id=conn_id, text="错误", ignore_case=true)
    → 检查是否有错误提示

click_text(connection_id=conn_id, text="重试")
    → 点击重试按钮
```

#### 使用模式：验证文字是否出现

```
keyboard_hotkey(keys=["ctrl", "s"])            → 保存文件
// 稍等片刻，然后验证
elements = find_element(connection_id=conn_id, text="已保存")
if elements.found:
    → 文件保存成功
```

| 工具 | 说明 |
|------|------|
| `mouse_click` | 在 (x, y) 处点击。支持 left/right/middle 按钮。 |
| `mouse_double_click` | 在 (x, y) 处双击。 |
| `mouse_move` | 将光标移动到 (x, y)。 |
| `mouse_scroll` | 在 (x, y) 处滚动，通过 delta_x/delta_y 控制方向。 |
| `mouse_drag` | 从 (start_x, start_y) 拖拽到 (end_x, end_y)。 |

### 键盘

| 工具 | 说明 |
|------|------|
| `keyboard_type` | 输入文本（内部使用剪贴板粘贴，完美支持 Unicode）。 |
| `keyboard_hotkey` | 按下组合键，例如 `["ctrl", "c"]`、`["win", "r"]`、`["alt", "f4"]`。 |
| `key_press` | 按住某个键不放（用于 Modifier+点击等场景）。 |
| `key_release` | 释放之前按住的键。 |

### 剪贴板

| 工具 | 说明 |
|------|------|
| `get_clipboard` | 获取远程最近同步的剪贴板内容。 |
| `set_clipboard` | 设置远程剪贴板内容。 |

### 主机与系统

| 工具 | 说明 |
|------|------|
| `get_host_info` | 获取本机设备 ID、访问码和信令状态。用于连接当前电脑。 |
| `get_host_clients` | 列出连接到本机的客户端。 |
| `get_status` | 系统总体状态（Host/Client 进程、信令服务器）。 |
| `get_signaling_status` | 信令服务器连接状态。 |
| `refresh_access_code` | 刷新本机访问码。 |

### 事件（响应式自动化）

相比轮询截图，事件工具让 AI Agent 高效等待远程桌面的状态变化。

| 工具 | 说明 |
|------|------|
| `wait_for_event` | 等待特定事件发生。返回匹配的事件数据，超时则返回错误。 |
| `wait_for_connection_state` | 等待连接达到目标状态（如 `connected`、`disconnected`）。 |
| `wait_for_clipboard_change` | 等待远程剪贴板内容变化，返回新内容。 |
| `get_recent_events` | 获取事件环形缓冲区中的近期事件，可按类型过滤。 |
| `list_event_types` | 列出所有支持的事件类型及其数据字段。 |

#### `wait_for_event`

通用事件等待器。可等待任意事件类型，支持数据字段过滤。

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `event` | string | ✅ | — | 等待的事件类型（见下方事件类型参考） |
| `filter` | object | — | — | JSON 对象，每个键值对必须与事件数据匹配。例如 `{"state": "connected"}` 仅匹配 `data.state == "connected"` 的事件。 |
| `timeout_ms` | integer | — | `30000` | 最大等待时间（毫秒） |

**返回：** 匹配的事件对象 `{event, data, timestamp}`，或超时错误字符串。

#### `wait_for_connection_state`

连接状态等待的便捷封装。在 `connect_device` 后使用，等待连接完全建立，无需轮询截图。

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `connection_id` | string | ✅ | — | 要监控的连接 ID |
| `state` | string | ✅ | — | 目标状态：`connected`、`disconnected`、`failed` |
| `timeout_ms` | integer | — | `30000` | 最大等待时间（毫秒） |

**返回：** `connectionStateChanged` 事件数据，或超时错误。

#### `wait_for_clipboard_change`

等待远程剪贴板变化。在远程桌面按下 Ctrl+C 后使用，无需轮询即可获取复制的文本。

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `connection_id` | string | ✅ | — | 要监控的连接 ID |
| `timeout_ms` | integer | — | `10000` | 最大等待时间（毫秒） |

**返回：** `clipboardChanged` 事件（含新剪贴板文本），或超时错误。

#### `get_recent_events`

查询事件环形缓冲区中的近期事件。适用于检查 AI 执行其他操作期间发生了什么。

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `event_type` | string | — | — | 按事件类型过滤。为空则返回所有类型。 |
| `limit` | integer | — | `20` | 最大返回数量（上限 100） |

**返回：** 按时间顺序排列的事件数组（最早的在前）。

#### `list_event_types`

返回所有支持的事件类型及其数据字段。无参数。

### 事件类型参考

| 事件类型 | 说明 | 数据字段 |
|----------|------|----------|
| `connectionStateChanged` | 连接状态变化 | `connectionId`、`state`、`hostInfo` |
| `clipboardChanged` | 远程剪贴板内容变化 | `connectionId`、`text` |
| `connectionAdded` | 新建出站连接 | `connectionId`、`deviceId` |
| `connectionRemoved` | 出站连接已移除 | `connectionId` |
| `videoLayoutChanged` | 远程桌面视频分辨率变化 | `connectionId`、`width`、`height` |
| `hostReady` | 本机主机服务就绪，可接受连接 | `deviceId`、`accessCode` |
| `accessCodeChanged` | 本机主机接入码已刷新 | `accessCode` |
| `hostClientConnected` | 远程客户端连入本机 | `connectionId`、... |
| `hostClientDisconnected` | 远程客户端从本机断开 | `connectionId`、`reason` |
| `hostSignalingStateChanged` | 主机信令连接状态变化 | `state`、`retryCount`、`nextRetryIn`、`error` |
| `hostProcessStatusChanged` | 主机进程状态变化 | `status` |
| `clientProcessStatusChanged` | 客户端进程状态变化 | `status` |

## MCP Resources（资源）

Resources 提供只读的实时系统状态数据。

| URI | 说明 |
|-----|------|
| `quickdesk://host` | 本机设备 ID、访问码、信令状态 |
| `quickdesk://status` | 系统总体状态 |
| `quickdesk://connection/{connectionId}` | 指定连接的详细信息 |

## MCP Prompts（提示词模板）

Prompts 是内置的指令模板，教 AI Agent 如何高效完成特定场景的任务。

### 通用操作

| Prompt | 说明 | 参数 |
|--------|------|------|
| `operate_remote_desktop` | 完整指南：截图→分析→操作→验证循环、坐标系、所有可用工具。 | （无） |
| `find_and_click` | 定位并点击特定 UI 元素的分步指令。 | `element_description`、`connection_id` |
| `run_command` | 打开终端运行命令的分步指令。 | `command`、`connection_id` |

### 运维与自动化

| Prompt | 说明 | 参数 |
|--------|------|------|
| `server_health_check` | 全面的服务器健康检查 — CPU、内存、磁盘、进程、服务、错误日志。生成结构化健康报告。 | `connection_id` |
| `batch_operation` | 在多台设备上依次执行同一任务的指南，含错误处理和汇总报告。 | `task_description` |

### 故障诊断

| Prompt | 说明 | 参数 |
|--------|------|------|
| `diagnose_system_issue` | 系统问题的系统性诊断（性能缓慢、崩溃、网络问题、磁盘满），含根因分析和修复建议。 | `issue_description`、`connection_id` |

### 屏幕理解

| Prompt | 说明 | 参数 |
|--------|------|------|
| `analyze_screen_content` | 深度分析屏幕内容 — 操作系统识别、打开的应用、文本提取、UI 元素清单、敏感信息安全扫描。 | `connection_id` |

### 多设备编排

| Prompt | 说明 | 参数 |
|--------|------|------|
| `multi_device_workflow` | 编排跨多台远程设备的复杂工作流，含依赖管理和跨设备数据传输。 | `task_description` |

### 操作文档化

| Prompt | 说明 | 参数 |
|--------|------|------|
| `document_procedure` | 观察或执行操作流程，自动生成标准操作规程（SOP）文档，含分步指令、截图和故障排除。 | `procedure_name`、`connection_id` |

完整使用示例请参阅 [典型场景 Demo](典型场景Demo.md)。

## 坐标系统

远程桌面的坐标空间由 `get_screen_size` 定义（例如 1920×1080）。

使用 `max_width`（例如 1280）截图时，图片会等比缩小，但远程屏幕坐标不变。转换公式：

```
screen_x = image_x × (screen_width / image_width)
screen_y = image_y × (screen_height / image_height)
```

在使用缩放截图计算点击坐标前，务必先调用 `get_screen_size`。

## 按键名称

以下按键名称可用于 `keyboard_hotkey`、`key_press` 和 `key_release`：

**修饰键**：`ctrl`、`shift`、`alt`、`win`（或 `meta`）

**功能键**：`f1` – `f12`

**导航键**：`enter`、`tab`、`escape`、`backspace`、`delete`、`insert`、`home`、`end`、`pageup`、`pagedown`、`up`、`down`、`left`、`right`

**特殊键**：`space`、`capslock`、`numlock`、`scrolllock`、`printscreen`、`pause`

**字母和数字**：`a` – `z`、`0` – `9`

**标点符号**：`minus`、`equal`、`leftbracket`、`rightbracket`、`backslash`、`semicolon`、`quote`、`backquote`、`comma`、`period`、`slash`

## 使用模式

### 基础：连接并截图

```
1. get_host_info          → 获取 device_id + access_code
2. connect_device          → 获取 connection_id
3. screenshot              → 查看屏幕内容
4. ... 交互操作 ...
5. disconnect_device       → 断开连接
```

### 无界面批量自动化

```
connect_device(show_window=false)  → 不弹出 GUI 窗口
screenshot → mouse_click → keyboard_type → ...
disconnect_device
```

### 多设备编排

```
conn_a = connect_device(device_id="111222333", ...)
conn_b = connect_device(device_id="444555666", ...)

screenshot(connection_id=conn_a)   → 查看设备 A
screenshot(connection_id=conn_b)   → 查看设备 B

keyboard_type(connection_id=conn_a, text="...")
keyboard_type(connection_id=conn_b, text="...")
```

### Modifier+点击（如 Ctrl+点击多选）

```
key_press(key="ctrl")
mouse_click(x=100, y=200)
mouse_click(x=300, y=400)     → 多选
key_release(key="ctrl")
```

### 拖拽选中文本

```
mouse_drag(start_x=100, start_y=200, end_x=500, end_y=200)
keyboard_hotkey(keys=["ctrl", "c"])
get_clipboard()                → 获取选中的文本
```

### 事件驱动：连接并等待就绪

使用 `wait_for_connection_state` 替代轮询 `list_connections`：

```
conn_id = connect_device(device_id="111222333", access_code="888888")
wait_for_connection_state(connection_id=conn_id, state="connected", timeout_ms=15000)
screenshot()                   → 屏幕已就绪
```

### 事件驱动：从远程复制文本

使用 `wait_for_clipboard_change` 替代轮询 `get_clipboard`：

```
mouse_drag(start_x=100, start_y=200, end_x=500, end_y=200)
keyboard_hotkey(keys=["ctrl", "c"])
wait_for_clipboard_change(connection_id=conn_id, timeout_ms=5000)
                               → 剪贴板更新后立即返回复制的文本
```

### 响应式：等待视频布局变化

```
// 等待远程桌面分辨率变化（例如窗口调整大小后）
wait_for_event(event="videoLayoutChanged", filter={"connectionId": conn_id}, timeout_ms=10000)
screenshot()                                         → 捕获新布局
```

### 查看事件历史

```
get_recent_events(event_type="connectionStateChanged", limit=10)
                               → 查看最近的连接状态变化
get_recent_events(limit=50)    → 查看所有近期事件
```

## 从源码编译

```bash
cd QuickDesk/quickdesk-mcp
cargo build --release
# 产物：target/release/quickdesk-mcp（Windows 上为 quickdesk-mcp.exe）
```

### 环境要求

- Rust 1.75+
- Cargo

### 依赖

所有依赖由 Cargo 管理。核心 crate：

| Crate | 用途 |
|-------|------|
| `rmcp` | Rust MCP SDK |
| `tokio-tungstenite` | WebSocket 客户端 |
| `serde` / `serde_json` | JSON 序列化 |
| `clap` | 命令行参数解析 |
| `tracing` | 结构化日志 |
| `schemars` | MCP 工具参数 JSON Schema 生成 |

## 常见问题

### 启动时提示 "Connection refused"

确保 QuickDesk 已启动。WebSocket API 服务器需要运行在 `ws://127.0.0.1:9600`。

### HTTP/SSE 模式："无法连接 MCP 服务器"

1. 确认 QuickDesk 已切换到 HTTP/SSE 模式并打开了 MCP HTTP 服务开关
2. 检查 URL 是否与 QuickDesk MCP 设置中显示的端点一致（默认：`http://127.0.0.1:18080/mcp`）
3. 确认防火墙未拦截 HTTP 端口
4. 如需远程访问，`--host` 应设为 `0.0.0.0`（而不是 `127.0.0.1`）

### 截图返回空

确认远程连接已建立且正在接收视频帧。调用 `list_connections` 检查连接状态是否为 "connected"。

### 点击坐标不准

如果截图使用了 `max_width` 缩放，需要使用 `get_screen_size` 获取实际分辨率，按比例换算坐标。

### 重新编译时提示 "文件被锁定"

编译前停止运行中的 `quickdesk-mcp` 进程：

```powershell
# Windows
Get-Process quickdesk-mcp -ErrorAction SilentlyContinue | Stop-Process -Force
```

## 日志

通过 `RUST_LOG` 环境变量启用调试日志：

```bash
RUST_LOG=debug quickdesk-mcp
```

日志输出到 stderr，不会干扰 stdio MCP 传输。

## 安全

QuickDesk MCP 包含完整的安全体系。

### 权限分级

| 级别 | Token 参数 | 允许的操作 |
|------|------------|-----------|
| **完全控制** | `--token` | 所有操作（截图、点击、输入、连接等） |
| **只读** | `--readonly-token` | 仅 `screenshot`、`getScreenSize`、`getHostInfo`、`getStatus`、`listConnections`、`getConnectionInfo`、`getClipboard`、`getHostClients`、`getSignalingStatus` |

### 设备白名单

限制 AI 可连接的设备：

```bash
quickdesk-mcp --token SECRET --allowed-devices "111222333,444555666,777888999"
```

连接白名单外的设备将返回 403 错误。

### 频率限制

防止 AI 失控高频操作：

```bash
quickdesk-mcp --token SECRET --rate-limit 60
```

超出限制返回 429 错误。使用滑动 1 分钟窗口计算。

### 会话超时

空闲自动断开：

```bash
quickdesk-mcp --token SECRET --session-timeout 3600
```

指定秒数内无活动的会话将被自动断开。

### 危险操作拦截

以下操作通过 API 自动拦截：

- `disconnectAll`（批量断开连接）
- `Alt+F4` 快捷键（关闭应用）
- `Ctrl+Alt+Delete` 快捷键
- 输入包含以下内容的文本：`shutdown`、`reboot`、`format`、`rm -rf`、`del /f /s /q`、`mkfs`

这些操作返回 403 错误并附带描述信息。

### 审计日志

所有 API 操作记录到 `logs/quickdesk_audit.log`（轮转文件，10MB × 5 个）：

```
[2026-03-01 23:45:12.345] [ALLOW] client_1 method=screenshot params={"connectionId":"abc123"}
[2026-03-01 23:45:13.456] [DENY] client_2 method=mouseClick params={"x":100,"y":200} reason=permission_denied
[2026-03-01 23:45:14.567] [DENY] client_1 method=keyboardType params={"text":"shutdown /s"} reason=dangerous_operation
```

### 安全配置示例

```json
{
  "mcpServers": {
    "quickdesk": {
      "command": "quickdesk-mcp",
      "args": [
        "--rate-limit", "120",
        "--session-timeout", "7200",
        "--allowed-devices", "111222333,444555666"
      ],
      "env": {
        "QUICKDESK_TOKEN": "your-secret-token"
      }
    }
  }
}
```
