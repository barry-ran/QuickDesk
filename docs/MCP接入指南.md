# QuickDesk MCP 接入指南

QuickDesk 内置 [MCP (Model Context Protocol)](https://modelcontextprotocol.io/) Server，让 AI Agent 能够以编程方式查看和控制远程桌面。

## 架构

```
AI Agent (Claude / Cursor / GPT)
    │  stdio (JSON-RPC 2.0)
    ▼
quickdesk-mcp (Rust 桥接程序)
    │  WebSocket
    ▼
QuickDesk GUI (Qt 6)
    │  Native Messaging + 共享内存
    ▼
远程桌面 (Chromium Remoting / WebRTC)
```

`quickdesk-mcp` 充当桥梁：对上通过 stdio 与 AI 客户端通信（标准 MCP 协议），对下通过 WebSocket 调用 QuickDesk 内部 API。

## 快速开始

### 1. 启动 QuickDesk

正常启动 QuickDesk，WebSocket API 服务器会自动在 `ws://127.0.0.1:9800` 上运行。

### 2. 配置 AI 客户端

#### Cursor IDE

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

#### Claude Desktop

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

#### 通用 MCP 客户端

`quickdesk-mcp` 是标准的 MCP stdio 服务器，任何支持 `stdio` 传输的客户端都可以使用：

```bash
quickdesk-mcp [--ws-url ws://127.0.0.1:9800] [--token YOUR_TOKEN]
```

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--ws-url` | `ws://127.0.0.1:9800` | QuickDesk WebSocket API 地址 |
| `--token` | （空） | WebSocket 服务器认证 Token |

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

### 鼠标

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

确保 QuickDesk 已启动。WebSocket API 服务器需要运行在 `ws://127.0.0.1:9800`。

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
