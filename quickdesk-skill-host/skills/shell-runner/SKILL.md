---
name: shell-runner
description: "Execute shell commands on a remote host via QuickDesk and return stdout, stderr, and exit code. Use when the user asks to run a command, script, or terminal operation on a remote machine — supports configurable timeout and working directory."
metadata:
  openclaw:
    os: ["win32", "darwin", "linux"]
    install:
      - id: binary
        kind: binary
        package: "shell-runner"
---

# shell-runner

Run shell commands on a remote host connected through QuickDesk. Commands execute via `cmd /C` on Windows or `sh -c` on Unix.

## Tools

### run_command

Execute a shell command on the remote host.

- **command** (string, required) — the shell command to execute
- **working_dir** (string, optional) — working directory for the command
- **timeout_secs** (integer, optional) — timeout in seconds (default: 60)

Returns:

```json
{
  "exit_code": 0,
  "stdout": "...",
  "stderr": "...",
  "stdout_truncated": false,
  "stderr_truncated": false
}
```

Output is truncated at 512 KB per stream if the command produces large output.

## Workflow

1. Call `run_command` with the desired command string.
2. Check `exit_code` — 0 means success, non-zero indicates failure.
3. If `exit_code` is non-zero, inspect `stderr` for error details.
4. For long-running commands, increase `timeout_secs` beyond the 60-second default.
5. Use `working_dir` to execute commands in a specific directory without changing global state.
