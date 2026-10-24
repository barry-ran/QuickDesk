use chrono::Utc;
use rusqlite::{params, Connection};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::path::PathBuf;
use std::sync::{Arc, Mutex};
use tracing::info;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeviceProfile {
    pub device_id: String,
    pub os_name: String,
    pub os_version: String,
    pub hostname: String,
    pub cpu_model: String,
    pub memory_total_mb: i64,
    pub disk_total_gb: i64,
    pub resolution: String,
    pub known_apps: Vec<String>,
    pub notes: String,
    pub first_seen: String,
    pub last_seen: String,
    pub connection_count: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OperationRecord {
    pub id: i64,
    pub session_id: String,
    pub device_id: String,
    pub timestamp: String,
    pub tool_name: String,
    pub arguments: Value,
    pub result_summary: String,
    pub success: bool,
    pub duration_ms: i64,
    pub error_message: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FailureRecord {
    pub id: i64,
    pub device_id: String,
    pub tool_name: String,
    pub arguments_pattern: String,
    pub error_category: String,
    pub error_message: String,
    pub occurrence_count: i64,
    pub last_occurred: String,
    pub resolution: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeviceSummary {
    pub device_id: String,
    pub total_operations: i64,
    pub success_rate: f64,
    pub most_used_tools: Vec<ToolUsage>,
    pub common_failures: Vec<FailureRecord>,
    pub sessions: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ToolUsage {
    pub tool: String,
    pub count: i64,
}

#[derive(Clone)]
pub struct MemoryStore {
    db: Arc<Mutex<Connection>>,
}

impl MemoryStore {
    pub fn new() -> Result<Self, String> {
        let db_path = Self::db_path();
        Self::open_at(db_path)
    }

    // Opens a store at a caller-chosen path. Production goes through `new`,
    // which resolves the platform data directory; this variant exists so tests
    // can use a throwaway file instead of the real one on the developer's disk.
    fn open_at(db_path: PathBuf) -> Result<Self, String> {
        if let Some(parent) = db_path.parent() {
            std::fs::create_dir_all(parent)
                .map_err(|e| format!("cannot create memory dir: {e}"))?;
        }

        let conn = Connection::open(&db_path)
            .map_err(|e| format!("cannot open memory db: {e}"))?;

        conn.execute_batch("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;")
            .map_err(|e| format!("pragma: {e}"))?;

        Self::init_tables(&conn)?;

        info!("MemoryStore opened at {}", db_path.display());
        Ok(Self {
            db: Arc::new(Mutex::new(conn)),
        })
    }

    fn db_path() -> PathBuf {
        let base = dirs_data_dir().unwrap_or_else(|| PathBuf::from("."));
        base.join(".quickdesk").join("memory").join("device_memory.db")
    }

    fn init_tables(conn: &Connection) -> Result<(), String> {
        conn.execute_batch(
            "CREATE TABLE IF NOT EXISTS device_profiles (
                device_id       TEXT PRIMARY KEY,
                os_name         TEXT DEFAULT '',
                os_version      TEXT DEFAULT '',
                hostname        TEXT DEFAULT '',
                cpu_model       TEXT DEFAULT '',
                memory_total_mb INTEGER DEFAULT 0,
                disk_total_gb   INTEGER DEFAULT 0,
                resolution      TEXT DEFAULT '',
                known_apps      TEXT DEFAULT '[]',
                notes           TEXT DEFAULT '',
                first_seen      TEXT NOT NULL,
                last_seen       TEXT NOT NULL,
                connection_count INTEGER DEFAULT 0,
                updated_at      TEXT NOT NULL
            );
            CREATE TABLE IF NOT EXISTS operation_history (
                id              INTEGER PRIMARY KEY AUTOINCREMENT,
                session_id      TEXT NOT NULL,
                device_id       TEXT NOT NULL,
                timestamp       TEXT NOT NULL,
                tool_name       TEXT NOT NULL,
                arguments       TEXT DEFAULT '{}',
                result_summary  TEXT DEFAULT '',
                success         INTEGER NOT NULL DEFAULT 1,
                duration_ms     INTEGER DEFAULT 0,
                error_message   TEXT DEFAULT ''
            );
            CREATE INDEX IF NOT EXISTS idx_history_device ON operation_history(device_id, timestamp);
            CREATE INDEX IF NOT EXISTS idx_history_tool ON operation_history(tool_name);
            CREATE INDEX IF NOT EXISTS idx_history_session ON operation_history(session_id);
            CREATE TABLE IF NOT EXISTS failure_memory (
                id                INTEGER PRIMARY KEY AUTOINCREMENT,
                device_id         TEXT NOT NULL,
                tool_name         TEXT NOT NULL,
                arguments_pattern TEXT DEFAULT '',
                error_category    TEXT DEFAULT 'unknown',
                error_message     TEXT DEFAULT '',
                occurrence_count  INTEGER DEFAULT 1,
                last_occurred     TEXT NOT NULL,
                resolution        TEXT DEFAULT ''
            );
            CREATE INDEX IF NOT EXISTS idx_failure_device ON failure_memory(device_id);
            CREATE INDEX IF NOT EXISTS idx_failure_tool ON failure_memory(device_id, tool_name);",
        )
        .map_err(|e| format!("init tables: {e}"))
    }

    // ---- Device Profile ----

    pub fn upsert_profile(&self, device_id: &str, info: &Value) {
        let now = Utc::now().to_rfc3339();
        let db = self.db.lock().unwrap();

        let exists: bool = db
            .query_row(
                "SELECT COUNT(*) FROM device_profiles WHERE device_id=?1",
                params![device_id],
                |r| r.get::<_, i64>(0),
            )
            .unwrap_or(0)
            > 0;

        if exists {
            let _ = db.execute(
                "UPDATE device_profiles SET
                    os_name=COALESCE(NULLIF(?2,''), os_name),
                    os_version=COALESCE(NULLIF(?3,''), os_version),
                    hostname=COALESCE(NULLIF(?4,''), hostname),
                    cpu_model=COALESCE(NULLIF(?5,''), cpu_model),
                    memory_total_mb=CASE WHEN ?6>0 THEN ?6 ELSE memory_total_mb END,
                    disk_total_gb=CASE WHEN ?7>0 THEN ?7 ELSE disk_total_gb END,
                    resolution=COALESCE(NULLIF(?8,''), resolution),
                    last_seen=?9,
                    connection_count=connection_count+1,
                    updated_at=?9
                WHERE device_id=?1",
                params![
                    device_id,
                    info.get("os_name").and_then(|v| v.as_str()).unwrap_or(""),
                    info.get("os_version").and_then(|v| v.as_str()).unwrap_or(""),
                    info.get("hostname").and_then(|v| v.as_str()).unwrap_or(""),
                    info.get("cpu_model").and_then(|v| v.as_str()).unwrap_or(""),
                    info.get("memory_total_mb").and_then(|v| v.as_i64()).unwrap_or(0),
                    info.get("disk_total_gb").and_then(|v| v.as_i64()).unwrap_or(0),
                    info.get("resolution").and_then(|v| v.as_str()).unwrap_or(""),
                    now,
                ],
            );
        } else {
            let _ = db.execute(
                "INSERT INTO device_profiles
                    (device_id, os_name, os_version, hostname, cpu_model,
                     memory_total_mb, disk_total_gb, resolution,
                     first_seen, last_seen, connection_count, updated_at)
                VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?9,1,?9)",
                params![
                    device_id,
                    info.get("os_name").and_then(|v| v.as_str()).unwrap_or(""),
                    info.get("os_version").and_then(|v| v.as_str()).unwrap_or(""),
                    info.get("hostname").and_then(|v| v.as_str()).unwrap_or(""),
                    info.get("cpu_model").and_then(|v| v.as_str()).unwrap_or(""),
                    info.get("memory_total_mb").and_then(|v| v.as_i64()).unwrap_or(0),
                    info.get("disk_total_gb").and_then(|v| v.as_i64()).unwrap_or(0),
                    info.get("resolution").and_then(|v| v.as_str()).unwrap_or(""),
                    now,
                ],
            );
        }
    }

    pub fn get_profile(&self, device_id: &str) -> Option<DeviceProfile> {
        let db = self.db.lock().unwrap();
        db.query_row(
            "SELECT device_id, os_name, os_version, hostname, cpu_model,
                    memory_total_mb, disk_total_gb, resolution, known_apps,
                    notes, first_seen, last_seen, connection_count
             FROM device_profiles WHERE device_id=?1",
            params![device_id],
            |row| {
                let apps_json: String = row.get(8)?;
                let apps: Vec<String> =
                    serde_json::from_str(&apps_json).unwrap_or_default();
                Ok(DeviceProfile {
                    device_id: row.get(0)?,
                    os_name: row.get(1)?,
                    os_version: row.get(2)?,
                    hostname: row.get(3)?,
                    cpu_model: row.get(4)?,
                    memory_total_mb: row.get(5)?,
                    disk_total_gb: row.get(6)?,
                    resolution: row.get(7)?,
                    known_apps: apps,
                    notes: row.get(9)?,
                    first_seen: row.get(10)?,
                    last_seen: row.get(11)?,
                    connection_count: row.get(12)?,
                })
            },
        )
        .ok()
    }

    pub fn update_profile_field(&self, device_id: &str, field: &str, value: &str) -> bool {
        let allowed = [
            "os_name", "os_version", "hostname", "resolution", "notes", "known_apps",
        ];
        if !allowed.contains(&field) {
            return false;
        }
        let now = Utc::now().to_rfc3339();
        let db = self.db.lock().unwrap();

        db.execute(
            "INSERT OR IGNORE INTO device_profiles (device_id, first_seen, last_seen, updated_at) VALUES (?1, ?2, ?3, ?4)",
            params![device_id, now, now, now],
        ).ok();

        let sql = format!(
            "UPDATE device_profiles SET {field}=?1, updated_at=?2 WHERE device_id=?3"
        );
        db.execute(&sql, params![value, now, device_id])
            .map(|n| n > 0)
            .unwrap_or(false)
    }

    // ---- Operation History ----

    pub fn log_operation(
        &self,
        session_id: &str,
        device_id: &str,
        tool_name: &str,
        arguments: &Value,
        result_summary: &str,
        success: bool,
        duration_ms: i64,
        error_message: &str,
    ) {
        let now = Utc::now().to_rfc3339();
        let args_str = serde_json::to_string(arguments).unwrap_or_default();
        let summary = if result_summary.len() > 1024 {
            &result_summary[..1024]
        } else {
            result_summary
        };

        let db = self.db.lock().unwrap();
        let _ = db.execute(
            "INSERT INTO operation_history
                (session_id, device_id, timestamp, tool_name, arguments,
                 result_summary, success, duration_ms, error_message)
            VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9)",
            params![
                session_id,
                device_id,
                now,
                tool_name,
                args_str,
                summary,
                success as i32,
                duration_ms,
                error_message,
            ],
        );

        // Release the guard before record_failure takes it again. The mutex is
        // not reentrant: holding it across that call blocked the thread on a
        // lock only that thread could release, and every failed tool call
        // reached this path, so the first failure froze the whole server.
        drop(db);
        if !success {
            self.record_failure(device_id, tool_name, &args_str, error_message);
        }
    }

    pub fn search_history(
        &self,
        device_id: &str,
        tool_name: Option<&str>,
        keyword: Option<&str>,
        success_only: bool,
        limit: i64,
    ) -> Vec<OperationRecord> {
        let db = self.db.lock().unwrap();
        let mut sql = String::from(
            "SELECT id, session_id, device_id, timestamp, tool_name,
                    arguments, result_summary, success, duration_ms, error_message
             FROM operation_history WHERE device_id=?1",
        );
        let mut bind_idx = 2;
        let mut params_vec: Vec<Box<dyn rusqlite::types::ToSql>> = vec![Box::new(device_id.to_string())];

        if let Some(tn) = tool_name {
            sql.push_str(&format!(" AND tool_name=?{bind_idx}"));
            params_vec.push(Box::new(tn.to_string()));
            bind_idx += 1;
        }
        if success_only {
            sql.push_str(" AND success=1");
        }
        if let Some(kw) = keyword {
            sql.push_str(&format!(" AND (result_summary LIKE ?{bind_idx} OR arguments LIKE ?{bind_idx})"));
            params_vec.push(Box::new(format!("%{kw}%")));
            bind_idx += 1;
        }
        let _ = bind_idx;
        sql.push_str(&format!(" ORDER BY timestamp DESC LIMIT {limit}"));

        let params_refs: Vec<&dyn rusqlite::types::ToSql> = params_vec.iter().map(|b| b.as_ref()).collect();
        let mut stmt = match db.prepare(&sql) {
            Ok(s) => s,
            Err(_) => return vec![],
        };
        let rows = stmt
            .query_map(params_refs.as_slice(), |row| {
                Ok(OperationRecord {
                    id: row.get(0)?,
                    session_id: row.get(1)?,
                    device_id: row.get(2)?,
                    timestamp: row.get(3)?,
                    tool_name: row.get(4)?,
                    arguments: serde_json::from_str(&row.get::<_, String>(5)?)
                        .unwrap_or(Value::Null),
                    result_summary: row.get(6)?,
                    success: row.get::<_, i32>(7)? != 0,
                    duration_ms: row.get(8)?,
                    error_message: row.get(9)?,
                })
            })
            .ok();

        rows.map(|r| r.filter_map(|x| x.ok()).collect())
            .unwrap_or_default()
    }

    // ---- Failure Memory ----

    fn record_failure(&self, device_id: &str, tool_name: &str, args: &str, error: &str) {
        let category = classify_error(error);
        let pattern = normalize_args(args);
        let now = Utc::now().to_rfc3339();
        let db = self.db.lock().unwrap();

        let existing: Option<i64> = db
            .query_row(
                "SELECT id FROM failure_memory
                 WHERE device_id=?1 AND tool_name=?2 AND arguments_pattern=?3",
                params![device_id, tool_name, pattern],
                |r| r.get(0),
            )
            .ok();

        if let Some(id) = existing {
            let _ = db.execute(
                "UPDATE failure_memory
                 SET occurrence_count=occurrence_count+1, last_occurred=?1, error_message=?2
                 WHERE id=?3",
                params![now, error, id],
            );
        } else {
            let _ = db.execute(
                "INSERT INTO failure_memory
                    (device_id, tool_name, arguments_pattern, error_category,
                     error_message, last_occurred)
                 VALUES (?1,?2,?3,?4,?5,?6)",
                params![device_id, tool_name, pattern, category, error, now],
            );
        }
    }

    pub fn get_failures(&self, device_id: &str, tool_name: Option<&str>) -> Vec<FailureRecord> {
        let db = self.db.lock().unwrap();
        let sql = if let Some(tn) = tool_name {
            format!(
                "SELECT id, device_id, tool_name, arguments_pattern, error_category,
                        error_message, occurrence_count, last_occurred, resolution
                 FROM failure_memory WHERE device_id='{}' AND tool_name='{}'
                 ORDER BY occurrence_count DESC LIMIT 50",
                device_id.replace('\'', "''"),
                tn.replace('\'', "''")
            )
        } else {
            format!(
                "SELECT id, device_id, tool_name, arguments_pattern, error_category,
                        error_message, occurrence_count, last_occurred, resolution
                 FROM failure_memory WHERE device_id='{}'
                 ORDER BY occurrence_count DESC LIMIT 50",
                device_id.replace('\'', "''")
            )
        };

        let mut stmt = match db.prepare(&sql) {
            Ok(s) => s,
            Err(_) => return vec![],
        };
        stmt.query_map([], |row| {
            Ok(FailureRecord {
                id: row.get(0)?,
                device_id: row.get(1)?,
                tool_name: row.get(2)?,
                arguments_pattern: row.get(3)?,
                error_category: row.get(4)?,
                error_message: row.get(5)?,
                occurrence_count: row.get(6)?,
                last_occurred: row.get(7)?,
                resolution: row.get(8)?,
            })
        })
        .ok()
        .map(|r| r.filter_map(|x| x.ok()).collect())
        .unwrap_or_default()
    }

    // ---- Summary ----

    pub fn device_summary(&self, device_id: &str) -> DeviceSummary {
        let db = self.db.lock().unwrap();

        let total: i64 = db
            .query_row(
                "SELECT COUNT(*) FROM operation_history WHERE device_id=?1",
                params![device_id],
                |r| r.get(0),
            )
            .unwrap_or(0);

        let successes: i64 = db
            .query_row(
                "SELECT COUNT(*) FROM operation_history WHERE device_id=?1 AND success=1",
                params![device_id],
                |r| r.get(0),
            )
            .unwrap_or(0);

        let rate = if total > 0 {
            successes as f64 / total as f64
        } else {
            0.0
        };

        let sessions: i64 = db
            .query_row(
                "SELECT COUNT(DISTINCT session_id) FROM operation_history WHERE device_id=?1",
                params![device_id],
                |r| r.get(0),
            )
            .unwrap_or(0);

        let mut stmt = db
            .prepare(
                "SELECT tool_name, COUNT(*) as cnt FROM operation_history
                 WHERE device_id=?1 GROUP BY tool_name ORDER BY cnt DESC LIMIT 10",
            )
            .unwrap();
        let tools: Vec<ToolUsage> = stmt
            .query_map(params![device_id], |row| {
                Ok(ToolUsage {
                    tool: row.get(0)?,
                    count: row.get(1)?,
                })
            })
            .ok()
            .map(|r| r.filter_map(|x| x.ok()).collect())
            .unwrap_or_default();

        drop(stmt);
        drop(db);

        let failures = self.get_failures(device_id, None);

        DeviceSummary {
            device_id: device_id.to_string(),
            total_operations: total,
            success_rate: rate,
            most_used_tools: tools,
            common_failures: failures,
            sessions,
        }
    }

    // ---- Cleanup ----

    pub fn cleanup_old_history(&self, max_days: i64) {
        let cutoff = Utc::now() - chrono::Duration::days(max_days);
        let db = self.db.lock().unwrap();
        let _ = db.execute(
            "DELETE FROM operation_history WHERE timestamp < ?1",
            params![cutoff.to_rfc3339()],
        );
    }
}

fn classify_error(msg: &str) -> String {
    let lower = msg.to_lowercase();
    if lower.contains("timeout") || lower.contains("timed out") {
        "timeout".to_string()
    } else if lower.contains("not found") || lower.contains("no such") {
        "not_found".to_string()
    } else if lower.contains("permission") || lower.contains("denied") || lower.contains("access") {
        "permission".to_string()
    } else {
        "unknown".to_string()
    }
}

fn normalize_args(args: &str) -> String {
    if let Ok(v) = serde_json::from_str::<Value>(args) {
        if let Value::Object(map) = v {
            let keys: Vec<&String> = map.keys().collect();
            return format!("{{{}}}", keys.iter().map(|k| k.as_str()).collect::<Vec<_>>().join(","));
        }
    }
    String::new()
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

#[cfg(test)]
mod tests {
    use super::*;

    fn store_in_temp() -> (MemoryStore, PathBuf) {
        let dir = std::env::temp_dir().join(format!(
            "quickdesk-memory-test-{}-{}",
            std::process::id(),
            Utc::now().timestamp_nanos_opt().unwrap_or_default()
        ));
        std::fs::create_dir_all(&dir).unwrap();
        let store = MemoryStore::open_at(dir.join("memory.db")).unwrap();
        (store, dir)
    }

    // A deadlock reports nothing: the test never returns, so CI kills the job
    // with no diagnosis. Running the body on a worker with a receive timeout
    // turns that silence into a failure that names the problem.
    fn without_deadlocking<F: FnOnce() + Send + 'static>(body: F) {
        let (done, finished) = std::sync::mpsc::channel();
        let worker = std::thread::spawn(move || {
            body();
            let _ = done.send(());
        });
        assert!(
            finished.recv_timeout(std::time::Duration::from_secs(20)).is_ok(),
            "timed out: the store is deadlocked, not slow"
        );
        worker.join().unwrap();
    }

    #[test]
    fn logging_a_failure_records_it_without_deadlocking() {
        without_deadlocking(|| {
            let (store, dir) = store_in_temp();
            store.log_operation(
                "session-1",
                "device-1",
                "mouse_click",
                &serde_json::json!({"x": 10, "y": 20}),
                "failed",
                false,
                12,
                "connection timed out",
            );

            let failures = store.get_failures("device-1", None);
            assert_eq!(failures.len(), 1, "the failure should be recorded");
            assert_eq!(failures[0].occurrence_count, 1);
            assert_eq!(failures[0].error_category, "timeout");

            let _ = std::fs::remove_dir_all(dir);
        });
    }

    #[test]
    fn logging_a_repeated_failure_counts_it_as_one_pattern() {
        without_deadlocking(|| {
            let (store, dir) = store_in_temp();
            for duration in [12, 14] {
                store.log_operation(
                    "session-1",
                    "device-1",
                    "mouse_click",
                    &serde_json::json!({"x": 10, "y": 20}),
                    "failed",
                    false,
                    duration,
                    "connection timed out",
                );
            }

            let failures = store.get_failures("device-1", None);
            assert_eq!(failures.len(), 1, "same shape is one pattern, not two");
            assert_eq!(failures[0].occurrence_count, 2);

            let _ = std::fs::remove_dir_all(dir);
        });
    }

    #[test]
    fn summarizing_a_device_with_failures_does_not_deadlock() {
        without_deadlocking(|| {
            let (store, dir) = store_in_temp();
            store.log_operation(
                "session-1",
                "device-1",
                "mouse_click",
                &serde_json::json!({"x": 1, "y": 2}),
                "failed",
                false,
                5,
                "connection timed out",
            );

            let summary = store.device_summary("device-1");
            assert_eq!(summary.total_operations, 1);
            assert_eq!(summary.common_failures.len(), 1);

            let _ = std::fs::remove_dir_all(dir);
        });
    }
}
