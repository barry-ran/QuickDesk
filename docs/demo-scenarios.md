# QuickDesk MCP — Demo Scenarios

This document demonstrates typical AI agent workflows using QuickDesk MCP tools. Each scenario shows the exact sequence of tool calls and expected behavior.

## Scenario 1: Remote Server Health Check

**Goal**: AI agent connects to a remote server and performs a comprehensive health check.

**Prompt**: Use the built-in `server_health_check` prompt.

### Conversation Example

> **User**: Check the health of my server. Device ID: 123456789, access code: 888888.

**AI Agent Workflow**:

```
Step 1: connect_device
  → device_id: "123456789"
  → access_code: "888888"
  → show_window: true
  ← connection_id: "abc123"

Step 2: screenshot
  → connection_id: "abc123", max_width: 1280
  ← [Image: Windows Server 2022 desktop]

Step 3: get_screen_size
  → connection_id: "abc123"
  ← { width: 1920, height: 1080 }

Step 4: keyboard_hotkey
  → keys: ["win", "r"]
  (Opens Run dialog)

Step 5: screenshot → verify Run dialog is open

Step 6: keyboard_type
  → text: "powershell"

Step 7: keyboard_hotkey → keys: ["enter"]

Step 8: screenshot → verify PowerShell is open

Step 9: keyboard_type
  → text: "systeminfo | Select-String 'OS','Memory'"

Step 10: keyboard_hotkey → keys: ["enter"]

Step 11: screenshot → capture output
  (AI reads: Windows Server 2022, 32GB RAM, 28GB in use)

Step 12: keyboard_type
  → text: "Get-WmiObject Win32_LogicalDisk | Format-Table DeviceID,Size,FreeSpace -Auto"

Step 13: keyboard_hotkey → keys: ["enter"]

Step 14: screenshot → capture disk info
  (AI reads: C: 95% full, D: 40% full)

Step 15: keyboard_type
  → text: "Get-Process | Sort-Object CPU -Descending | Select-Object -First 10 Name,CPU,WS"

Step 16: keyboard_hotkey → keys: ["enter"]

Step 17: screenshot → capture process list

Step 18: disconnect_device → connection_id: "abc123"
```

**AI Report**:
> Server health check complete:
> - **OS**: Windows Server 2022 Standard
> - **CPU**: 45% average load — Normal
> - **Memory**: 28/32 GB used (87%) — **WARNING**: High memory usage
> - **Disk C:**: 95% full — **CRITICAL**: Needs immediate cleanup
> - **Disk D:**: 40% full — Normal
> - **Top process**: sqlservr.exe using 12GB RAM
> - **Verdict**: WARNING — Disk C: is nearly full, recommend clearing temp files and old logs.

---

## Scenario 2: Batch Software Update

**Goal**: AI agent updates software on 3 remote devices.

**Prompt**: Use the built-in `batch_operation` prompt.

### Conversation Example

> **User**: Install the latest Chrome on these 3 machines:
> - Server A: ID 111222333, code 111111
> - Server B: ID 444555666, code 222222
> - Server C: ID 777888999, code 333333

**AI Agent Workflow**:

```
For each device:

  1. connect_device(show_window=false)
  2. screenshot → identify OS
  3. Open terminal
  4. Download Chrome installer:
     Windows: keyboard_type → "winget install Google.Chrome --accept-package-agreements"
     Linux:   keyboard_type → "wget https://dl.google.com/... && sudo dpkg -i ..."
  5. keyboard_hotkey → ["enter"]
  6. Wait and screenshot periodically until install completes
  7. screenshot → verify "Successfully installed"
  8. Record result
  9. disconnect_device
```

**AI Report**:

| Device | ID | OS | Result |
|--------|----|----|--------|
| Server A | 111222333 | Windows 11 | ✅ Chrome 125 installed |
| Server B | 444555666 | Windows 10 | ✅ Chrome 125 installed |
| Server C | 777888999 | Ubuntu 22.04 | ✅ Chrome 125 installed |

> Batch operation complete: 3/3 devices updated successfully.

---

## Scenario 3: Remote Troubleshooting

**Goal**: AI agent diagnoses why a user's computer is running slowly.

**Prompt**: Use the built-in `diagnose_system_issue` prompt.

### Conversation Example

> **User**: My work computer is super slow today. Device ID: 999888777, code: 123456.

**AI Agent Workflow**:

```
Step 1: connect_device → connection_id: "xyz789"
Step 2: screenshot
  (AI sees a cluttered desktop with many windows open)

Step 3: keyboard_hotkey → ["ctrl", "shift", "esc"]
  (Opens Task Manager)

Step 4: screenshot
  (AI reads: Memory 15.2/16 GB, CPU 92%, Disk 100%)

Step 5: Click "Memory" column header to sort by memory usage
Step 6: screenshot
  (AI sees: chrome.exe — 8.3 GB, Teams — 2.1 GB, outlook.exe — 1.8 GB)

Step 7: Click "Disk" column header
Step 8: screenshot
  (AI sees: SearchIndexer.exe — 100% disk, MsMpEng.exe — 50%)
```

**AI Response**:
> I found the issue. Your computer is slow because of three problems:
>
> 1. **Memory**: Chrome is using 8.3 GB (52% of total). You have 47 tabs open.
> 2. **Disk**: Windows Search Indexer is consuming 100% of your disk I/O.
> 3. **CPU**: Overall 92% CPU load from Chrome + Teams + Outlook.
>
> Recommended actions:
> - Close unused Chrome tabs (I can see 47 open)
> - Restart Windows Search service to stop the indexer storm
> - Restart Teams (it has a known memory leak)
>
> Shall I proceed with these fixes?

---

## Scenario 4: Cross-Device File Transfer

**Goal**: AI copies a report from Device A's desktop to Device B's shared folder.

**Prompt**: Use the built-in `multi_device_workflow` prompt.

### Conversation Example

> **User**: Copy the file "Q4_Report.xlsx" from my office PC (ID: 111111111, code: 999999) desktop to my home PC (ID: 222222222, code: 888888) Documents folder.

**AI Agent Workflow**:

```
Phase 1: Connect both devices
  conn_office = connect_device("111111111", "999999", show_window=true)
  conn_home   = connect_device("222222222", "888888", show_window=true)

Phase 2: On office PC — copy file path
  screenshot(conn_office)
  → See desktop with Q4_Report.xlsx
  mouse_click on Q4_Report.xlsx → right-click → "Copy as path"
  get_clipboard(conn_office)
  → "C:\Users\user\Desktop\Q4_Report.xlsx"

Phase 3: Transfer strategy
  (Since there is no direct file transfer API, use a shared method):
  
  Option A: If both PCs are on the same network:
    On office PC: Open PowerShell
    keyboard_type → "Copy-Item 'C:\Users\user\Desktop\Q4_Report.xlsx' '\\home-pc\shared\'"
  
  Option B: Use clipboard for small text files:
    Open file, Ctrl+A, Ctrl+C
    get_clipboard → file content
    Switch to home PC
    set_clipboard → paste content
    Save file

Phase 4: Verify
  screenshot(conn_home) → navigate to Documents folder
  → Confirm Q4_Report.xlsx exists
  
Phase 5: Cleanup
  disconnect_device(conn_office)
  disconnect_device(conn_home)
```

---

## Scenario 5: Automated SOP Documentation

**Goal**: AI performs a procedure and documents every step as an SOP.

**Prompt**: Use the built-in `document_procedure` prompt.

### Conversation Example

> **User**: Document the procedure for restarting our web service on the production server (ID: 333444555, code: 777777).

**AI Agent Workflow**:

```
1. connect_device → screenshot → "Desktop with IIS Manager icon"
   📝 Step 0: Prerequisites — RDP access to production server

2. keyboard_hotkey(["win", "r"]) → keyboard_type("services.msc") → Enter
   📸 Before: Desktop | After: Services console open
   📝 Step 1: Open Services Manager (Win+R → services.msc)

3. keyboard_type("W3SVC") → scroll to find World Wide Web Publishing Service
   📸 Before: Services list | After: W3SVC selected
   📝 Step 2: Locate "World Wide Web Publishing Service"

4. right-click → "Restart"
   📸 Before: Service running | After: Service restarting
   📝 Step 3: Right-click the service → select "Restart"

5. Wait 10 seconds, screenshot
   📸 Service status: "Running"
   📝 Step 4: Verify service status returns to "Running"

6. Open browser → navigate to https://localhost
   📸 Website loads successfully
   📝 Step 5: Verify the website is accessible

7. disconnect_device
```

**Generated SOP**:

```markdown
# Procedure: Restart Web Service on Production Server

## Prerequisites
- Remote access to production server (Device ID: 333444555)
- Administrator privileges

## Steps

### Step 1: Open Services Manager
**Action**: Press Win+R, type "services.msc", press Enter
**Expected Result**: Windows Services console opens
**Time**: ~3 seconds

### Step 2: Locate W3SVC Service
**Action**: Type "W3SVC" to jump to World Wide Web Publishing Service
**Expected Result**: Service is highlighted in the list

### Step 3: Restart the Service
**Action**: Right-click → Restart
**Expected Result**: Service stops then starts (status changes to "Running")
**Warning**: Website will be briefly unavailable during restart

### Step 4: Verify Service Status
**Action**: Wait 10 seconds, check status column
**Expected Result**: Status shows "Running"
**If Failed**: Check Event Viewer for errors

### Step 5: Verify Website
**Action**: Open browser → navigate to https://localhost
**Expected Result**: Website loads without errors

## Troubleshooting
- If service fails to start: check Event Viewer → Windows Logs → Application
- If website doesn't load: check IIS configuration, port bindings

## Summary
- Total steps: 5
- Estimated time: 2 minutes
- Difficulty: Easy
```

---

## Scenario 6: Screen Content Analysis for Security Audit

**Goal**: AI analyzes the remote desktop for sensitive information exposure.

**Prompt**: Use the built-in `analyze_screen_content` prompt.

### Conversation Example

> **User**: Audit what's visible on the developer workstation (ID: 555666777, code: 444444). Check for any sensitive info.

**AI Agent Workflow**:

```
1. connect_device → screenshot (full resolution, no max_width)
2. Analyze every visible element on screen
```

**AI Report**:

> ## Screen Content Analysis
>
> **OS**: Windows 11, Dark theme
> **Open Applications**: VS Code, Chrome (3 tabs), Slack, Terminal
>
> ### Security Findings
>
> **⚠️ WARNING — Sensitive Information Detected:**
>
> 1. **VS Code** (active window): File `.env` is open
>    - Line 3: `DATABASE_URL=postgres://admin:P@ssw0rd123@prod-db.company.com:5432/main`
>    - Line 5: `AWS_SECRET_KEY=AKIA...redacted...`
>    - **Risk**: Database credentials and AWS key are visible in plaintext
>
> 2. **Chrome Tab**: "AWS Console" — logged in as root user
>    - **Risk**: Root account usage violates least-privilege principle
>
> 3. **Terminal**: `git log` output shows commit with message "added api key"
>    - **Risk**: Secrets may have been committed to version control
>
> **Recommendations**:
> - Immediately rotate the exposed database password and AWS key
> - Use a secrets manager (AWS Secrets Manager, HashiCorp Vault)
> - Set up pre-commit hooks to prevent secret commits (e.g., git-secrets)
> - Create an IAM user instead of using root account

---

## Quick Reference: Which Prompt for Which Scenario

| Scenario | MCP Prompt | Key Tools Used |
|----------|------------|----------------|
| Check server health | `server_health_check` | `connect_device`, `screenshot`, `keyboard_type`, `keyboard_hotkey`, `get_clipboard` |
| Batch operations on N devices | `batch_operation` | `connect_device(show_window=false)`, `screenshot`, `keyboard_type`, `disconnect_device` |
| Diagnose slow/broken computer | `diagnose_system_issue` | `connect_device`, `screenshot`, `keyboard_hotkey`, `mouse_click` |
| Cross-device workflow | `multi_device_workflow` | `connect_device` ×N, `get_clipboard`, `set_clipboard` |
| Document a procedure as SOP | `document_procedure` | `screenshot` (before+after each step), all input tools |
| Security screen audit | `analyze_screen_content` | `screenshot` (full resolution) |
| General remote operation | `operate_remote_desktop` | All tools |
| Find and click a UI element | `find_and_click` | `screenshot`, `get_screen_size`, `mouse_click` |
| Run a command in terminal | `run_command` | `keyboard_hotkey`, `keyboard_type`, `screenshot` |
