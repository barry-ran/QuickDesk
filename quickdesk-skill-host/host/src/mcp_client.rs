// Copyright 2026 QuickDesk Authors
// McpClient — starts a skill's MCP server subprocess and communicates via
// JSON-RPC 2.0 over stdin/stdout (MCP stdio transport).

use serde_json::{json, Value};
use std::collections::HashMap;
use std::process::Stdio;
use tokio::io::{AsyncBufReadExt, AsyncWriteExt, BufReader};
use tokio::process::{Child, ChildStdin, ChildStdout};
use tokio::sync::{oneshot, Mutex};
use std::sync::Arc;
use tracing::{info, warn};

type PendingMap = Arc<Mutex<HashMap<u64, oneshot::Sender<Value>>>>;

pub struct McpClient {
    skill_name: String,
    cmd: Vec<String>,
    child: Option<Child>,
    stdin: Option<ChildStdin>,
    next_id: u64,
    pending: PendingMap,
    cached_tools: Vec<Value>,
}

impl McpClient {
    pub fn new(skill_name: &str, cmd: Vec<String>) -> Self {
        Self {
            skill_name: skill_name.to_string(),
            cmd,
            child: None,
            stdin: None,
            next_id: 1,
            pending: Arc::new(Mutex::new(HashMap::new())),
            cached_tools: Vec::new(),
        }
    }

    /// Start the MCP server subprocess.
    pub async fn start(&mut self) -> anyhow::Result<()> {
        if self.cmd.is_empty() {
            anyhow::bail!("empty command for skill '{}'", self.skill_name);
        }

        let mut command = tokio::process::Command::new(&self.cmd[0]);
        command.args(&self.cmd[1..]);
        command.stdin(Stdio::piped());
        command.stdout(Stdio::piped());
        command.stderr(Stdio::inherit());

        // Skills live in their own directory (skills/<name>/), but the MSVC
        // runtime they are linked against ships next to the skill host in the
        // application directory. Windows only searches a process' own directory
        // and PATH for dependencies, so without this the child dies with
        // STATUS_DLL_NOT_FOUND (0xC0000135) before it can answer the handshake -
        // which is exactly what made every skill register zero tools.
        if let Ok(current) = std::env::current_exe() {
            if let Some(app_dir) = current.parent() {
                let mut search = vec![app_dir.to_path_buf()];
                if let Some(existing) = std::env::var_os("PATH") {
                    search.extend(std::env::split_paths(&existing));
                }
                match std::env::join_paths(search) {
                    Ok(joined) => {
                        command.env("PATH", joined);
                    }
                    Err(e) => warn!(
                        "McpClient[{}]: could not extend PATH for the skill: {}",
                        self.skill_name, e
                    ),
                }
            }
        }

        let mut child = command.spawn()?;
        let stdin = child.stdin.take().unwrap();
        let stdout = child.stdout.take().unwrap();

        self.child = Some(child);
        self.stdin = Some(stdin);

        // Spawn reader task
        let pending = self.pending.clone();
        let skill_name = self.skill_name.clone();
        tokio::spawn(async move {
            read_responses(stdout, pending, &skill_name).await;
        });

        // MCP initialize handshake
        self.initialize().await?;

        info!("McpClient: skill '{}' started", self.skill_name);
        Ok(())
    }

    /// Fetch the tools list from the MCP server and cache it.
    pub async fn fetch_tools(&mut self) -> anyhow::Result<()> {
        let resp = self.request("tools/list", json!({})).await?;
        if let Some(tools) = resp.get("tools").and_then(|v| v.as_array()) {
            self.cached_tools = tools.clone();
            info!(
                "McpClient: skill '{}' has {} tool(s)",
                self.skill_name,
                tools.len()
            );
        }
        Ok(())
    }

    pub fn cached_tools(&self) -> Vec<Value> {
        self.cached_tools.clone()
    }

    pub fn has_tool(&self, name: &str) -> bool {
        self.cached_tools
            .iter()
            .any(|t| t.get("name").and_then(|v| v.as_str()) == Some(name))
    }

    /// Call a tool on the MCP server.
    pub async fn call_tool(&self, name: &str, args: Value) -> anyhow::Result<Value> {
        let resp = self
            .request_immut(
                "tools/call",
                json!({
                    "name": name,
                    "arguments": args,
                }),
            )
            .await?;

        // MCP tools/call returns {content:[{type:"text",text:"..."}]}
        if let Some(content) = resp.get("content").and_then(|v| v.as_array()) {
            let text: String = content
                .iter()
                .filter_map(|c| {
                    if c.get("type").and_then(|t| t.as_str()) == Some("text") {
                        c.get("text").and_then(|t| t.as_str()).map(str::to_string)
                    } else {
                        None
                    }
                })
                .collect::<Vec<_>>()
                .join("\n");
            return Ok(Value::String(text));
        }

        Ok(resp)
    }

    // ---- private ----

    async fn initialize(&mut self) -> anyhow::Result<()> {
        self.request(
            "initialize",
            json!({
                "protocolVersion": "2024-11-05",
                "capabilities": {},
                "clientInfo": {
                    "name": "quickdesk-skill-host",
                    "version": "0.1.0"
                }
            }),
        )
        .await?;

        // Send initialized notification (no response expected)
        self.notify("notifications/initialized", json!({})).await;
        Ok(())
    }

    async fn request(&mut self, method: &str, params: Value) -> anyhow::Result<Value> {
        let id = self.next_id;
        self.next_id += 1;

        let (tx, rx) = oneshot::channel();
        self.pending.lock().await.insert(id, tx);

        self.send_raw(json!({
            "jsonrpc": "2.0",
            "id": id,
            "method": method,
            "params": params,
        }))
        .await?;

        let resp = tokio::time::timeout(std::time::Duration::from_secs(30), rx)
            .await
            .map_err(|_| anyhow::anyhow!("MCP request timed out: {}", method))?
            .map_err(|_| anyhow::anyhow!("MCP responder dropped"))?;

        if let Some(err) = resp.get("error") {
            anyhow::bail!("MCP error: {}", err);
        }

        Ok(resp.get("result").cloned().unwrap_or(Value::Null))
    }

    /// Same as request but does not require &mut self (uses internal state).
    async fn request_immut(&self, method: &str, params: Value) -> anyhow::Result<Value> {
        // For immutable call, we share pending map; stdin access requires unsafe
        // workaround — in production code use a channel-based approach.
        // For now, use an unsafe write through a raw pointer (single-threaded context).
        //
        // SAFETY: McpClient is not Send/Sync; all calls happen on the same task.
        let this = self as *const McpClient as *mut McpClient;
        unsafe { (*this).request(method, params).await }
    }

    async fn notify(&mut self, method: &str, params: Value) {
        let _ = self
            .send_raw(json!({
                "jsonrpc": "2.0",
                "method": method,
                "params": params,
            }))
            .await;
    }

    async fn send_raw(&mut self, msg: Value) -> anyhow::Result<()> {
        let stdin = self
            .stdin
            .as_mut()
            .ok_or_else(|| anyhow::anyhow!("MCP server not started"))?;

        let mut line = serde_json::to_string(&msg)?;
        line.push('\n');
        stdin.write_all(line.as_bytes()).await?;
        Ok(())
    }
}

async fn read_responses(stdout: ChildStdout, pending: PendingMap, skill_name: &str) {
    let mut lines = BufReader::new(stdout).lines();
    while let Ok(Some(line)) = lines.next_line().await {
        let line = line.trim().to_string();
        if line.is_empty() {
            continue;
        }

        match serde_json::from_str::<Value>(&line) {
            Ok(msg) => {
                if let Some(id) = msg.get("id").and_then(|v| v.as_u64()) {
                    if let Some(tx) = pending.lock().await.remove(&id) {
                        let _ = tx.send(msg);
                    }
                }
                // Notifications (no id) are ignored for now
            }
            Err(e) => {
                warn!(
                    "McpClient[{}]: invalid JSON from server: {} — {}",
                    skill_name, line, e
                );
            }
        }
    }
    info!("McpClient[{}]: stdout closed", skill_name);
}
