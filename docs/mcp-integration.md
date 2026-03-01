# QuickDesk MCP Integration Guide

QuickDesk includes a built-in [MCP (Model Context Protocol)](https://modelcontextprotocol.io/) server that allows AI agents to see and control remote desktops programmatically.

## Architecture

```
AI Agent (Claude / Cursor / GPT)
    │  stdio (JSON-RPC 2.0)
    ▼
quickdesk-mcp (Rust binary)
    │  WebSocket
    ▼
QuickDesk GUI (Qt 6)
    │  Native Messaging + Shared Memory
    ▼
Remote Desktop (Chromium Remoting / WebRTC)
```

The `quickdesk-mcp` binary acts as a bridge: it speaks MCP over stdio to AI clients and forwards requests to QuickDesk's internal WebSocket API.

## Quick Start

### 1. Start QuickDesk

Launch QuickDesk normally. The WebSocket API server starts automatically on `ws://127.0.0.1:9800`.

### 2. Configure Your AI Client

#### Cursor IDE

Create or edit `.cursor/mcp.json` in your project root:

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

Edit `claude_desktop_config.json`:

- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`

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

#### Any MCP Client

`quickdesk-mcp` is a standard MCP stdio server. Any client that supports `stdio` transport can use it:

```bash
quickdesk-mcp [--ws-url ws://127.0.0.1:9800] [--token YOUR_TOKEN]
```

| Argument | Default | Description |
|----------|---------|-------------|
| `--ws-url` | `ws://127.0.0.1:9800` | QuickDesk WebSocket API URL |
| `--token` | (none) | Auth token for WebSocket server |

### 3. Use It

Once configured, your AI agent can use QuickDesk tools directly. Example conversation:

> **You**: Connect to my remote server (device ID: 123456789, access code: 888888) and check what's on screen.
>
> **AI Agent**: *(calls `connect_device` → `screenshot` → analyzes the image)* I can see the Windows desktop with File Explorer open...

## MCP Tools Reference

### Connection Management

| Tool | Description |
|------|-------------|
| `connect_device` | Connect to a remote device by device ID + access code. Returns a `connection_id`. Set `show_window=false` for headless automation. |
| `disconnect_device` | Disconnect a specific remote connection. |
| `disconnect_all` | Disconnect all active remote connections. |
| `list_connections` | List all active connections with their IDs and states. |
| `get_connection_info` | Get detailed info for a specific connection. |

### Vision & Screen

| Tool | Description |
|------|-------------|
| `screenshot` | Capture the remote screen. Returns base64 image. Use `max_width` to scale down for faster processing. |
| `get_screen_size` | Get the remote desktop resolution (width × height). |

### Mouse

| Tool | Description |
|------|-------------|
| `mouse_click` | Click at (x, y). Supports left/right/middle button. |
| `mouse_double_click` | Double-click at (x, y). |
| `mouse_move` | Move cursor to (x, y). |
| `mouse_scroll` | Scroll at (x, y) with delta_x/delta_y. |
| `mouse_drag` | Drag from (start_x, start_y) to (end_x, end_y). |

### Keyboard

| Tool | Description |
|------|-------------|
| `keyboard_type` | Type text using clipboard paste for reliable unicode input. |
| `keyboard_hotkey` | Press a key combination, e.g. `["ctrl", "c"]`, `["win", "r"]`, `["alt", "f4"]`. |
| `key_press` | Press and hold a key (for modifier+click scenarios). |
| `key_release` | Release a previously pressed key. |

### Clipboard

| Tool | Description |
|------|-------------|
| `get_clipboard` | Get the last clipboard content synced from remote. |
| `set_clipboard` | Set the remote clipboard content. |

### Host & System

| Tool | Description |
|------|-------------|
| `get_host_info` | Get local device ID, access code, and signaling state. Use this to connect to the current machine. |
| `get_host_clients` | List clients connected to this host. |
| `get_status` | Overall system status (host/client processes, signaling). |
| `get_signaling_status` | Signaling server connection status. |
| `refresh_access_code` | Generate a new access code for the local host. |

## MCP Resources

Resources provide read-only real-time data about system state.

| URI | Description |
|-----|-------------|
| `quickdesk://host` | Local device ID, access code, signaling state |
| `quickdesk://status` | Overall system status |
| `quickdesk://connection/{connectionId}` | Detailed info for a specific connection |

## MCP Prompts

Prompts are instruction templates that teach AI agents best practices for specific scenarios.

### General Operation

| Prompt | Description | Parameters |
|--------|-------------|------------|
| `operate_remote_desktop` | Complete guide for the screenshot→analyze→act→verify loop, coordinate system, and all available tools. | (none) |
| `find_and_click` | Step-by-step instructions for locating and clicking a specific UI element. | `element_description`, `connection_id` |
| `run_command` | Step-by-step instructions for opening a terminal and running a command. | `command`, `connection_id` |

### DevOps & Automation

| Prompt | Description | Parameters |
|--------|-------------|------------|
| `server_health_check` | Comprehensive server health check — CPU, memory, disk, processes, services, error logs. Generates a structured health report. | `connection_id` |
| `batch_operation` | Guide for executing the same task across multiple devices sequentially, with error handling and summary reporting. | `task_description` |

### Troubleshooting

| Prompt | Description | Parameters |
|--------|-------------|------------|
| `diagnose_system_issue` | Systematic diagnosis of system problems (slow performance, crashes, network issues, disk full) with root cause analysis and remediation suggestions. | `issue_description`, `connection_id` |

### Screen Intelligence

| Prompt | Description | Parameters |
|--------|-------------|------------|
| `analyze_screen_content` | Deep analysis of screen content — OS detection, open applications, text extraction, UI element inventory, and security scan for exposed sensitive information. | `connection_id` |

### Multi-Device

| Prompt | Description | Parameters |
|--------|-------------|------------|
| `multi_device_workflow` | Orchestrate complex workflows across multiple remote devices with dependency management and cross-device data transfer. | `task_description` |

### Documentation

| Prompt | Description | Parameters |
|--------|-------------|------------|
| `document_procedure` | Observe or perform a procedure and generate a Standard Operating Procedure (SOP) document with step-by-step instructions, screenshots, and troubleshooting. | `procedure_name`, `connection_id` |

See [Demo Scenarios](demo-scenarios.md) for complete usage examples of each prompt.

## Coordinate System

The remote desktop has a coordinate space defined by `get_screen_size` (e.g. 1920×1080).

When you take a screenshot with `max_width` (e.g. 1280), the image is scaled down but the remote screen coordinates remain unchanged. To convert image coordinates to screen coordinates:

```
screen_x = image_x × (screen_width / image_width)
screen_y = image_y × (screen_height / image_height)
```

Always call `get_screen_size` before computing click targets from scaled screenshots.

## Key Names

The following key names can be used with `keyboard_hotkey`, `key_press`, and `key_release`:

**Modifiers**: `ctrl`, `shift`, `alt`, `win` (or `meta`)

**Function keys**: `f1` – `f12`

**Navigation**: `enter`, `tab`, `escape`, `backspace`, `delete`, `insert`, `home`, `end`, `pageup`, `pagedown`, `up`, `down`, `left`, `right`

**Special**: `space`, `capslock`, `numlock`, `scrolllock`, `printscreen`, `pause`

**Letters & digits**: `a` – `z`, `0` – `9`

**Punctuation**: `minus`, `equal`, `leftbracket`, `rightbracket`, `backslash`, `semicolon`, `quote`, `backquote`, `comma`, `period`, `slash`

## Usage Patterns

### Basic: Connect and Screenshot

```
1. get_host_info          → get device_id + access_code
2. connect_device          → get connection_id
3. screenshot              → see what's on screen
4. ... interact ...
5. disconnect_device       → clean up
```

### Headless Batch Automation

```
connect_device(show_window=false)  → no GUI window opened
screenshot → mouse_click → keyboard_type → ...
disconnect_device
```

### Multi-Device Orchestration

```
conn_a = connect_device(device_id="111222333", ...)
conn_b = connect_device(device_id="444555666", ...)

screenshot(connection_id=conn_a)   → see device A
screenshot(connection_id=conn_b)   → see device B

keyboard_type(connection_id=conn_a, text="...")
keyboard_type(connection_id=conn_b, text="...")
```

### Modifier+Click (e.g. Ctrl+Click)

```
key_press(key="ctrl")
mouse_click(x=100, y=200)
mouse_click(x=300, y=400)     → multi-select
key_release(key="ctrl")
```

### Text Selection via Drag

```
mouse_drag(start_x=100, start_y=200, end_x=500, end_y=200)
keyboard_hotkey(keys=["ctrl", "c"])
get_clipboard()                → get selected text
```

## Building from Source

```bash
cd QuickDesk/quickdesk-mcp
cargo build --release
# Binary: target/release/quickdesk-mcp (or quickdesk-mcp.exe on Windows)
```

### Requirements

- Rust 1.75+
- Cargo

### Dependencies

All dependencies are managed by Cargo. Key crates:

| Crate | Purpose |
|-------|---------|
| `rmcp` | MCP SDK for Rust |
| `tokio-tungstenite` | WebSocket client |
| `serde` / `serde_json` | JSON serialization |
| `clap` | CLI argument parsing |
| `tracing` | Structured logging |
| `schemars` | JSON Schema generation for MCP tool parameters |

## Troubleshooting

### "Connection refused" on startup

Make sure QuickDesk is running. The WebSocket API server must be active at `ws://127.0.0.1:9800`.

### Screenshot returns empty

Ensure the remote connection is established and video frames are being received. Call `list_connections` to verify the connection state is "connected".

### Coordinates don't match

If using `max_width` in screenshots, remember to scale coordinates back to full screen resolution using `get_screen_size`.

### "File locked" when rebuilding

Stop any running `quickdesk-mcp` process before rebuilding:

```powershell
# Windows
Get-Process quickdesk-mcp -ErrorAction SilentlyContinue | Stop-Process -Force
```

## Logging

Enable debug logging via the `RUST_LOG` environment variable:

```bash
RUST_LOG=debug quickdesk-mcp
```

Logs are written to stderr and won't interfere with the stdio MCP transport.
