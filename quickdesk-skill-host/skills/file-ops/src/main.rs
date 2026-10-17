// Copyright 2026 QuickDesk Authors
// file-ops — MCP server that exposes file operation tools.

use anyhow::Result;
use base64::Engine;
use mcp_server_common::{McpServer, ToolDef, ToolHandler};
use serde_json::{json, Value};
use std::fs;
use std::fs::{File, OpenOptions};
use std::io::{Read, Seek, SeekFrom, Write};
use std::path::Path;

const DEFAULT_CHUNK_BYTES: usize = 64 * 1024;
const MAX_CHUNK_BYTES: usize = 256 * 1024;

struct FileOpsHandler;

impl ToolHandler for FileOpsHandler {
    fn server_name(&self) -> &str {
        "file-ops"
    }

    fn server_version(&self) -> &str {
        "0.1.0"
    }

    fn tool_defs(&self) -> Vec<ToolDef> {
        vec![
            ToolDef {
                name: "read_file",
                description: "Read the contents of a file. Returns the text content.",
                input_schema: json!({
                    "type": "object",
                    "properties": {
                        "path": {
                            "type": "string",
                            "description": "Absolute path to the file"
                        }
                    },
                    "required": ["path"]
                }),
            },
            ToolDef {
                name: "write_file",
                description: "Write content to a file. Creates the file if it doesn't exist, overwrites if it does.",
                input_schema: json!({
                    "type": "object",
                    "properties": {
                        "path": {
                            "type": "string",
                            "description": "Absolute path to the file"
                        },
                        "content": {
                            "type": "string",
                            "description": "Content to write"
                        }
                    },
                    "required": ["path", "content"]
                }),
            },
            ToolDef {
                name: "list_directory",
                description: "List files and directories at a given path",
                input_schema: json!({
                    "type": "object",
                    "properties": {
                        "path": {
                            "type": "string",
                            "description": "Absolute path to the directory"
                        }
                    },
                    "required": ["path"]
                }),
            },
            ToolDef {
                name: "create_directory",
                description: "Create a directory (and parent directories if needed)",
                input_schema: json!({
                    "type": "object",
                    "properties": {
                        "path": {
                            "type": "string",
                            "description": "Absolute path of the directory to create"
                        }
                    },
                    "required": ["path"]
                }),
            },
            ToolDef {
                name: "move_file",
                description: "Move or rename a file or directory",
                input_schema: json!({
                    "type": "object",
                    "properties": {
                        "source": {
                            "type": "string",
                            "description": "Source path"
                        },
                        "destination": {
                            "type": "string",
                            "description": "Destination path"
                        }
                    },
                    "required": ["source", "destination"]
                }),
            },
            ToolDef {
                name: "get_file_info",
                description: "Get file metadata: size, modification time, permissions, type",
                input_schema: json!({
                    "type": "object",
                    "properties": {
                        "path": {
                            "type": "string",
                            "description": "Absolute path to the file or directory"
                        }
                    },
                    "required": ["path"]
                }),
            },
            ToolDef {
                name: "get_default_directory",
                description: "Get the current user's home directory for the remote file browser",
                input_schema: json!({
                    "type": "object",
                    "properties": {}
                }),
            },
            ToolDef {
                name: "read_file_chunk",
                description: "Read a bounded base64 file chunk for browser download",
                input_schema: json!({
                    "type": "object",
                    "properties": {
                        "path": {"type": "string"},
                        "offset": {"type": "integer", "minimum": 0},
                        "length": {
                            "type": "integer",
                            "minimum": 1
                        }
                    },
                    "required": ["path"]
                }),
            },
            ToolDef {
                name: "write_file_chunk",
                description: "Write a bounded base64 file chunk for browser upload",
                input_schema: json!({
                    "type": "object",
                    "properties": {
                        "path": {"type": "string"},
                        "offset": {"type": "integer", "minimum": 0},
                        "truncate": {"type": "boolean"},
                        "data_base64": {"type": "string"}
                    },
                    "required": ["path", "offset", "data_base64"]
                }),
            },
            ToolDef {
                name: "trash_path",
                description: "Move a file or directory into .quickdesk-trash without overwriting older deleted versions",
                input_schema: json!({
                    "type": "object",
                    "properties": {
                        "path": {"type": "string"}
                    },
                    "required": ["path"]
                }),
            },
        ]
    }

    fn call_tool(
        &self,
        name: &str,
        args: Value,
    ) -> std::pin::Pin<Box<dyn std::future::Future<Output = Result<Value>> + Send + '_>> {
        let name = name.to_string();
        let args = args.clone();
        Box::pin(async move {
            match name.as_str() {
                "read_file" => read_file(args),
                "write_file" => write_file(args),
                "list_directory" => list_directory(args),
                "create_directory" => create_directory(args),
                "move_file" => move_file(args),
                "get_file_info" => get_file_info(args),
                "get_default_directory" => default_directory(args),
                "read_file_chunk" => read_file_chunk(args),
                "write_file_chunk" => write_file_chunk(args),
                "trash_path" => trash_path(args),
                _ => anyhow::bail!("unknown tool: {}", name),
            }
        })
    }
}

fn require_str<'a>(args: &'a Value, field: &str) -> Result<&'a str> {
    args.get(field)
        .and_then(|v| v.as_str())
        .ok_or_else(|| anyhow::anyhow!("missing required field: {}", field))
}

fn read_file(args: Value) -> Result<Value> {
    let path = require_str(&args, "path")?;
    let content = fs::read_to_string(path)
        .map_err(|e| anyhow::anyhow!("failed to read '{}': {}", path, e))?;
    Ok(json!({
        "path": path,
        "content": content,
        "size": content.len(),
    }))
}

fn write_file(args: Value) -> Result<Value> {
    let path = require_str(&args, "path")?;
    let content = require_str(&args, "content")?;

    if let Some(parent) = Path::new(path).parent() {
        fs::create_dir_all(parent)?;
    }
    fs::write(path, content)?;

    Ok(json!({
        "path": path,
        "bytes_written": content.len(),
    }))
}

fn list_directory(args: Value) -> Result<Value> {
    let path = require_str(&args, "path")?;
    let entries = fs::read_dir(path)
        .map_err(|e| anyhow::anyhow!("failed to read directory '{}': {}", path, e))?;

    let mut items: Vec<Value> = Vec::new();
    for entry in entries.flatten() {
        let meta = entry.metadata();
        let is_dir = meta.as_ref().map(|m| m.is_dir()).unwrap_or(false);
        let size = meta.as_ref().map(|m| m.len()).unwrap_or(0);
        items.push(json!({
            "name": entry.file_name().to_string_lossy(),
            "type": if is_dir { "directory" } else { "file" },
            "size": size,
        }));
    }

    items.sort_by(|a, b| {
        let ta = a["type"].as_str().unwrap_or("");
        let tb = b["type"].as_str().unwrap_or("");
        let na = a["name"].as_str().unwrap_or("");
        let nb = b["name"].as_str().unwrap_or("");
        tb.cmp(ta).then(na.to_lowercase().cmp(&nb.to_lowercase()))
    });

    Ok(json!({
        "path": path,
        "count": items.len(),
        "entries": items,
    }))
}

fn create_directory(args: Value) -> Result<Value> {
    let path = require_str(&args, "path")?;
    fs::create_dir_all(path)?;
    Ok(json!({
        "path": path,
        "created": true,
    }))
}

fn move_file(args: Value) -> Result<Value> {
    let source = require_str(&args, "source")?;
    let destination = require_str(&args, "destination")?;
    fs::rename(source, destination)
        .map_err(|e| anyhow::anyhow!("failed to move '{}' to '{}': {}", source, destination, e))?;
    Ok(json!({
        "source": source,
        "destination": destination,
        "moved": true,
    }))
}

fn get_file_info(args: Value) -> Result<Value> {
    let path = require_str(&args, "path")?;
    let meta = fs::metadata(path)
        .map_err(|e| anyhow::anyhow!("failed to get info for '{}': {}", path, e))?;

    let file_type = if meta.is_dir() {
        "directory"
    } else if meta.is_symlink() {
        "symlink"
    } else {
        "file"
    };

    let modified = meta
        .modified()
        .ok()
        .and_then(|t| t.duration_since(std::time::UNIX_EPOCH).ok())
        .map(|d| d.as_secs());

    let mut info = json!({
        "path": path,
        "type": file_type,
        "size": meta.len(),
        "readonly": meta.permissions().readonly(),
    });
    if let Some(ts) = modified {
        info["modified_unix"] = json!(ts);
    }

    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        info["permissions_octal"] = json!(format!("{:o}", meta.permissions().mode()));
    }

    Ok(info)
}

fn optional_u64(args: &Value, field: &str) -> Result<Option<u64>> {
    match args.get(field) {
        None | Some(Value::Null) => Ok(None),
        Some(value) => value
            .as_u64()
            .map(Some)
            .ok_or_else(|| anyhow::anyhow!("{} must be a non-negative integer", field)),
    }
}

fn read_file_chunk(args: Value) -> Result<Value> {
    let path = require_str(&args, "path")?;
    let offset = optional_u64(&args, "offset")?.unwrap_or(0);
    let requested = optional_u64(&args, "length")?.unwrap_or(DEFAULT_CHUNK_BYTES as u64);
    let length =
        usize::try_from(requested.clamp(1, MAX_CHUNK_BYTES as u64)).unwrap_or(DEFAULT_CHUNK_BYTES);

    let mut file =
        File::open(path).map_err(|e| anyhow::anyhow!("failed to open '{}': {}", path, e))?;
    let size = file.metadata()?.len();
    if offset > size {
        anyhow::bail!("offset {} is beyond file size {}", offset, size);
    }
    file.seek(SeekFrom::Start(offset))?;

    let remaining = (size - offset).min(length as u64) as usize;
    let mut data = Vec::with_capacity(remaining);
    file.take(remaining as u64).read_to_end(&mut data)?;
    let next_offset = offset + data.len() as u64;
    Ok(json!({
        "path": path,
        "offset": offset,
        "next_offset": next_offset,
        "size": size,
        "eof": next_offset >= size,
        "data_base64": base64::engine::general_purpose::STANDARD.encode(data),
    }))
}

fn write_file_chunk(args: Value) -> Result<Value> {
    let path = require_str(&args, "path")?;
    let offset = optional_u64(&args, "offset")?.unwrap_or(0);
    let truncate = args
        .get("truncate")
        .and_then(Value::as_bool)
        .unwrap_or(false);
    if truncate && offset != 0 {
        anyhow::bail!("truncate is only valid at offset 0");
    }
    let encoded = require_str(&args, "data_base64")?;
    let data = base64::engine::general_purpose::STANDARD
        .decode(encoded)
        .map_err(|e| anyhow::anyhow!("invalid base64 data: {}", e))?;
    if data.len() > MAX_CHUNK_BYTES {
        anyhow::bail!(
            "chunk is {} bytes; maximum is {}",
            data.len(),
            MAX_CHUNK_BYTES
        );
    }

    if let Some(parent) = Path::new(path).parent() {
        fs::create_dir_all(parent)?;
    }
    let mut file = OpenOptions::new()
        .create(true)
        .truncate(false)
        .read(true)
        .write(true)
        .open(path)
        .map_err(|e| anyhow::anyhow!("failed to open '{}' for writing: {}", path, e))?;
    if truncate {
        file.set_len(0)?;
    }
    file.seek(SeekFrom::Start(offset))?;
    file.write_all(&data)?;
    file.flush()?;
    Ok(json!({
        "path": path,
        "offset": offset,
        "bytes_written": data.len(),
        "next_offset": offset + data.len() as u64,
    }))
}

fn default_directory(_args: Value) -> Result<Value> {
    let var = if cfg!(target_os = "windows") {
        "USERPROFILE"
    } else {
        "HOME"
    };
    let path = std::env::var_os(var)
        .map(std::path::PathBuf::from)
        .or_else(|| std::env::current_dir().ok())
        .ok_or_else(|| anyhow::anyhow!("cannot determine a default directory"))?;
    let path = fs::canonicalize(&path).unwrap_or(path);
    if !path.is_dir() {
        anyhow::bail!("default directory is unavailable: {}", path.display());
    }
    Ok(json!({
        "path": path.to_string_lossy(),
        "separator": if cfg!(target_os = "windows") { "\\" } else { "/" },
    }))
}

fn trash_path(args: Value) -> Result<Value> {
    let source_text = require_str(&args, "path")?;
    let source = Path::new(source_text);
    if !source.exists() {
        anyhow::bail!("path does not exist: {}", source.display());
    }
    let parent = source
        .parent()
        .ok_or_else(|| anyhow::anyhow!("cannot trash a path without a parent"))?;
    let trash_dir = parent.join(".quickdesk-trash");
    if source == trash_dir {
        anyhow::bail!("the trash directory cannot trash itself");
    }
    fs::create_dir_all(&trash_dir)?;

    let name = source
        .file_name()
        .and_then(|value| value.to_str())
        .filter(|value| !value.is_empty())
        .ok_or_else(|| anyhow::anyhow!("path has no usable file name"))?;
    let stamp = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap_or_default()
        .as_millis();
    let mut destination = trash_dir.join(format!("{name}.{stamp}"));
    let mut suffix = 1u32;
    while destination.exists() {
        destination = trash_dir.join(format!("{name}.{stamp}.{suffix}"));
        suffix += 1;
    }

    fs::rename(source, &destination)
        .map_err(|e| anyhow::anyhow!("failed to trash '{}': {}", source.display(), e))?;
    Ok(json!({
        "path": source_text,
        "trash_path": destination.to_string_lossy(),
        "trashed": true,
    }))
}

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt()
        .with_writer(std::io::stderr)
        .with_env_filter("file_ops=info")
        .init();

    let server = McpServer::new(FileOpsHandler);
    server.run().await
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::atomic::{AtomicU64, Ordering};

    static NEXT_TEMP: AtomicU64 = AtomicU64::new(1);

    fn temp_dir() -> std::path::PathBuf {
        let id = NEXT_TEMP.fetch_add(1, Ordering::SeqCst);
        let dir =
            std::env::temp_dir().join(format!("quickdesk-file-ops-{}-{id}", std::process::id()));
        fs::create_dir_all(&dir).unwrap();
        dir
    }

    #[test]
    fn file_chunks_round_trip_and_truncate_on_first_write() {
        let dir = temp_dir();
        let path = dir.join("report.txt");
        let path_text = path.to_string_lossy().to_string();

        let first = write_file_chunk(json!({
            "path": path_text,
            "offset": 0,
            "truncate": true,
            "data_base64": "c3RhbGU="
        }))
        .unwrap();
        assert_eq!(first["bytes_written"], 5);

        let second = write_file_chunk(json!({
            "path": path_text,
            "offset": 5,
            "data_base64": "LW5ldw=="
        }))
        .unwrap();
        assert_eq!(second["next_offset"], 9);

        let chunk = read_file_chunk(json!({
            "path": path_text,
            "offset": 0,
            "length": 64
        }))
        .unwrap();
        assert_eq!(chunk["data_base64"], "c3RhbGUtbmV3");
        assert_eq!(chunk["size"], 9);
        assert_eq!(chunk["eof"], true);

        let replaced = write_file_chunk(json!({
            "path": path_text,
            "offset": 0,
            "truncate": true,
            "data_base64": "eA=="
        }))
        .unwrap();
        assert_eq!(replaced["next_offset"], 1);
        assert_eq!(fs::read(&path).unwrap(), b"x");

        let _ = fs::remove_dir_all(dir);
    }

    #[test]
    fn file_chunks_reject_invalid_base64_and_oversized_reads() {
        let dir = temp_dir();
        let path = dir.join("large.bin");
        fs::write(&path, vec![7u8; MAX_CHUNK_BYTES + 100]).unwrap();
        let path_text = path.to_string_lossy().to_string();

        let chunk = read_file_chunk(json!({
            "path": path_text,
            "offset": 0,
            "length": 9999999
        }))
        .unwrap();
        let decoded = base64::engine::general_purpose::STANDARD
            .decode(chunk["data_base64"].as_str().unwrap())
            .unwrap();
        assert_eq!(decoded.len(), MAX_CHUNK_BYTES);
        assert_eq!(chunk["next_offset"], MAX_CHUNK_BYTES);
        assert_eq!(chunk["eof"], false);

        let error = write_file_chunk(json!({
            "path": path_text,
            "offset": 0,
            "truncate": true,
            "data_base64": "not base64!"
        }))
        .unwrap_err();
        assert!(error.to_string().contains("base64"));

        let _ = fs::remove_dir_all(dir);
    }

    #[test]
    fn trash_path_preserves_every_deleted_version() {
        let dir = temp_dir();
        let original = dir.join("notes.txt");
        fs::write(&original, b"first").unwrap();

        let first = trash_path(json!({"path": original.to_string_lossy()})).unwrap();
        let first_trash = Path::new(first["trash_path"].as_str().unwrap());
        assert!(first_trash.starts_with(dir.join(".quickdesk-trash")));
        assert_eq!(fs::read(first_trash).unwrap(), b"first");
        assert!(!original.exists());

        fs::write(&original, b"second").unwrap();
        let second = trash_path(json!({"path": original.to_string_lossy()})).unwrap();
        let second_trash = Path::new(second["trash_path"].as_str().unwrap());
        assert_ne!(first_trash, second_trash);
        assert_eq!(fs::read(second_trash).unwrap(), b"second");

        let _ = fs::remove_dir_all(dir);
    }

    #[test]
    fn default_directory_returns_an_existing_absolute_path() {
        let result = default_directory(json!({})).unwrap();
        let path = Path::new(result["path"].as_str().unwrap());
        assert!(path.is_absolute());
        assert!(path.exists());
        assert!(path.is_dir());
    }
}
