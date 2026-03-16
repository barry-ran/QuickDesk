use rmcp::handler::server::router::prompt::PromptRouter;
use rmcp::handler::server::tool::ToolRouter;
use rmcp::handler::server::wrapper::Parameters;
use rmcp::model::{
    Annotated, ErrorData, GetPromptRequestParams, GetPromptResult, Implementation,
    ListPromptsResult, ListResourceTemplatesResult, ListResourcesResult, PaginatedRequestParams,
    PromptMessage, PromptMessageRole, RawResource, RawResourceTemplate, ReadResourceRequestParams,
    ReadResourceResult, ResourceContents, ServerCapabilities, ServerInfo,
};
use rmcp::service::{RequestContext, RoleServer};
use rmcp::{prompt, prompt_handler, prompt_router, tool, tool_handler, tool_router, ServerHandler};
use schemars::JsonSchema;
use serde::Deserialize;
use serde_json::json;

use crate::event_bus::EventBus;
use crate::ws_client::WsClient;

fn make_resource(uri: &str, name: &str, description: &str) -> Annotated<RawResource> {
    Annotated {
        raw: RawResource {
            uri: uri.to_string(),
            name: name.to_string(),
            title: None,
            description: Some(description.to_string()),
            mime_type: Some("application/json".to_string()),
            size: None,
            icons: None,
            meta: None,
        },
        annotations: None,
    }
}

// ---- Parameter structs ----

#[derive(Deserialize, JsonSchema)]
struct ConnectionIdParam {
    /// Connection ID
    connection_id: String,
}

#[derive(Deserialize, JsonSchema)]
struct ConnectDeviceParam {
    /// 9-digit device ID of the remote host
    device_id: String,
    /// Access code of the remote host
    access_code: String,
    /// Signaling server URL (optional, uses default if empty)
    server_url: Option<String>,
    /// Whether to show the remote desktop viewer window in QuickDesk UI. Defaults to true so the user can observe AI operations. Set to false for background/batch automation.
    show_window: Option<bool>,
}

#[derive(Deserialize, JsonSchema)]
struct ScreenshotParam {
    /// Connection ID
    connection_id: String,
    /// Maximum width of the screenshot in pixels. Image will be scaled down proportionally if wider than this value. Use this to reduce data transfer size.
    max_width: Option<i32>,
    /// Image format: "jpeg" (default) or "png"
    format: Option<String>,
    /// JPEG quality 1-100 (default: 80)
    quality: Option<i32>,
}

#[derive(Deserialize, JsonSchema)]
struct MouseClickParam {
    /// Connection ID
    connection_id: String,
    /// X coordinate
    x: f64,
    /// Y coordinate
    y: f64,
    /// Mouse button: "left", "right", or "middle". Defaults to "left".
    button: Option<String>,
}

#[derive(Deserialize, JsonSchema)]
struct MousePositionParam {
    /// Connection ID
    connection_id: String,
    /// X coordinate
    x: f64,
    /// Y coordinate
    y: f64,
}

#[derive(Deserialize, JsonSchema)]
struct MouseScrollParam {
    /// Connection ID
    connection_id: String,
    /// X coordinate
    x: f64,
    /// Y coordinate
    y: f64,
    /// Horizontal scroll delta
    delta_x: Option<f64>,
    /// Vertical scroll delta (positive=up, negative=down)
    delta_y: Option<f64>,
}

#[derive(Deserialize, JsonSchema)]
struct KeyboardTypeParam {
    /// Connection ID
    connection_id: String,
    /// Text to type
    text: String,
}

#[derive(Deserialize, JsonSchema)]
struct KeyboardHotkeyParam {
    /// Connection ID
    connection_id: String,
    /// Key names to press together, e.g. ["ctrl","c"], ["win","r"], ["alt","f4"]
    keys: Vec<String>,
}

#[derive(Deserialize, JsonSchema)]
struct SetClipboardParam {
    /// Connection ID
    connection_id: String,
    /// Text to set in remote clipboard
    text: String,
}

#[derive(Deserialize, JsonSchema)]
struct MouseDragParam {
    /// Connection ID
    connection_id: String,
    /// Start X coordinate
    start_x: f64,
    /// Start Y coordinate
    start_y: f64,
    /// End X coordinate
    end_x: f64,
    /// End Y coordinate
    end_y: f64,
    /// Mouse button: "left" (default), "right", or "middle"
    button: Option<String>,
}

#[derive(Deserialize, JsonSchema)]
struct KeyParam {
    /// Connection ID
    connection_id: String,
    /// Key name, e.g. "shift", "ctrl", "a", "enter". Same key names as keyboard_hotkey.
    key: String,
}

#[derive(Deserialize, JsonSchema)]
struct FindAndClickArgs {
    /// Description of the UI element to find, e.g. "the Save button", "the search input field"
    element_description: String,
    /// Connection ID of the remote desktop
    connection_id: String,
}

#[derive(Deserialize, JsonSchema)]
struct RunCommandArgs {
    /// The command to run, e.g. "dir C:\\", "ipconfig /all"
    command: String,
    /// Connection ID of the remote desktop
    connection_id: String,
}

#[derive(Deserialize, JsonSchema)]
struct ConnectionIdArg {
    /// Connection ID of the remote desktop
    connection_id: String,
}

#[derive(Deserialize, JsonSchema)]
struct DiagnoseIssueArgs {
    /// Description of the problem to diagnose, e.g. "computer is very slow", "internet not working", "application crashes"
    issue_description: String,
    /// Connection ID of the remote desktop
    connection_id: String,
}

// ---- Event-related parameter structs ----

#[derive(Deserialize, JsonSchema)]
struct WaitForEventParam {
    /// Event type to wait for, e.g. "connectionStateChanged", "clipboardChanged", "connectionAdded", "videoLayoutChanged". Call list_event_types for all options.
    event: String,
    /// Optional filter: JSON object where each key-value pair must match the event data. E.g. {"state": "connected"} will only match events whose data.state == "connected".
    filter: Option<serde_json::Value>,
    /// Maximum time to wait in milliseconds. Returns an error if no matching event arrives within this time.
    timeout_ms: Option<u64>,
}

#[derive(Deserialize, JsonSchema)]
struct WaitForConnectionStateParam {
    /// Connection ID to monitor
    connection_id: String,
    /// Target state to wait for, e.g. "connected", "disconnected", "failed"
    state: String,
    /// Maximum time to wait in milliseconds (default: 30000)
    timeout_ms: Option<u64>,
}

#[derive(Deserialize, JsonSchema)]
struct WaitForClipboardChangeParam {
    /// Connection ID to monitor
    connection_id: String,
    /// Maximum time to wait in milliseconds (default: 10000)
    timeout_ms: Option<u64>,
}

#[derive(Deserialize, JsonSchema)]
struct GetRecentEventsParam {
    /// Filter by event type. If empty, returns all event types.
    event_type: Option<String>,
    /// Maximum number of events to return (default: 20, max: 100)
    limit: Option<usize>,
}

// ---- OCR / UI state param structs ----

#[derive(Deserialize, JsonSchema)]
struct GetScreenTextParam {
    /// Connection ID of the remote desktop
    connection_id: String,
}

#[derive(Deserialize, JsonSchema)]
struct FindElementParam {
    /// Connection ID of the remote desktop
    connection_id: String,
    /// Text to search for on screen (supports partial match by default)
    text: String,
    /// If true, require exact text match. Default: false (partial match)
    exact: Option<bool>,
    /// If true, ignore letter case. Default: true
    ignore_case: Option<bool>,
}

#[derive(Deserialize, JsonSchema)]
struct ClickTextParam {
    /// Connection ID of the remote desktop
    connection_id: String,
    /// Text to find and click. Clicks the center of the first matched text block.
    text: String,
    /// If true, require exact text match. Default: false
    exact: Option<bool>,
    /// If true, ignore letter case. Default: true
    ignore_case: Option<bool>,
    /// Mouse button: "left" (default), "right", or "middle"
    button: Option<String>,
}

#[derive(Deserialize, JsonSchema)]
struct MultiDeviceTaskArgs {
    /// Description of the task to perform across devices, e.g. "check disk usage on all servers", "install security updates"
    task_description: String,
}

#[derive(Deserialize, JsonSchema)]
struct DocumentProcedureArgs {
    /// Description of the procedure to document, e.g. "how to deploy the application", "how to configure the firewall"
    procedure_name: String,
    /// Connection ID of the remote desktop
    connection_id: String,
}

// ---- UI state param structs ----

#[derive(Deserialize, JsonSchema)]
struct GetUiStateParam {
    /// Connection ID of the remote desktop
    connection_id: String,
}

#[derive(Deserialize, JsonSchema)]
struct WaitForTextParam {
    /// Connection ID of the remote desktop
    connection_id: String,
    /// Text to wait for on screen (supports partial match by default)
    text: String,
    /// If true, require exact text match. Default: false
    exact: Option<bool>,
    /// If true, ignore letter case. Default: true
    ignore_case: Option<bool>,
    /// Maximum time to wait in milliseconds (default: 5000)
    timeout_ms: Option<i64>,
}

#[derive(Deserialize, JsonSchema)]
struct AssertTextPresentParam {
    /// Connection ID of the remote desktop
    connection_id: String,
    /// Text to assert is present on screen (supports partial match by default)
    text: String,
    /// If true, require exact text match. Default: false
    exact: Option<bool>,
    /// If true, ignore letter case. Default: true
    ignore_case: Option<bool>,
}

// ---- Verification param structs ----

#[derive(Deserialize, JsonSchema)]
struct VerificationCondition {
    /// Condition type: "text_present" | "text_absent" | "text_present_exact" |
    /// "window_title_contains" | "window_title_equals"
    #[serde(rename = "type")]
    condition_type: String,
    /// The expected value to check against
    value: String,
}

#[derive(Deserialize, JsonSchema)]
struct VerifyActionResultParam {
    /// Connection ID of the remote desktop
    connection_id: String,
    /// List of conditions that must all pass
    expectations: Vec<VerificationCondition>,
    /// Maximum time to poll in milliseconds (default: 3000)
    timeout_ms: Option<i32>,
}

#[derive(Deserialize, JsonSchema)]
struct ScreenDiffParam {
    /// Connection ID of the remote desktop
    connection_id: String,
    /// Frame hash from a previous get_ui_state or screen_diff_summary call.
    /// Leave empty to compare against an empty baseline (shows all current text as "added").
    from_hash: Option<String>,
}

#[derive(Deserialize, JsonSchema)]
struct AssertScreenStateParam {
    /// Connection ID of the remote desktop
    connection_id: String,
    /// List of conditions to assert simultaneously (no polling)
    expectations: Vec<VerificationCondition>,
}

// ---- MCP Server ----

#[derive(Clone)]
pub struct QuickDeskMcpServer {
    tool_router: ToolRouter<Self>,
    prompt_router: PromptRouter<Self>,
    ws: WsClient,
    allowed_devices: Vec<String>,
    event_bus: EventBus,
}

impl QuickDeskMcpServer {
    pub fn new(ws: WsClient, allowed_devices: Vec<String>) -> Self {
        let event_bus = ws.event_bus().clone();
        Self {
            tool_router: Self::tool_router(),
            prompt_router: Self::prompt_router(),
            ws,
            allowed_devices,
            event_bus,
        }
    }

    fn check_device_allowed(&self, device_id: &str) -> Result<(), String> {
        if self.allowed_devices.is_empty() {
            return Ok(());
        }
        if self.allowed_devices.iter().any(|d| d == device_id) {
            Ok(())
        } else {
            Err(format!(
                "Device '{}' is not in the allowed device list. Allowed: {:?}",
                device_id, self.allowed_devices
            ))
        }
    }
}

#[tool_router]
impl QuickDeskMcpServer {
    #[tool(description = "Get local host device ID, access code, signaling state, and client count. Use this to get credentials for connecting to the current computer.")]
    async fn get_host_info(&self) -> String {
        match self.ws.request("getHostInfo", json!({})).await {
            Ok(v) => serde_json::to_string_pretty(&v).unwrap_or_default(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "List clients currently connected to this host machine.")]
    async fn get_host_clients(&self) -> String {
        match self.ws.request("getHostClients", json!({})).await {
            Ok(v) => serde_json::to_string_pretty(&v).unwrap_or_default(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Get overall status including host process, client process, and signaling server state.")]
    async fn get_status(&self) -> String {
        match self.ws.request("getStatus", json!({})).await {
            Ok(v) => serde_json::to_string_pretty(&v).unwrap_or_default(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Get signaling server connection status for both host and client.")]
    async fn get_signaling_status(&self) -> String {
        match self.ws.request("getSignalingStatus", json!({})).await {
            Ok(v) => serde_json::to_string_pretty(&v).unwrap_or_default(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Refresh the local host access code.")]
    async fn refresh_access_code(&self) -> String {
        match self.ws.request("refreshAccessCode", json!({})).await {
            Ok(v) => serde_json::to_string_pretty(&v).unwrap_or_default(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "List all active remote desktop connections.")]
    async fn list_connections(&self) -> String {
        match self.ws.request("listConnections", json!({})).await {
            Ok(v) => serde_json::to_string_pretty(&v).unwrap_or_default(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Get detailed info for a specific remote connection.")]
    async fn get_connection_info(&self, params: Parameters<ConnectionIdParam>) -> String {
        match self
            .ws
            .request("getConnectionInfo", json!({ "connectionId": params.0.connection_id }))
            .await
        {
            Ok(v) => serde_json::to_string_pretty(&v).unwrap_or_default(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Connect to a remote device. Returns a connection ID. By default, a remote desktop viewer window is shown so the user can observe your operations. Set show_window=false for silent background automation. To control the current computer, first call get_host_info to get the device ID and access code, then pass them here.")]
    async fn connect_device(&self, params: Parameters<ConnectDeviceParam>) -> String {
        let p = params.0;
        if let Err(e) = self.check_device_allowed(&p.device_id) {
            return format!("Error: {e}");
        }
        let mut req = json!({
            "deviceId": p.device_id,
            "accessCode": p.access_code,
        });
        if let Some(url) = p.server_url {
            req["serverUrl"] = json!(url);
        }
        if let Some(show) = p.show_window {
            req["showWindow"] = json!(show);
        }
        match self.ws.request("connectToHost", req).await {
            Ok(v) => serde_json::to_string_pretty(&v).unwrap_or_default(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Disconnect from a remote device.")]
    async fn disconnect_device(&self, params: Parameters<ConnectionIdParam>) -> String {
        match self
            .ws
            .request("disconnectFromHost", json!({ "connectionId": params.0.connection_id }))
            .await
        {
            Ok(v) => serde_json::to_string_pretty(&v).unwrap_or_default(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Disconnect all remote connections.")]
    async fn disconnect_all(&self) -> String {
        match self.ws.request("disconnectAll", json!({})).await {
            Ok(v) => serde_json::to_string_pretty(&v).unwrap_or_default(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Capture a screenshot of the remote desktop. Returns base64 JPEG image data with width and height. Use max_width to reduce image size for faster transfer.")]
    async fn screenshot(&self, params: Parameters<ScreenshotParam>) -> String {
        let p = params.0;
        let mut req = json!({
            "connectionId": p.connection_id,
            "format": p.format.unwrap_or_else(|| "jpeg".to_string()),
            "quality": p.quality.unwrap_or(80),
        });
        if let Some(mw) = p.max_width {
            req["maxWidth"] = json!(mw);
        }
        match self.ws.request("screenshot", req).await {
            Ok(v) => serde_json::to_string_pretty(&v).unwrap_or_default(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Click at coordinates on the remote desktop.")]
    async fn mouse_click(&self, params: Parameters<MouseClickParam>) -> String {
        let p = params.0;
        match self
            .ws
            .request(
                "mouseClick",
                json!({
                    "connectionId": p.connection_id,
                    "x": p.x, "y": p.y,
                    "button": p.button.unwrap_or_else(|| "left".to_string()),
                }),
            )
            .await
        {
            Ok(_) => format!("Clicked at ({}, {})", p.x, p.y),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Double-click at coordinates on the remote desktop.")]
    async fn mouse_double_click(&self, params: Parameters<MousePositionParam>) -> String {
        let p = params.0;
        match self
            .ws
            .request(
                "mouseDoubleClick",
                json!({
                    "connectionId": p.connection_id,
                    "x": p.x, "y": p.y,
                }),
            )
            .await
        {
            Ok(_) => format!("Double-clicked at ({}, {})", p.x, p.y),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Move the mouse cursor to coordinates on the remote desktop.")]
    async fn mouse_move(&self, params: Parameters<MousePositionParam>) -> String {
        let p = params.0;
        match self
            .ws
            .request(
                "mouseMove",
                json!({
                    "connectionId": p.connection_id,
                    "x": p.x, "y": p.y,
                }),
            )
            .await
        {
            Ok(_) => format!("Moved to ({}, {})", p.x, p.y),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Scroll the mouse wheel on the remote desktop.")]
    async fn mouse_scroll(&self, params: Parameters<MouseScrollParam>) -> String {
        let p = params.0;
        match self
            .ws
            .request(
                "mouseScroll",
                json!({
                    "connectionId": p.connection_id,
                    "x": p.x, "y": p.y,
                    "deltaX": p.delta_x.unwrap_or(0.0),
                    "deltaY": p.delta_y.unwrap_or(0.0),
                }),
            )
            .await
        {
            Ok(_) => "Scrolled".to_string(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Type text on the remote desktop. Uses clipboard paste for reliable unicode input.")]
    async fn keyboard_type(&self, params: Parameters<KeyboardTypeParam>) -> String {
        let p = params.0;
        match self
            .ws
            .request(
                "keyboardType",
                json!({
                    "connectionId": p.connection_id,
                    "text": p.text,
                }),
            )
            .await
        {
            Ok(_) => format!("Typed: {}", p.text),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Send a keyboard shortcut (hotkey) on the remote desktop. Keys are pressed in order and released in reverse. Examples: [\"ctrl\",\"c\"], [\"ctrl\",\"shift\",\"esc\"], [\"win\",\"r\"], [\"alt\",\"f4\"]")]
    async fn keyboard_hotkey(&self, params: Parameters<KeyboardHotkeyParam>) -> String {
        let p = params.0;
        match self
            .ws
            .request(
                "keyboardHotkey",
                json!({
                    "connectionId": p.connection_id,
                    "keys": p.keys,
                }),
            )
            .await
        {
            Ok(_) => format!("Hotkey sent: {}", p.keys.join("+")),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Get the last known clipboard content from the remote desktop. Clipboard is synced automatically when something is copied on the remote machine.")]
    async fn get_clipboard(&self, params: Parameters<ConnectionIdParam>) -> String {
        let p = params.0;
        match self
            .ws
            .request("getClipboard", json!({"connectionId": p.connection_id}))
            .await
        {
            Ok(v) => serde_json::to_string_pretty(&v).unwrap_or_default(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Set remote clipboard content.")]
    async fn set_clipboard(&self, params: Parameters<SetClipboardParam>) -> String {
        let p = params.0;
        match self
            .ws
            .request(
                "setClipboard",
                json!({
                    "connectionId": p.connection_id,
                    "text": p.text,
                }),
            )
            .await
        {
            Ok(_) => "Clipboard set".to_string(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Drag the mouse from one position to another on the remote desktop. Useful for drag-and-drop, text selection, resizing windows, and moving objects.")]
    async fn mouse_drag(&self, params: Parameters<MouseDragParam>) -> String {
        let p = params.0;
        match self
            .ws
            .request(
                "mouseDrag",
                json!({
                    "connectionId": p.connection_id,
                    "startX": p.start_x, "startY": p.start_y,
                    "endX": p.end_x, "endY": p.end_y,
                    "button": p.button.unwrap_or_else(|| "left".to_string()),
                }),
            )
            .await
        {
            Ok(_) => format!(
                "Dragged from ({}, {}) to ({}, {})",
                p.start_x, p.start_y, p.end_x, p.end_y
            ),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Press and hold a key on the remote desktop. The key stays pressed until explicitly released with key_release. Useful for modifier keys (shift, ctrl, alt) while performing mouse operations like Ctrl+click for multi-select.")]
    async fn key_press(&self, params: Parameters<KeyParam>) -> String {
        let p = params.0;
        match self
            .ws
            .request(
                "keyPress",
                json!({
                    "connectionId": p.connection_id,
                    "key": p.key,
                }),
            )
            .await
        {
            Ok(_) => format!("Key pressed: {}", p.key),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Release a previously pressed key on the remote desktop. Must be paired with a prior key_press call.")]
    async fn key_release(&self, params: Parameters<KeyParam>) -> String {
        let p = params.0;
        match self
            .ws
            .request(
                "keyRelease",
                json!({
                    "connectionId": p.connection_id,
                    "key": p.key,
                }),
            )
            .await
        {
            Ok(_) => format!("Key released: {}", p.key),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Get the remote screen resolution.")]
    async fn get_screen_size(&self, params: Parameters<ConnectionIdParam>) -> String {
        match self
            .ws
            .request("getScreenSize", json!({ "connectionId": params.0.connection_id }))
            .await
        {
            Ok(v) => serde_json::to_string_pretty(&v).unwrap_or_default(),
            Err(e) => format!("Error: {e}"),
        }
    }

    // ---- Event tools ----

    #[tool(description = "Wait for a specific event from the remote desktop. Instead of polling with repeated screenshots, use this to efficiently wait for state changes like connection established, clipboard updated, video layout changed, etc. Returns the matching event data or an error on timeout. Call list_event_types to see all available event types.")]
    async fn wait_for_event(&self, params: Parameters<WaitForEventParam>) -> String {
        let p = params.0;
        let timeout = p.timeout_ms.unwrap_or(30000);

        match self
            .event_bus
            .wait_for(&p.event, p.filter.as_ref(), timeout, false)
            .await
        {
            Ok(event) => serde_json::to_string_pretty(&event).unwrap_or_default(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Wait for a remote connection to reach a specific state. Use this after connect_device to wait until the connection is fully established, instead of polling with screenshots. Returns the event data when the target state is reached, or an error on timeout.")]
    async fn wait_for_connection_state(
        &self,
        params: Parameters<WaitForConnectionStateParam>,
    ) -> String {
        let p = params.0;
        let timeout = p.timeout_ms.unwrap_or(30000);
        let filter = json!({ "connectionId": p.connection_id, "state": p.state });

        // check_history=true: safe because each connect_device creates a unique
        // connection ID, so a stale match for the same ID cannot exist.
        match self
            .event_bus
            .wait_for("connectionStateChanged", Some(&filter), timeout, true)
            .await
        {
            Ok(event) => serde_json::to_string_pretty(&event).unwrap_or_default(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Wait for the remote clipboard content to change. Use this after copying text on the remote desktop (e.g. via Ctrl+C) to get the clipboard content without polling. Returns the clipboard change event with the new content.")]
    async fn wait_for_clipboard_change(
        &self,
        params: Parameters<WaitForClipboardChangeParam>,
    ) -> String {
        let p = params.0;
        let timeout = p.timeout_ms.unwrap_or(10000);
        let filter = json!({ "connectionId": p.connection_id });

        // check_history=false: must wait for a NEW clipboard change, not return stale data
        match self
            .event_bus
            .wait_for("clipboardChanged", Some(&filter), timeout, false)
            .await
        {
            Ok(event) => serde_json::to_string_pretty(&event).unwrap_or_default(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Get recent events received from the remote desktop. Returns cached events in chronological order. Useful for checking what happened recently without waiting for new events. Each event has: event (type name), data (payload), timestamp (unix ms).")]
    async fn get_recent_events(&self, params: Parameters<GetRecentEventsParam>) -> String {
        let p = params.0;
        let limit = p.limit.unwrap_or(20).min(100);
        let events = self
            .event_bus
            .recent_events(p.event_type.as_deref(), limit)
            .await;

        serde_json::to_string_pretty(&events).unwrap_or_default()
    }

    // ---- OCR / UI state tools ----

    #[tool(description = "Extract all text from the current remote desktop screen using OCR (PP-OCRv4). \
Returns a list of text blocks with their text content, bounding box (x/y/w/h in pixels), center \
coordinates, and confidence score. Supports both Chinese and English. \
Results are cached per frame — calling this multiple times on the same frame costs nothing. \
Use this instead of screenshot+vision for text-heavy tasks to reduce token cost and improve reliability.")]
    async fn get_screen_text(&self, params: Parameters<GetScreenTextParam>) -> String {
        let p = params.0;
        match self
            .ws
            .request("getScreenText", json!({ "connectionId": p.connection_id }))
            .await
        {
            Ok(v) => serde_json::to_string_pretty(&v).unwrap_or_default(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Find a UI element on the remote desktop by its visible text. \
Runs OCR on the current screen and returns all blocks that match the search text. \
Each match includes the bounding box and center coordinates. \
Use exact=true for precise matching (e.g. button labels); leave it false for partial search. \
Returns found=false if the text is not on screen.")]
    async fn find_element(&self, params: Parameters<FindElementParam>) -> String {
        let p = params.0;
        let mut req = json!({
            "connectionId": p.connection_id,
            "text": p.text,
        });
        if let Some(exact) = p.exact {
            req["exact"] = json!(exact);
        }
        if let Some(ic) = p.ignore_case {
            req["ignoreCase"] = json!(ic);
        }
        match self.ws.request("findElement", req).await {
            Ok(v) => serde_json::to_string_pretty(&v).unwrap_or_default(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Find text on the remote desktop and click it in one step. \
Equivalent to find_element + mouse_click at the text center. \
If multiple matches exist, clicks the first one. \
Returns success=true with the clicked text and coordinates, or found=false if not on screen. \
Prefer this over screenshot→find coordinates→click for text-based UI interactions.")]
    async fn click_text(&self, params: Parameters<ClickTextParam>) -> String {
        let p = params.0;
        let mut req = json!({
            "connectionId": p.connection_id,
            "text": p.text,
        });
        if let Some(exact) = p.exact {
            req["exact"] = json!(exact);
        }
        if let Some(ic) = p.ignore_case {
            req["ignoreCase"] = json!(ic);
        }
        if let Some(btn) = p.button {
            req["button"] = json!(btn);
        }
        match self.ws.request("clickText", req).await {
            Ok(v) => serde_json::to_string_pretty(&v).unwrap_or_default(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Get a unified UI state snapshot: screen resolution, OCR text blocks, and active window title. \
Returns structured data instead of a raw image, reducing token cost and enabling reliable text-based navigation. \
Use this as a lightweight alternative to screenshot when you need to understand what is on screen without visual analysis. \
The `ocr.blocks` array contains every recognised text block with its coordinates.")]
    async fn get_ui_state(&self, params: Parameters<GetUiStateParam>) -> String {
        match self
            .ws
            .request("getUiState", json!({ "connectionId": params.0.connection_id }))
            .await
        {
            Ok(v) => serde_json::to_string_pretty(&v).unwrap_or_default(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Block until the specified text appears on the remote desktop screen, or until the timeout expires. \
Returns found=true with the matching text block coordinates when the text appears. \
Use this after performing an action to wait for the expected result (e.g. wait for 'Save successful' after saving a file). \
Prefer this over polling with screenshot in a loop.")]
    async fn wait_for_text(&self, params: Parameters<WaitForTextParam>) -> String {
        let p = params.0;
        let mut req = json!({
            "connectionId": p.connection_id,
            "text": p.text,
            "timeoutMs": p.timeout_ms.unwrap_or(5000),
        });
        if let Some(exact) = p.exact {
            req["exact"] = json!(exact);
        }
        if let Some(ic) = p.ignore_case {
            req["ignoreCase"] = json!(ic);
        }
        match self.ws.request("waitForText", req).await {
            Ok(v) => serde_json::to_string_pretty(&v).unwrap_or_default(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Assert that the specified text is currently visible on the remote desktop screen. \
Returns present=true with the matching text block if found, or present=false if not found. \
Use this to validate the current screen state before proceeding with the next step. \
Unlike wait_for_text, this returns immediately without polling.")]
    async fn assert_text_present(&self, params: Parameters<AssertTextPresentParam>) -> String {
        let p = params.0;
        let mut req = json!({
            "connectionId": p.connection_id,
            "text": p.text,
        });
        if let Some(exact) = p.exact {
            req["exact"] = json!(exact);
        }
        if let Some(ic) = p.ignore_case {
            req["ignoreCase"] = json!(ic);
        }
        match self.ws.request("assertTextPresent", req).await {
            Ok(v) => serde_json::to_string_pretty(&v).unwrap_or_default(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Verify that a set of conditions are met after performing an action. \
Polls OCR and window state until all conditions pass or the timeout elapses. \
Conditions can check for text presence/absence on screen or the active window title. \
Returns allPassed=true with per-condition details, or allPassed=false with a failure summary \
that the agent can use to decide the next action. \
Supported condition types: \"text_present\", \"text_absent\", \"text_present_exact\", \
\"window_title_contains\", \"window_title_equals\".")]
    async fn verify_action_result(&self, params: Parameters<VerifyActionResultParam>) -> String {
        let p = params.0;
        let expectations: Vec<serde_json::Value> = p.expectations.iter().map(|c| {
            json!({ "type": c.condition_type, "value": c.value })
        }).collect();
        let req = json!({
            "connectionId": p.connection_id,
            "expectations": expectations,
            "timeoutMs": p.timeout_ms.unwrap_or(3000),
        });
        match self.ws.request("verifyActionResult", req).await {
            Ok(v) => serde_json::to_string_pretty(&v).unwrap_or_default(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Compare the current screen's OCR state against a previous snapshot identified \
by from_hash. Returns a structured diff listing text blocks that appeared or disappeared between the \
two frames, plus a human-readable summary. \
Use get_ui_state to capture the baseline frame_hash before performing an action, then call \
screen_diff_summary afterwards to understand what changed.")]
    async fn screen_diff_summary(&self, params: Parameters<ScreenDiffParam>) -> String {
        let p = params.0;
        let req = json!({
            "connectionId": p.connection_id,
            "fromHash": p.from_hash.unwrap_or_default(),
        });
        match self.ws.request("screenDiffSummary", req).await {
            Ok(v) => serde_json::to_string_pretty(&v).unwrap_or_default(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "Immediately assert that a set of conditions are all satisfied on the current \
screen without any polling. Returns allPassed=true/false with per-condition detail and a summary. \
Use this as a pre-condition check before a risky action (e.g. confirm a delete dialog is visible \
before clicking OK). For post-action verification with retries, use verify_action_result instead.")]
    async fn assert_screen_state(&self, params: Parameters<AssertScreenStateParam>) -> String {
        let p = params.0;
        let expectations: Vec<serde_json::Value> = p.expectations.iter().map(|c| {
            json!({ "type": c.condition_type, "value": c.value })
        }).collect();
        let req = json!({
            "connectionId": p.connection_id,
            "expectations": expectations,
        });
        match self.ws.request("assertScreenState", req).await {
            Ok(v) => serde_json::to_string_pretty(&v).unwrap_or_default(),
            Err(e) => format!("Error: {e}"),
        }
    }

    #[tool(description = "List all supported event types that can be waited on with wait_for_event. Returns the event type names and their descriptions.")]
    async fn list_event_types(&self) -> String {
        let types = json!([
            {
                "event": "connectionStateChanged",
                "description": "Fired when a remote connection changes state (connecting, connected, disconnected, failed)",
                "data_fields": ["connectionId", "state", "hostInfo"]
            },
            {
                "event": "clipboardChanged",
                "description": "Fired when the remote clipboard content changes",
                "data_fields": ["connectionId", "text"]
            },
            {
                "event": "connectionAdded",
                "description": "Fired when a new outgoing connection is created",
                "data_fields": ["connectionId", "deviceId"]
            },
            {
                "event": "connectionRemoved",
                "description": "Fired when an outgoing connection is removed",
                "data_fields": ["connectionId"]
            },
            {
                "event": "videoLayoutChanged",
                "description": "Fired when the remote desktop video resolution changes",
                "data_fields": ["connectionId", "width", "height"]
            },
            {
                "event": "hostReady",
                "description": "Fired when the local host service is ready to accept connections",
                "data_fields": ["deviceId", "accessCode"]
            },
            {
                "event": "accessCodeChanged",
                "description": "Fired when the local host access code is refreshed",
                "data_fields": ["accessCode"]
            },
            {
                "event": "hostClientConnected",
                "description": "Fired when a remote client connects to this host",
                "data_fields": ["connectionId"]
            },
            {
                "event": "hostClientDisconnected",
                "description": "Fired when a remote client disconnects from this host",
                "data_fields": ["connectionId", "reason"]
            },
            {
                "event": "hostSignalingStateChanged",
                "description": "Fired when the host signaling server connection state changes",
                "data_fields": ["state", "retryCount", "nextRetryIn", "error"]
            },
            {
                "event": "hostProcessStatusChanged",
                "description": "Fired when the host process status changes",
                "data_fields": ["status"]
            },
            {
                "event": "clientProcessStatusChanged",
                "description": "Fired when the client process status changes",
                "data_fields": ["status"]
            }
        ]);
        serde_json::to_string_pretty(&types).unwrap_or_default()
    }
}

#[prompt_router]
impl QuickDeskMcpServer {
    /// System prompt for AI agents operating a remote desktop via QuickDesk.
    /// Explains the screenshot-analyze-act loop, coordinate system, best practices, and available tools.
    #[prompt(name = "operate_remote_desktop")]
    async fn operate_remote_desktop(&self) -> Result<Vec<PromptMessage>, ErrorData> {
        Ok(vec![PromptMessage::new_text(
            PromptMessageRole::User,
            "You are controlling a remote desktop through QuickDesk MCP tools. Follow this workflow:\n\
            \n\
            ## Core Loop: Screenshot → Analyze → Act → Verify\n\
            \n\
            1. **Screenshot**: Call `screenshot` to see the current screen state. Use `max_width: 1280` to reduce transfer size while keeping enough detail.\n\
            2. **Analyze**: Study the screenshot to understand what's on screen. Identify UI elements, their positions, and the current state.\n\
            3. **Act**: Perform ONE action (click, type, hotkey, etc.) based on your analysis.\n\
            4. **Verify**: Take another screenshot to confirm the action succeeded. If it didn't, retry or try an alternative approach.\n\
            \n\
            ## Coordinate System\n\
            \n\
            - Call `get_screen_size` to get the remote desktop resolution (e.g. 1920x1080).\n\
            - Screenshot coordinates map directly to screen coordinates when using full resolution.\n\
            - If you used `max_width` in screenshot, scale coordinates proportionally: `actual_x = screenshot_x * (screen_width / image_width)`.\n\
            \n\
            ## Best Practices\n\
            \n\
            - **Wait after actions**: UI animations and loading take time. After clicking or typing, wait briefly before taking the next screenshot.\n\
            - **One action at a time**: Don't chain multiple actions without verifying each one.\n\
            - **Use keyboard shortcuts**: They are more reliable than clicking menus. E.g. Ctrl+S to save, Ctrl+C to copy, Win+R to open Run dialog.\n\
            - **Type via keyboard_type**: It uses clipboard paste internally for reliable unicode support.\n\
            - **Error recovery**: If an action fails, take a screenshot to understand the current state, then try an alternative approach.\n\
            - **Modifier+click**: Use `key_press` to hold a modifier (e.g. \"ctrl\"), then `mouse_click`, then `key_release` the modifier. This enables Ctrl+click for multi-select.\n\
            - **Drag operations**: Use `mouse_drag` for selecting text, moving files, resizing windows, or drag-and-drop.\n\
            \n\
            ## Available Tools Summary\n\
            \n\
            | Category | Tools |\n\
            |----------|-------|\n\
            | Connection | connect_device, disconnect_device, list_connections |\n\
            | Vision | screenshot, get_screen_size |\n\
            | Mouse | mouse_click, mouse_double_click, mouse_move, mouse_scroll, mouse_drag |\n\
            | Keyboard | keyboard_type, keyboard_hotkey, key_press, key_release |\n\
            | Clipboard | get_clipboard, set_clipboard |\n\
            | System | get_host_info, get_status |"
        )])
    }

    /// Guide for finding and clicking a specific UI element on the remote desktop.
    #[prompt(name = "find_and_click")]
    async fn find_and_click(
        &self,
        params: Parameters<FindAndClickArgs>,
    ) -> Result<Vec<PromptMessage>, ErrorData> {
        let args = params.0;
        Ok(vec![PromptMessage::new_text(
            PromptMessageRole::User,
            format!(
                "Find and click the following element on the remote desktop: \"{}\"\n\
                Connection ID: {}\n\
                \n\
                Follow these steps:\n\
                \n\
                1. Call `screenshot` with connection_id=\"{}\" and max_width=1280 to see the current screen.\n\
                2. Analyze the screenshot to locate \"{}\".\n\
                   - Look for text labels, icons, or visual cues that match the description.\n\
                   - If the element is not visible, you may need to scroll or navigate to find it.\n\
                3. Call `get_screen_size` to get the actual screen resolution.\n\
                4. Calculate the actual coordinates: if the screenshot is smaller than the screen, scale up proportionally.\n\
                   - `actual_x = screenshot_x * (screen_width / image_width)`\n\
                   - `actual_y = screenshot_y * (screen_height / image_height)`\n\
                5. Click at the calculated coordinates using `mouse_click`.\n\
                6. Take another screenshot to verify the click worked (e.g. a menu opened, a button was pressed, a page navigated).\n\
                7. If the click didn't hit the right target, adjust coordinates and retry.\n\
                \n\
                Tips:\n\
                - Click the CENTER of the element, not the edge.\n\
                - For small elements (checkboxes, radio buttons), be extra precise with coordinates.\n\
                - If the element is in a scrollable area and not visible, use mouse_scroll first.",
                args.element_description, args.connection_id,
                args.connection_id, args.element_description
            ),
        )])
    }

    /// Guide for running a command or script on the remote desktop via terminal/command prompt.
    #[prompt(name = "run_command")]
    async fn run_command(
        &self,
        params: Parameters<RunCommandArgs>,
    ) -> Result<Vec<PromptMessage>, ErrorData> {
        let args = params.0;
        Ok(vec![PromptMessage::new_text(
            PromptMessageRole::User,
            format!(
                "Run the following command on the remote desktop: `{}`\n\
                Connection ID: {}\n\
                \n\
                Follow these steps:\n\
                \n\
                1. Take a screenshot to see the current screen state.\n\
                2. Open a terminal / command prompt:\n\
                   - **Windows**: Use `keyboard_hotkey` with keys [\"win\", \"r\"], wait, then `keyboard_type` \"cmd\" and press Enter. Or use [\"win\", \"x\"] then \"i\" for PowerShell.\n\
                   - **macOS**: Use Spotlight with [\"meta\", \"space\"], type \"Terminal\", press Enter.\n\
                   - **Linux**: Try [\"ctrl\", \"alt\", \"t\"] to open terminal.\n\
                   - If a terminal is already open, skip this step.\n\
                3. Take a screenshot to verify the terminal is open and ready for input.\n\
                4. Type the command using `keyboard_type` with text=\"{}\".\n\
                5. Press Enter using `keyboard_hotkey` with keys [\"enter\"].\n\
                6. Wait briefly for the command to execute.\n\
                7. Take a screenshot to capture the output.\n\
                8. If the output is long, you may need to scroll up to see all of it.\n\
                \n\
                Tips:\n\
                - For long-running commands, take multiple screenshots to monitor progress.\n\
                - If the command requires elevated privileges, you may need to run as administrator.\n\
                - Use `get_clipboard` after selecting output text (Ctrl+A in terminal, then Ctrl+C) to get the text content.",
                args.command, args.connection_id, args.command
            ),
        )])
    }

    /// Guide for performing a server health check — checking CPU, memory, disk usage, services, and logs.
    #[prompt(name = "server_health_check")]
    async fn server_health_check(
        &self,
        params: Parameters<ConnectionIdArg>,
    ) -> Result<Vec<PromptMessage>, ErrorData> {
        let conn = params.0.connection_id;
        Ok(vec![PromptMessage::new_text(
            PromptMessageRole::User,
            format!(
                "Perform a comprehensive health check on the remote server.\n\
                Connection ID: {conn}\n\
                \n\
                ## Procedure\n\
                \n\
                1. **Take a screenshot** to observe the current desktop state.\n\
                2. **Open a terminal** (Win+R → cmd / PowerShell, or Ctrl+Alt+T on Linux).\n\
                3. **Detect OS type** from the screenshot (Windows / Linux / macOS).\n\
                4. **Run the following checks** (adapt commands to the detected OS):\n\
                \n\
                ### Windows\n\
                ```\n\
                systeminfo | findstr /C:\"OS\" /C:\"Memory\"\n\
                wmic cpu get loadpercentage\n\
                wmic logicaldisk get size,freespace,caption\n\
                tasklist /FI \"STATUS eq running\" | sort /R /+65\n\
                net start\n\
                Get-EventLog -LogName System -Newest 20 -EntryType Error\n\
                ```\n\
                \n\
                ### Linux\n\
                ```\n\
                uname -a\n\
                uptime\n\
                free -h\n\
                df -h\n\
                top -bn1 | head -20\n\
                systemctl --failed\n\
                journalctl -p err --since \"1 hour ago\" --no-pager | tail -30\n\
                ```\n\
                \n\
                5. **For each command**: type it with `keyboard_type`, press Enter, wait, then screenshot the output.\n\
                6. **Copy important output** using Ctrl+A, Ctrl+C in the terminal, then `get_clipboard` to capture text.\n\
                7. **Compile a summary report** with:\n\
                   - OS version and uptime\n\
                   - CPU load (normal / warning / critical)\n\
                   - Memory usage (used / total / percentage)\n\
                   - Disk usage per partition (flag any above 80%)\n\
                   - Top 5 processes by resource usage\n\
                   - Failed services (if any)\n\
                   - Recent error log entries\n\
                   - Overall health verdict: HEALTHY / WARNING / CRITICAL\n\
                \n\
                ## Tips\n\
                - Run one command at a time and verify output before proceeding.\n\
                - If a command fails, try an alternative (e.g. `Get-Process` instead of `tasklist`).\n\
                - For long output, scroll up or use `| more` / `| head` to paginate."
            ),
        )])
    }

    /// Guide for batch operations across multiple remote devices — install software, run scripts, collect status.
    #[prompt(name = "batch_operation")]
    async fn batch_operation(
        &self,
        params: Parameters<MultiDeviceTaskArgs>,
    ) -> Result<Vec<PromptMessage>, ErrorData> {
        let task = params.0.task_description;
        Ok(vec![PromptMessage::new_text(
            PromptMessageRole::User,
            format!(
                "Perform the following batch operation across multiple remote devices: \"{task}\"\n\
                \n\
                ## Workflow\n\
                \n\
                1. **Discover devices**: Call `get_host_info` to get the local device info. If you have a list of device IDs and access codes, proceed to step 2.\n\
                2. **Plan the operation**: Break down \"{task}\" into specific steps for each device.\n\
                3. **Connect to each device** sequentially using `connect_device`. Use `show_window=false` for efficiency during batch jobs.\n\
                4. **For each device**:\n\
                   a. Take a screenshot to verify the connection.\n\
                   b. Execute the required steps (commands, clicks, etc.).\n\
                   c. Take a screenshot to verify success.\n\
                   d. Record the result (success / failure / warnings).\n\
                   e. Disconnect with `disconnect_device` when done.\n\
                5. **Compile a batch report**:\n\
                   - Device ID → Result (success/failure)\n\
                   - Any errors or warnings per device\n\
                   - Summary: X/Y devices completed successfully\n\
                \n\
                ## Best Practices\n\
                - Process devices one at a time to avoid confusion.\n\
                - If a device fails, log the error and continue with the next device.\n\
                - Use `show_window=false` for background automation — no GUI windows will open.\n\
                - For identical operations, prepare the command sequence once, then repeat for each device.\n\
                - After completing all devices, present a summary table to the user.\n\
                \n\
                ## Error Handling\n\
                - If connection fails: skip the device, record \"connection failed\".\n\
                - If a command fails: screenshot the error, try once more, then skip and record.\n\
                - If the operation is destructive (uninstall, delete, format), ask the user for confirmation before proceeding."
            ),
        )])
    }

    /// Guide for diagnosing system issues on the remote desktop — slow performance, crashes, network problems, etc.
    #[prompt(name = "diagnose_system_issue")]
    async fn diagnose_system_issue(
        &self,
        params: Parameters<DiagnoseIssueArgs>,
    ) -> Result<Vec<PromptMessage>, ErrorData> {
        let args = params.0;
        Ok(vec![PromptMessage::new_text(
            PromptMessageRole::User,
            format!(
                "Diagnose the following issue on the remote desktop: \"{}\"\n\
                Connection ID: {}\n\
                \n\
                ## Diagnostic Procedure\n\
                \n\
                1. **Screenshot** the current state of the desktop.\n\
                2. **Gather system info** — open a terminal and run:\n\
                   - **Windows**: `systeminfo`, `tasklist`, `ipconfig /all`, `netstat -an`\n\
                   - **Linux**: `uname -a`, `top -bn1`, `free -h`, `df -h`, `ip addr`, `ss -tlnp`\n\
                3. **Analyze based on symptom type**:\n\
                \n\
                ### Slow Performance\n\
                - Check CPU usage: identify processes using >50% CPU\n\
                - Check memory: look for high usage or swapping\n\
                - Check disk: look for full partitions or high I/O\n\
                - Recommend: kill unnecessary processes, free memory, clear disk space\n\
                \n\
                ### Application Crash\n\
                - Check event logs: `Event Viewer` (Windows) or `journalctl` (Linux)\n\
                - Look for crash dumps or error messages\n\
                - Check if the application can be restarted\n\
                - Check for updates or known issues\n\
                \n\
                ### Network Issues\n\
                - `ping 8.8.8.8` — check basic connectivity\n\
                - `nslookup google.com` — check DNS\n\
                - `tracert` / `traceroute` — identify network path issues\n\
                - Check firewall rules and proxy settings\n\
                \n\
                ### Disk Full\n\
                - Find large files: `dir /s /o-s` (Windows) or `du -sh /* | sort -rh` (Linux)\n\
                - Check temp folders, log files, recycle bin\n\
                - Recommend cleanup actions\n\
                \n\
                4. **Screenshot** all diagnostic output for evidence.\n\
                5. **Compile diagnosis report**:\n\
                   - Problem summary\n\
                   - Root cause (identified or suspected)\n\
                   - Evidence (what commands/observations led to the conclusion)\n\
                   - Recommended fix (with specific steps)\n\
                   - Risk level of the fix\n\
                \n\
                6. **Ask the user** before applying any fix. Present the diagnosis and recommended action, wait for confirmation.\n\
                \n\
                ## Tips\n\
                - Start broad (system overview) then narrow down.\n\
                - Always screenshot before and after any remediation.\n\
                - For destructive actions (killing processes, deleting files), explicitly ask for user confirmation.\n\
                - If unsure, suggest the user consult a professional.",
                args.issue_description, args.connection_id
            ),
        )])
    }

    /// Guide for analyzing and describing the current screen content in detail — OCR, UI understanding, content extraction.
    #[prompt(name = "analyze_screen_content")]
    async fn analyze_screen_content(
        &self,
        params: Parameters<ConnectionIdArg>,
    ) -> Result<Vec<PromptMessage>, ErrorData> {
        let conn = params.0.connection_id;
        Ok(vec![PromptMessage::new_text(
            PromptMessageRole::User,
            format!(
                "Analyze and describe the current screen content of the remote desktop in detail.\n\
                Connection ID: {conn}\n\
                \n\
                ## Procedure\n\
                \n\
                1. **Take a full-resolution screenshot** (do NOT use max_width, we need maximum detail).\n\
                2. **Get screen size** with `get_screen_size` for context.\n\
                3. **Provide a structured analysis**:\n\
                \n\
                ### Desktop Overview\n\
                - What OS is running? (identify from taskbar, dock, or desktop)\n\
                - What applications are open? (window titles, taskbar items)\n\
                - What is the active/focused window?\n\
                \n\
                ### Active Window Content\n\
                - Application name and window title\n\
                - Visible text content (read all text you can see)\n\
                - UI state (dialogs, menus, forms — what's filled in, what's selected)\n\
                - Error messages or notifications visible\n\
                \n\
                ### Key Elements Inventory\n\
                For each important UI element visible, provide:\n\
                - Description (e.g. \"Save button\", \"username text field\", \"error dialog\")\n\
                - Approximate coordinates (x, y)\n\
                - Current state (enabled/disabled, checked/unchecked, text content)\n\
                \n\
                ### Notifications & Alerts\n\
                - System tray notifications\n\
                - Toast messages\n\
                - Dialog boxes or popups\n\
                - Error indicators (red icons, warning triangles)\n\
                \n\
                ### Security Scan\n\
                - Flag any visible sensitive information:\n\
                  - Passwords shown in plaintext\n\
                  - API keys or tokens\n\
                  - Personal data (emails, phone numbers, addresses)\n\
                  - Financial information\n\
                - If sensitive content is found, warn the user immediately.\n\
                \n\
                ## Output Format\n\
                Present the analysis in a clear, structured format. Use headers and bullet points.\n\
                For any text you can read on screen, quote it exactly.\n\
                For UI elements, provide coordinates so the user or AI can interact with them later."
            ),
        )])
    }

    /// Guide for orchestrating a task across multiple remote devices simultaneously.
    #[prompt(name = "multi_device_workflow")]
    async fn multi_device_workflow(
        &self,
        params: Parameters<MultiDeviceTaskArgs>,
    ) -> Result<Vec<PromptMessage>, ErrorData> {
        let task = params.0.task_description;
        Ok(vec![PromptMessage::new_text(
            PromptMessageRole::User,
            format!(
                "Orchestrate the following workflow across multiple remote devices: \"{task}\"\n\
                \n\
                ## Multi-Device Orchestration Framework\n\
                \n\
                ### Phase 1: Planning\n\
                1. Parse the task description to identify:\n\
                   - Which devices are involved (device IDs / access codes)\n\
                   - What operations each device needs\n\
                   - Dependencies between operations (does device B need output from device A?)\n\
                   - The correct execution order\n\
                \n\
                ### Phase 2: Connection\n\
                1. Connect to all required devices using `connect_device`.\n\
                   - Use `show_window=false` for background automation.\n\
                   - Track each `connection_id` with its device identity.\n\
                2. Verify each connection with a screenshot.\n\
                3. Call `get_screen_size` for each device to understand their coordinate spaces.\n\
                \n\
                ### Phase 3: Execution\n\
                Execute operations following the dependency order:\n\
                \n\
                **Independent tasks** (no cross-device dependencies):\n\
                - Execute on each device sequentially.\n\
                - Screenshot → Act → Verify for each step on each device.\n\
                \n\
                **Cross-device tasks** (e.g. copy data from A to B):\n\
                - On source device: select content, copy to clipboard.\n\
                - Use `get_clipboard` to retrieve the data.\n\
                - On target device: use `set_clipboard` to set the data, then paste.\n\
                - Screenshot both devices to verify.\n\
                \n\
                **Coordinated tasks** (e.g. test client-server communication):\n\
                - Set up the server side first.\n\
                - Screenshot to confirm it's ready.\n\
                - Then perform the client side action.\n\
                - Screenshot both sides to verify.\n\
                \n\
                ### Phase 4: Reporting\n\
                Compile a cross-device report:\n\
                - Per-device status: what was done, what succeeded, what failed\n\
                - Cross-device verification: did the overall workflow achieve its goal?\n\
                - Any issues that need manual attention\n\
                \n\
                ## Tips\n\
                - Always include the correct `connection_id` in every tool call — mixing up devices is the most common error.\n\
                - Name or label your connections clearly (e.g. \"conn_server\", \"conn_client\") in your reasoning.\n\
                - If a cross-device operation fails, check both devices for error state.\n\
                - Disconnect all devices when the workflow is complete."
            ),
        )])
    }

    /// Guide for documenting an operational procedure by observing and recording each step performed on the remote desktop.
    #[prompt(name = "document_procedure")]
    async fn document_procedure(
        &self,
        params: Parameters<DocumentProcedureArgs>,
    ) -> Result<Vec<PromptMessage>, ErrorData> {
        let args = params.0;
        Ok(vec![PromptMessage::new_text(
            PromptMessageRole::User,
            format!(
                "Document the following procedure on the remote desktop: \"{}\"\n\
                Connection ID: {}\n\
                \n\
                ## Documentation Workflow\n\
                \n\
                You will observe (or perform) the procedure step-by-step and generate a Standard Operating Procedure (SOP) document.\n\
                \n\
                ### For Each Step\n\
                1. **Screenshot** the screen BEFORE the action.\n\
                2. **Describe** what action is about to be taken and why.\n\
                3. **Perform** the action (click, type, etc.).\n\
                4. **Screenshot** the screen AFTER the action.\n\
                5. **Record** the step in this format:\n\
                   - Step number\n\
                   - Action description (what to do)\n\
                   - Expected result (what should happen)\n\
                   - Actual result (what happened)\n\
                   - Screenshot reference (before/after)\n\
                \n\
                ### SOP Output Format\n\
                \n\
                ```\n\
                # Procedure: {}\n\
                \n\
                ## Prerequisites\n\
                - [List required access, software, credentials]\n\
                \n\
                ## Steps\n\
                \n\
                ### Step 1: [Action Title]\n\
                **Action**: [Detailed description of what to do]\n\
                **Expected Result**: [What should happen]\n\
                **Screenshot**: [Before → After]\n\
                **Notes**: [Any warnings, tips, or edge cases]\n\
                \n\
                ### Step 2: ...\n\
                \n\
                ## Troubleshooting\n\
                - [Common issues and how to resolve them]\n\
                \n\
                ## Summary\n\
                - Total steps: N\n\
                - Estimated time: X minutes\n\
                - Difficulty: Easy / Medium / Hard\n\
                ```\n\
                \n\
                ### Tips\n\
                - Be verbose — someone unfamiliar with the system should be able to follow the SOP.\n\
                - Include exact coordinates or UI element descriptions for click targets.\n\
                - Note any delays or waiting periods between steps.\n\
                - If a step can fail, document the failure mode and recovery action.\n\
                - Use `get_clipboard` to capture exact text output from terminals.",
                args.procedure_name, args.connection_id, args.procedure_name
            ),
        )])
    }
}

#[tool_handler]
#[prompt_handler]
impl ServerHandler for QuickDeskMcpServer {
    fn get_info(&self) -> ServerInfo {
        ServerInfo {
            capabilities: ServerCapabilities::builder()
                .enable_tools()
                .enable_resources()
                .enable_prompts()
                .build(),
            server_info: Implementation {
                name: "quickdesk-mcp".to_string(),
                version: env!("CARGO_PKG_VERSION").to_string(),
                ..Default::default()
            },
            instructions: Some(
                "QuickDesk MCP Server - Control remote desktops via QuickDesk. \
                 Use get_host_info to get the local device credentials, then \
                 connect_device to establish a remote session. After connecting, \
                 use screenshot, mouse_click, keyboard_type, etc. to interact \
                 with the remote desktop. \
                 QuickDesk is a desktop application with a GUI. When you connect \
                 to a device, a remote desktop viewer window is shown by default \
                 so the user can observe your operations in real time. You can \
                 set show_window=false in connect_device for silent batch automation."
                    .to_string(),
            ),
            ..Default::default()
        }
    }

    async fn list_resources(
        &self,
        _request: Option<PaginatedRequestParams>,
        _context: RequestContext<RoleServer>,
    ) -> Result<ListResourcesResult, ErrorData> {
        let mut resources = vec![
            make_resource(
                "quickdesk://host",
                "Host Info",
                "Local device ID, access code, signaling state and connected client count",
            ),
            make_resource(
                "quickdesk://status",
                "System Status",
                "Overall status of host process, client process and signaling server",
            ),
        ];

        if let Ok(v) = self.ws.request("listConnections", json!({})).await {
            if let Some(conns) = v.get("connections").and_then(|c| c.as_array()) {
                for conn in conns {
                    let id = conn
                        .get("connectionId")
                        .and_then(|v| v.as_str())
                        .unwrap_or("unknown");
                    let device_id = conn
                        .get("deviceId")
                        .and_then(|v| v.as_str())
                        .unwrap_or("unknown");
                    resources.push(make_resource(
                        &format!("quickdesk://connection/{id}"),
                        &format!("Connection {id} (device {device_id})"),
                        &format!(
                            "Detailed info for remote connection {id} to device {device_id}"
                        ),
                    ));
                }
            }
        }

        Ok(ListResourcesResult {
            resources,
            next_cursor: None,
            meta: None,
        })
    }

    async fn list_resource_templates(
        &self,
        _request: Option<PaginatedRequestParams>,
        _context: RequestContext<RoleServer>,
    ) -> Result<ListResourceTemplatesResult, ErrorData> {
        Ok(ListResourceTemplatesResult {
            resource_templates: vec![Annotated {
                raw: RawResourceTemplate {
                    uri_template: "quickdesk://connection/{connectionId}".to_string(),
                    name: "Connection Info".to_string(),
                    title: None,
                    description: Some(
                        "Detailed info for a specific remote connection by connection ID"
                            .to_string(),
                    ),
                    mime_type: Some("application/json".to_string()),
                    icons: None,
                },
                annotations: None,
            }],
            next_cursor: None,
            meta: None,
        })
    }

    async fn read_resource(
        &self,
        request: ReadResourceRequestParams,
        _context: RequestContext<RoleServer>,
    ) -> Result<ReadResourceResult, ErrorData> {
        let uri = &request.uri;

        let result = if uri == "quickdesk://host" {
            self.ws.request("getHostInfo", json!({})).await
        } else if uri == "quickdesk://status" {
            self.ws.request("getStatus", json!({})).await
        } else if let Some(conn_id) = uri.strip_prefix("quickdesk://connection/") {
            self.ws
                .request("getConnectionInfo", json!({ "connectionId": conn_id }))
                .await
        } else {
            return Err(ErrorData::invalid_params(
                format!("Unknown resource URI: {uri}"),
                None,
            ));
        };

        match result {
            Ok(v) => {
                let text = serde_json::to_string_pretty(&v).unwrap_or_default();
                Ok(ReadResourceResult {
                    contents: vec![ResourceContents::TextResourceContents {
                        uri: uri.to_string(),
                        mime_type: Some("application/json".to_string()),
                        text,
                        meta: None,
                    }],
                })
            }
            Err(e) => Err(ErrorData::internal_error(e, None)),
        }
    }
}
