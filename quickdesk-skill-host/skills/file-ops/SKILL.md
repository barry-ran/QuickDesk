---
name: file-ops
description: "Read, write, list, move, and inspect files and directories on a remote host via QuickDesk. Use when the user needs to browse folders, view file contents, create or edit files, rename or relocate files, or check file metadata (size, permissions, modification time) on a remote machine."
metadata:
  openclaw:
    os: ["win32", "darwin", "linux"]
    install:
      - id: binary
        kind: binary
        package: "file-ops"
---

# file-ops

Manage files and directories on a remote host connected through QuickDesk.

## Tools

### read_file

Read the text contents of a file.

- **path** (string, required) — absolute path to the file

Returns `{ "path", "content", "size" }`.

### write_file

Write content to a file. Creates the file (and parent directories) if it does not exist; overwrites if it does.

- **path** (string, required) — absolute path to the file
- **content** (string, required) — content to write

Returns `{ "path", "bytes_written" }`.

### list_directory

List files and directories at a given path, sorted directories-first then alphabetically.

- **path** (string, required) — absolute path to the directory

Returns `{ "path", "count", "entries": [{ "name", "type", "size" }] }`.

### create_directory

Create a directory and any missing parent directories.

- **path** (string, required) — absolute path of the directory to create

Returns `{ "path", "created": true }`.

### move_file

Move or rename a file or directory.

- **source** (string, required) — source path
- **destination** (string, required) — destination path

Returns `{ "source", "destination", "moved": true }`.

### get_file_info

Get file metadata including size, type, modification time, and permissions.

- **path** (string, required) — absolute path to the file or directory

Returns `{ "path", "type", "size", "readonly", "modified_unix", "permissions_octal" }` (permissions_octal on Unix only).

## Workflow

1. Use `list_directory` to browse a remote folder.
2. Use `read_file` to inspect a specific file.
3. Use `write_file` to create or update content — parent directories are created automatically.
4. Use `get_file_info` before destructive operations like `move_file` to confirm the target exists and check permissions.
