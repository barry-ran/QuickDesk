---
name: sys-info
description: "Retrieve system information and list running processes on a remote host via QuickDesk. Use when the user asks to check server health, monitor CPU or memory usage, view disk space, list running processes, or query OS and uptime details on a remote machine."
metadata:
  openclaw:
    os: ["win32", "darwin", "linux"]
    install:
      - id: binary
        kind: binary
        package: "sys-info"
---

# sys-info

Query system health and running processes on a remote host connected through QuickDesk.

## Tools

### get_system_info

Retrieve OS version, CPU model and usage, memory and swap totals/used, disk usage, hostname, and uptime. Takes no parameters.

Returns:

```json
{
  "os": "Ubuntu 22.04",
  "kernel": "5.15.0",
  "hostname": "prod-server-01",
  "cpu": { "model": "AMD EPYC 7763", "cores": 8, "usage_percent": "12.3" },
  "memory": { "total_mb": 16384, "used_mb": 8200, "usage_percent": "50.0" },
  "swap": { "total_mb": 4096, "used_mb": 128 },
  "disks": [{ "mount": "/", "total_gb": 500.0, "available_gb": 320.5, "fs": "ext4" }],
  "uptime": "14h 32m"
}
```

### list_processes

List running processes with resource usage, sorted and limited.

- **sort_by** (string, optional) — sort by `"cpu"`, `"memory"`, or `"name"` (default: `"cpu"`)
- **limit** (integer, optional) — max number of processes to return (default: 50)

Returns:

```json
{
  "total_processes": 312,
  "showing": 5,
  "sort_by": "cpu",
  "processes": [
    { "pid": 1234, "name": "node", "cpu_percent": "45.2", "memory_mb": 512 }
  ]
}
```

## Workflow

1. Call `get_system_info` to get an overview of the remote machine's health.
2. If CPU or memory usage is high, call `list_processes` sorted by `"cpu"` or `"memory"` to identify the top consumers.
3. Use `limit` to control result size when only the top offenders are needed.
