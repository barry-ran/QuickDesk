use futures_util::{SinkExt, StreamExt};
use serde_json::Value;
use std::collections::HashMap;
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::{Mutex, oneshot};
use tokio_tungstenite::{connect_async, tungstenite::Message};

use crate::event_bus::{Event, EventBus};

type PendingMap = HashMap<String, oneshot::Sender<Result<Value, String>>>;

/// Live connection state — replaced atomically on reconnect.
struct WsInner {
    sender: futures_util::stream::SplitSink<
        tokio_tungstenite::WebSocketStream<
            tokio_tungstenite::MaybeTlsStream<tokio::net::TcpStream>,
        >,
        Message,
    >,
    pending: Arc<Mutex<PendingMap>>,
    req_counter: u64,
}

/// Auto-reconnecting WebSocket client.
///
/// On every `request()` call, if the underlying connection is gone
/// (dropped by the remote or due to a send error) it transparently
/// reconnects before retrying the request once.
#[derive(Clone)]
pub struct WsClient {
    url: Arc<String>,
    token: Arc<Option<String>>,
    /// `None` means the connection is currently down.
    inner: Arc<Mutex<Option<WsInner>>>,
    event_bus: EventBus,
}

impl WsClient {
    pub async fn connect(url: &str, token: Option<&str>, event_bus: EventBus) -> Result<Self, String> {
        let client = Self {
            url: Arc::new(url.to_string()),
            token: Arc::new(token.map(str::to_string)),
            inner: Arc::new(Mutex::new(None)),
            event_bus,
        };
        client.do_connect().await?;
        Ok(client)
    }

    /// Establish (or re-establish) the WebSocket connection.
    /// Must be called with `self.inner` lock NOT held.
    async fn do_connect(&self) -> Result<(), String> {
        let (ws_stream, _) = connect_async(self.url.as_str())
            .await
            .map_err(|e| format!("WebSocket connect failed: {e}"))?;

        let (write, read) = ws_stream.split();
        let pending: Arc<Mutex<PendingMap>> = Arc::new(Mutex::new(HashMap::new()));

        // Spawn the reader; it clears `inner` when the connection closes.
        tokio::spawn(Self::reader_loop(
            read,
            pending.clone(),
            self.inner.clone(),
            self.event_bus.clone(),
        ));

        let mut guard = self.inner.lock().await;
        *guard = Some(WsInner { sender: write, pending, req_counter: 0 });
        drop(guard);

        // Authenticate if a token was provided.
        if let Some(ref tok) = *self.token {
            self.authenticate(tok).await?;
        }

        tracing::info!("WsClient connected to {}", self.url);
        Ok(())
    }

    async fn authenticate(&self, token: &str) -> Result<(), String> {
        let resp = self
            .try_request("auth", serde_json::json!({ "token": token }), Duration::from_secs(10))
            .await?;
        if resp.get("authenticated").and_then(|v| v.as_bool()) == Some(true) {
            Ok(())
        } else {
            Err("Authentication failed".to_string())
        }
    }

    pub async fn request(&self, method: &str, params: Value) -> Result<Value, String> {
        self.request_with_timeout(method, params, Duration::from_secs(30)).await
    }

    pub async fn request_with_timeout(&self, method: &str, params: Value, timeout: Duration) -> Result<Value, String> {
        match self.try_request(method, params.clone(), timeout).await {
            Ok(v) => Ok(v),
            Err(e) => {
                tracing::warn!("WsClient request '{}' failed ({}), reconnecting…", method, e);
                self.do_connect().await?;
                self.try_request(method, params, timeout).await
            }
        }
    }

    /// Send a mutating request once with the default budget. A timed-out call
    /// may still execute on the remote, so it is never resent.
    pub async fn request_once(&self, method: &str, params: Value) -> Result<Value, String> {
        self.request_no_retry(method, params, Duration::from_secs(30)).await
    }

    /// Same as `request_with_timeout`, but never resends the request after it
    /// was handed to the socket. A timed-out mutating call (skill_exec,
    /// key presses, clicks) must not be retried: the first attempt may still
    /// execute on the remote, so a silent retry can run it twice. Only
    /// reconnect-and-retry when the request was provably never sent.
    pub async fn request_no_retry(&self, method: &str, params: Value, timeout: Duration) -> Result<Value, String> {
        match self.try_request(method, params.clone(), timeout).await {
            Ok(v) => Ok(v),
            Err(e) if e == "Not connected" => {
                tracing::warn!("WsClient '{}' not connected, reconnecting…", method);
                self.do_connect().await?;
                self.try_request(method, params, timeout).await
            }
            Err(e) => Err(e),
        }
    }

    async fn try_request(&self, method: &str, params: Value, timeout: Duration) -> Result<Value, String> {
        let (id, rx) = {
            let mut guard = self.inner.lock().await;
            let inner = guard.as_mut().ok_or("Not connected")?;

            inner.req_counter += 1;
            let id = format!("req_{}", inner.req_counter);

            let (tx, rx) = oneshot::channel();
            inner.pending.lock().await.insert(id.clone(), tx);

            let msg = serde_json::json!({ "id": id, "method": method, "params": params });
            let text = serde_json::to_string(&msg).unwrap();

            inner
                .sender
                .send(Message::Text(text.into()))
                .await
                .map_err(|e| {
                    format!("WebSocket send failed: {e}")
                })?;

            (id, rx)
        };
        let _ = id;

        let secs = timeout.as_secs();
        match tokio::time::timeout(timeout, rx).await {
            Ok(Ok(v)) => v,
            Ok(Err(_)) => Err("Response channel closed".to_string()),
            Err(_) => Err(format!("WebSocket request timed out after {secs}s")),
        }
    }

    async fn reader_loop(
        mut read: futures_util::stream::SplitStream<
            tokio_tungstenite::WebSocketStream<
                tokio_tungstenite::MaybeTlsStream<tokio::net::TcpStream>,
            >,
        >,
        pending: Arc<Mutex<PendingMap>>,
        inner: Arc<Mutex<Option<WsInner>>>,
        event_bus: EventBus,
    ) {
        while let Some(msg) = read.next().await {
            let msg = match msg {
                Ok(Message::Text(t)) => t,
                Ok(Message::Close(_)) => break,
                Err(e) => {
                    tracing::warn!("WsClient reader error: {e}");
                    break;
                }
                _ => continue,
            };

            let parsed: Value = match serde_json::from_str(&msg) {
                Ok(v) => v,
                Err(_) => continue,
            };

            if let Some(id) = parsed.get("id").and_then(|v| v.as_str()) {
                let mut map = pending.lock().await;
                if let Some(tx) = map.remove(id) {
                    if let Some(err) = parsed.get("error") {
                        let msg = err
                            .get("message")
                            .and_then(|v| v.as_str())
                            .unwrap_or("Unknown error");
                        let _ = tx.send(Err(msg.to_string()));
                    } else if let Some(result) = parsed.get("result") {
                        let _ = tx.send(Ok(result.clone()));
                    } else {
                        let _ = tx.send(Ok(parsed));
                    }
                }
            } else if let Some(event_name) = parsed.get("event").and_then(|v| v.as_str()) {
                let data = parsed.get("data").cloned().unwrap_or(Value::Null);
                let timestamp = std::time::SystemTime::now()
                    .duration_since(std::time::UNIX_EPOCH)
                    .unwrap_or_default()
                    .as_millis() as u64;

                tracing::debug!("Received event: {} data={}", event_name, data);
                event_bus.publish(Event { event: event_name.to_string(), data, timestamp }).await;
            }
        }

        // Connection closed — mark inner as None so the next request triggers reconnect.
        tracing::info!("WsClient connection closed, will reconnect on next request");
        // Fail any in-flight requests.
        let mut map = pending.lock().await;
        for (_, tx) in map.drain() {
            let _ = tx.send(Err("Connection closed".to_string()));
        }
        drop(map);
        *inner.lock().await = None;
    }

    pub fn event_bus(&self) -> &EventBus {
        &self.event_bus
    }
}
