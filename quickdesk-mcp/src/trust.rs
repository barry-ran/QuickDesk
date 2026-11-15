use chrono::Utc;
use rusqlite::{params, Connection};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::collections::HashMap;
use std::path::PathBuf;
use std::sync::{Arc, Mutex};
use tokio::sync::oneshot;
use tracing::{info, warn};
use uuid::Uuid;

// ---- Risk Classification ----

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum RiskLevel {
    Safe,
    Low,
    Medium,
    High,
    Critical,
}

impl RiskLevel {
    pub fn requires_confirmation(&self, policy: &TrustPolicy) -> bool {
        match self {
            RiskLevel::Safe | RiskLevel::Low => false,
            RiskLevel::Medium => policy.confirm_medium,
            RiskLevel::High | RiskLevel::Critical => true,
        }
    }

    pub fn is_blocked(&self, policy: &TrustPolicy) -> bool {
        match self {
            RiskLevel::Critical => policy.block_critical,
            _ => false,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskAssessment {
    pub tool_name: String,
    pub risk_level: RiskLevel,
    pub reasons: Vec<String>,
    pub recommendation: String,
}

// ---- Trust Policy ----

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TrustPolicy {
    pub confirm_medium: bool,
    pub block_critical: bool,
    pub auto_approve_tools: Vec<String>,
    pub blocked_tools: Vec<String>,
    pub dangerous_patterns: Vec<DangerousPattern>,
    pub max_batch_size: u32,
}

impl Default for TrustPolicy {
    fn default() -> Self {
        Self {
            confirm_medium: false,
            block_critical: true,
            auto_approve_tools: vec![
                "screenshot".to_string(),
                "getUiState".to_string(),
                "get_system_info".to_string(),
                "list_processes".to_string(),
                "list_directory".to_string(),
                "get_file_info".to_string(),
                "read_file".to_string(),
                "getScreenText".to_string(),
                "getHostInfo".to_string(),
                "getStatus".to_string(),
            ],
            blocked_tools: vec![],
            dangerous_patterns: vec![
                DangerousPattern {
                    tool: "run_command".to_string(),
                    arg_key: "command".to_string(),
                    patterns: vec![
                        "rm -rf".to_string(),
                        "del /s /q".to_string(),
                        "format ".to_string(),
                        "mkfs".to_string(),
                        "dd if=".to_string(),
                        "shutdown".to_string(),
                        "reboot".to_string(),
                        "reg delete".to_string(),
                        "net user".to_string(),
                    ],
                },
                DangerousPattern {
                    tool: "write_file".to_string(),
                    arg_key: "path".to_string(),
                    patterns: vec![
                        "/etc/".to_string(),
                        "C:\\Windows\\System32".to_string(),
                        ".bashrc".to_string(),
                        ".profile".to_string(),
                        "hosts".to_string(),
                    ],
                },
            ],
            max_batch_size: 20,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DangerousPattern {
    pub tool: String,
    pub arg_key: String,
    pub patterns: Vec<String>,
}

// ---- Confirmation ----

#[derive(Debug, Clone, Serialize, Deserialize)]
#[allow(dead_code)]
pub struct ConfirmationRequest {
    pub id: String,
    pub device_id: String,
    pub tool_name: String,
    pub arguments: Value,
    pub risk: RiskAssessment,
    pub timeout_secs: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConfirmationResponse {
    pub id: String,
    pub approved: bool,
    pub reason: String,
}

// ---- Audit Log ----

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditEntry {
    pub id: i64,
    pub timestamp: String,
    pub device_id: String,
    pub tool_name: String,
    pub arguments_summary: String,
    pub risk_level: String,
    pub action: String,
    pub outcome: String,
    pub user_decision: String,
    #[serde(default)]
    pub duration_ms: i64,
}

// ---- Emergency Stop State ----

pub struct EmergencyState {
    active: bool,
    activated_at: Option<String>,
    reason: String,
}

// ---- Trust Engine ----

pub struct TrustEngine {
    policy: Arc<Mutex<TrustPolicy>>,
    pending_confirmations: Arc<Mutex<HashMap<String, oneshot::Sender<ConfirmationResponse>>>>,
    emergency: Arc<Mutex<EmergencyState>>,
    audit_db: Arc<Mutex<Connection>>,
}

impl TrustEngine {
    pub fn new() -> Result<Self, String> {
        let db_path = Self::db_path();
        if let Some(parent) = db_path.parent() {
            std::fs::create_dir_all(parent)
                .map_err(|e| format!("cannot create trust dir: {e}"))?;
        }

        let conn = Connection::open(&db_path)
            .map_err(|e| format!("cannot open audit db: {e}"))?;
        conn.execute_batch("PRAGMA journal_mode=WAL;")
            .map_err(|e| format!("pragma: {e}"))?;
        Self::init_tables(&conn)?;
        Self::migrate_audit_device_column(&conn)?;
        Self::migrate_audit_duration_column(&conn)?;

        let policy = Self::load_or_init_policy(&conn);

        info!("TrustEngine initialized at {}", db_path.display());
        Ok(Self {
            policy: Arc::new(Mutex::new(policy)),
            pending_confirmations: Arc::new(Mutex::new(HashMap::new())),
            emergency: Arc::new(Mutex::new(EmergencyState {
                active: false,
                activated_at: None,
                reason: String::new(),
            })),
            audit_db: Arc::new(Mutex::new(conn)),
        })
    }

    fn db_path() -> PathBuf {
        let base = dirs_data_dir().unwrap_or_else(|| PathBuf::from("."));
        base.join(".quickdesk").join("trust").join("audit.db")
    }

    fn init_tables(conn: &Connection) -> Result<(), String> {
        conn.execute_batch(
            "CREATE TABLE IF NOT EXISTS audit_log (
                id              INTEGER PRIMARY KEY AUTOINCREMENT,
                timestamp       TEXT NOT NULL,
                device_id       TEXT NOT NULL,
                tool_name       TEXT NOT NULL,
                arguments_summary TEXT DEFAULT '',
                risk_level      TEXT NOT NULL,
                action          TEXT NOT NULL,
                outcome         TEXT DEFAULT '',
                user_decision   TEXT DEFAULT '',
                duration_ms     INTEGER DEFAULT 0
            );
            CREATE INDEX IF NOT EXISTS idx_audit_time ON audit_log(timestamp);

            CREATE TABLE IF NOT EXISTS trust_policy (
                id      INTEGER PRIMARY KEY CHECK (id = 1),
                policy  TEXT NOT NULL
            );",
        )
        .map_err(|e| format!("init audit tables: {e}"))
    }

    /// Renames legacy `connection_id` column to `device_id` for existing databases.
    fn migrate_audit_device_column(conn: &Connection) -> Result<(), String> {
        let mut stmt = conn
            .prepare("SELECT name FROM pragma_table_info('audit_log')")
            .map_err(|e| format!("pragma_table_info audit_log: {e}"))?;
        let cols: Vec<String> = stmt
            .query_map([], |row| row.get(0))
            .map_err(|e| format!("read audit_log columns: {e}"))?
            .filter_map(|r| r.ok())
            .collect();

        if cols.iter().any(|c| c == "connection_id") && !cols.iter().any(|c| c == "device_id") {
            conn.execute(
                "ALTER TABLE audit_log RENAME COLUMN connection_id TO device_id",
                [],
            )
            .map_err(|e| format!("migrate audit_log column: {e}"))?;
            let _ = conn.execute("DROP INDEX IF EXISTS idx_audit_conn", []);
        }

        conn.execute(
            "CREATE INDEX IF NOT EXISTS idx_audit_device ON audit_log(device_id)",
            [],
        )
        .map_err(|e| format!("create idx_audit_device: {e}"))?;

        Ok(())
    }

    fn migrate_audit_duration_column(conn: &Connection) -> Result<(), String> {
        let mut stmt = conn
            .prepare("SELECT name FROM pragma_table_info('audit_log')")
            .map_err(|e| format!("pragma_table_info audit_log: {e}"))?;
        let cols: Vec<String> = stmt
            .query_map([], |row| row.get(0))
            .map_err(|e| format!("read audit_log columns: {e}"))?
            .filter_map(|r| r.ok())
            .collect();

        if !cols.iter().any(|c| c == "duration_ms") {
            conn.execute(
                "ALTER TABLE audit_log ADD COLUMN duration_ms INTEGER DEFAULT 0",
                [],
            )
            .map_err(|e| format!("migrate audit_log duration_ms: {e}"))?;
        }
        Ok(())
    }

    fn load_or_init_policy(conn: &Connection) -> TrustPolicy {
        let stored: Option<String> = conn
            .query_row("SELECT policy FROM trust_policy WHERE id=1", [], |row| row.get(0))
            .ok();

        match stored {
            Some(json) => serde_json::from_str(&json).unwrap_or_default(),
            None => {
                // Persist the default on first run so get_trust_policy, the
                // database, and later edits agree instead of the UI showing
                // defaults that were never saved.
                let policy = TrustPolicy::default();
                if let Ok(json) = serde_json::to_string(&policy) {
                    let _ = conn.execute(
                        "INSERT OR IGNORE INTO trust_policy (id, policy) VALUES (1, ?1)",
                        params![json],
                    );
                }
                policy
            }
        }
    }

    // ---- Risk Assessment ----

    pub fn assess_risk(&self, tool_name: &str, arguments: &Value) -> RiskAssessment {
        let policy = self.policy.lock().unwrap();

        if policy.auto_approve_tools.contains(&tool_name.to_string()) {
            return RiskAssessment {
                tool_name: tool_name.to_string(),
                risk_level: RiskLevel::Safe,
                reasons: vec!["auto-approved tool".to_string()],
                recommendation: "proceed".to_string(),
            };
        }

        if policy.blocked_tools.contains(&tool_name.to_string()) {
            return RiskAssessment {
                tool_name: tool_name.to_string(),
                risk_level: RiskLevel::Critical,
                reasons: vec!["tool is blocked by policy".to_string()],
                recommendation: "block".to_string(),
            };
        }

        let mut risk_level = self.base_risk(tool_name);
        let mut reasons = Vec::new();

        for pattern in &policy.dangerous_patterns {
            if pattern.tool == tool_name {
                if let Some(arg_val) = arguments
                    .get(&pattern.arg_key)
                    .and_then(|v| v.as_str())
                {
                    let lower = arg_val.to_lowercase();
                    for p in &pattern.patterns {
                        if lower.contains(&p.to_lowercase()) {
                            risk_level = RiskLevel::High;
                            reasons.push(format!(
                                "argument '{}' matches dangerous pattern '{}'",
                                pattern.arg_key, p
                            ));
                        }
                    }
                }
            }
        }

        if reasons.is_empty() {
            reasons.push(format!("base risk for tool '{tool_name}'"));
        }

        let recommendation = if risk_level.is_blocked(&policy) {
            "block"
        } else if risk_level.requires_confirmation(&policy) {
            "confirm"
        } else {
            "proceed"
        }
        .to_string();

        RiskAssessment {
            tool_name: tool_name.to_string(),
            risk_level,
            reasons,
            recommendation,
        }
    }

    fn base_risk(&self, tool_name: &str) -> RiskLevel {
        match tool_name {
            "screenshot" | "getUiState" | "getScreenText" | "findElement"
            | "getHostInfo" | "getStatus" | "getPerformanceStats"
            | "get_system_info" | "list_processes" | "list_directory"
            | "get_file_info" | "read_file" | "skill_list_tools"
            | "getClipboard" | "wait_for_event" => RiskLevel::Safe,

            "clickText" | "typeText" | "pressKey" | "scrollScreen"
            | "mouseClick" | "mouseDrag" | "setClipboard"
            | "write_file" | "create_directory" | "move_file" => RiskLevel::Medium,

            "run_command" | "sendAction" | "skill_exec"
            | "startFileUpload" => RiskLevel::High,

            _ => RiskLevel::Low,
        }
    }

    // ---- Emergency Stop ----

    pub fn is_emergency_active(&self) -> bool {
        self.emergency.lock().unwrap().active
    }

    pub fn activate_emergency(&self, reason: &str) {
        let mut e = self.emergency.lock().unwrap();
        e.active = true;
        e.activated_at = Some(Utc::now().to_rfc3339());
        e.reason = reason.to_string();
        warn!("EMERGENCY STOP activated: {reason}");

        let mut pending = self.pending_confirmations.lock().unwrap();
        for (id, tx) in pending.drain() {
            let _ = tx.send(ConfirmationResponse {
                id,
                approved: false,
                reason: "emergency stop".to_string(),
            });
        }
    }

    pub fn deactivate_emergency(&self) {
        let mut e = self.emergency.lock().unwrap();
        e.active = false;
        e.reason.clear();
        e.activated_at = None;
        info!("Emergency stop deactivated");
    }

    pub fn emergency_status(&self) -> Value {
        let e = self.emergency.lock().unwrap();
        serde_json::json!({
            "active": e.active,
            "activated_at": e.activated_at,
            "reason": e.reason,
        })
    }

    // ---- Confirmation Flow ----

    #[allow(dead_code)]
    pub fn create_confirmation(
        &self,
        device_id: &str,
        tool_name: &str,
        arguments: &Value,
        risk: &RiskAssessment,
    ) -> (ConfirmationRequest, oneshot::Receiver<ConfirmationResponse>) {
        let id = Uuid::new_v4().to_string();
        let (tx, rx) = oneshot::channel();

        let req = ConfirmationRequest {
            id: id.clone(),
            device_id: device_id.to_string(),
            tool_name: tool_name.to_string(),
            arguments: arguments.clone(),
            risk: risk.clone(),
            timeout_secs: 60,
        };

        self.pending_confirmations.lock().unwrap().insert(id, tx);
        (req, rx)
    }

    pub fn resolve_confirmation(&self, id: &str, approved: bool, reason: &str) -> bool {
        let mut pending = self.pending_confirmations.lock().unwrap();
        if let Some(tx) = pending.remove(id) {
            let _ = tx.send(ConfirmationResponse {
                id: id.to_string(),
                approved,
                reason: reason.to_string(),
            });
            true
        } else {
            false
        }
    }

    // ---- Audit Logging ----

    pub fn log_audit(
        &self,
        device_id: &str,
        tool_name: &str,
        arguments: &Value,
        risk_level: RiskLevel,
        action: &str,
        outcome: &str,
        user_decision: &str,
    ) {
        let now = Utc::now().to_rfc3339();
        let args_summary = summarize_args(arguments);
        let level_str = serde_json::to_string(&risk_level).unwrap_or_default();
        let level_str = level_str.trim_matches('"');

        let db = self.audit_db.lock().unwrap();
        let _ = db.execute(
            "INSERT INTO audit_log
                (timestamp, device_id, tool_name, arguments_summary,
                 risk_level, action, outcome, user_decision, duration_ms)
             VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9)",
            params![now, device_id, tool_name, args_summary, level_str, action, outcome, user_decision, 0],
        );
    }

    /// Audit entry with the elapsed time of the operation. Transport-level
    /// failures are recorded with outcome="transport error: …" so they are
    /// distinguishable from the tool's own failure messages.
    #[allow(clippy::too_many_arguments)]
    pub fn log_audit_ext(
        &self,
        device_id: &str,
        tool_name: &str,
        arguments: &Value,
        risk_level: RiskLevel,
        action: &str,
        outcome: &str,
        user_decision: &str,
        duration_ms: i64,
    ) {
        let now = Utc::now().to_rfc3339();
        let args_summary = summarize_args(arguments);
        let level_str = serde_json::to_string(&risk_level).unwrap_or_default();
        let level_str = level_str.trim_matches('"');

        let db = self.audit_db.lock().unwrap();
        let _ = db.execute(
            "INSERT INTO audit_log
                (timestamp, device_id, tool_name, arguments_summary,
                 risk_level, action, outcome, user_decision, duration_ms)
             VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9)",
            params![now, device_id, tool_name, args_summary, level_str, action, outcome, user_decision, duration_ms],
        );
    }

    pub fn get_audit_log(
        &self,
        device_id: Option<&str>,
        limit: i64,
    ) -> Vec<AuditEntry> {
        let db = self.audit_db.lock().unwrap();
        let (sql, conn_param) = if let Some(did) = device_id {
            (
                "SELECT id, timestamp, device_id, tool_name, arguments_summary,
                        risk_level, action, outcome, user_decision, duration_ms
                 FROM audit_log WHERE device_id=?1
                 ORDER BY timestamp DESC LIMIT ?2".to_string(),
                Some(did.to_string()),
            )
        } else {
            (
                "SELECT id, timestamp, device_id, tool_name, arguments_summary,
                        risk_level, action, outcome, user_decision, duration_ms
                 FROM audit_log ORDER BY timestamp DESC LIMIT ?1".to_string(),
                None,
            )
        };

        if let Some(did) = conn_param {
            let mut stmt = match db.prepare(&sql) {
                Ok(s) => s,
                Err(_) => return vec![],
            };
            stmt.query_map(params![did, limit], Self::map_audit_row)
                .ok()
                .map(|r| r.filter_map(|x| x.ok()).collect())
                .unwrap_or_default()
        } else {
            let mut stmt = match db.prepare(&sql) {
                Ok(s) => s,
                Err(_) => return vec![],
            };
            stmt.query_map(params![limit], Self::map_audit_row)
                .ok()
                .map(|r| r.filter_map(|x| x.ok()).collect())
                .unwrap_or_default()
        }
    }

    fn map_audit_row(row: &rusqlite::Row) -> rusqlite::Result<AuditEntry> {
        Ok(AuditEntry {
            id: row.get(0)?,
            timestamp: row.get(1)?,
            device_id: row.get(2)?,
            tool_name: row.get(3)?,
            arguments_summary: row.get(4)?,
            risk_level: row.get(5)?,
            action: row.get(6)?,
            outcome: row.get(7)?,
            user_decision: row.get(8)?,
            duration_ms: row.get(9).unwrap_or(0),
        })
    }

    // ---- Policy Management ----

    pub fn get_policy(&self) -> TrustPolicy {
        self.policy.lock().unwrap().clone()
    }

    pub fn update_policy(&self, new_policy: TrustPolicy) -> Result<(), String> {
        let json = serde_json::to_string(&new_policy)
            .map_err(|e| format!("serialize policy: {e}"))?;

        let db = self.audit_db.lock().unwrap();
        db.execute(
            "INSERT OR REPLACE INTO trust_policy (id, policy) VALUES (1, ?1)",
            params![json],
        )
        .map_err(|e| format!("save policy: {e}"))?;

        *self.policy.lock().unwrap() = new_policy;
        info!("Trust policy updated");
        Ok(())
    }
}

fn summarize_args(args: &Value) -> String {
    let s = serde_json::to_string(args).unwrap_or_default();
    if s.len() > 256 {
        format!("{}...", &s[..256])
    } else {
        s
    }
}

fn dirs_data_dir() -> Option<PathBuf> {
    #[cfg(target_os = "windows")]
    {
        std::env::var("USERPROFILE").ok().map(PathBuf::from)
    }
    #[cfg(not(target_os = "windows"))]
    {
        std::env::var("HOME").ok().map(PathBuf::from)
    }
}
