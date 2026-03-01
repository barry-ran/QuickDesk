# QuickDesk MCP — 典型场景 Demo

本文档展示 AI Agent 使用 QuickDesk MCP 工具的典型工作流。每个场景展示完整的工具调用序列和预期行为。

## 场景 1：远程服务器健康检查

**目标**：AI Agent 连接远程服务器，执行全面的健康检查。

**Prompt**：使用内置 `server_health_check` 提示词模板。

### 对话示例

> **用户**：检查一下我服务器的健康状态。设备 ID：123456789，访问码：888888。

**AI Agent 工作流**：

```
步骤 1: connect_device
  → device_id: "123456789"
  → access_code: "888888"
  → show_window: true
  ← connection_id: "abc123"

步骤 2: screenshot
  → connection_id: "abc123", max_width: 1280
  ← [图片：Windows Server 2022 桌面]

步骤 3: get_screen_size
  → connection_id: "abc123"
  ← { width: 1920, height: 1080 }

步骤 4: keyboard_hotkey → keys: ["win", "r"]
  （打开运行对话框）

步骤 5: screenshot → 确认运行对话框已打开

步骤 6: keyboard_type → text: "powershell"

步骤 7: keyboard_hotkey → keys: ["enter"]

步骤 8: screenshot → 确认 PowerShell 已打开

步骤 9-11: 执行 systeminfo，截图捕获输出
  （AI 读取：Windows Server 2022, 32GB RAM, 28GB 使用中）

步骤 12-14: 执行磁盘检查命令，截图
  （AI 读取：C: 已用 95%, D: 已用 40%）

步骤 15-17: 执行进程列表命令，截图

步骤 18: disconnect_device → connection_id: "abc123"
```

**AI 报告**：
> 服务器健康检查完成：
> - **操作系统**：Windows Server 2022 Standard
> - **CPU**：平均负载 45% — 正常
> - **内存**：28/32 GB 已使用 (87%) — **警告**：内存使用偏高
> - **磁盘 C:**：已用 95% — **严重**：需要立即清理
> - **磁盘 D:**：已用 40% — 正常
> - **最大进程**：sqlservr.exe 占用 12GB 内存
> - **综合判定**：警告 — C 盘接近满载，建议清理临时文件和旧日志。

---

## 场景 2：批量软件更新

**目标**：AI Agent 在 3 台远程设备上安装最新版 Chrome。

**Prompt**：使用内置 `batch_operation` 提示词模板。

### 对话示例

> **用户**：在下面 3 台机器上安装最新版 Chrome：
> - 服务器 A：ID 111222333，访问码 111111
> - 服务器 B：ID 444555666，访问码 222222
> - 服务器 C：ID 777888999，访问码 333333

**AI Agent 工作流**：

```
对每台设备：

  1. connect_device(show_window=false)  ← 后台模式，不弹窗
  2. screenshot → 识别操作系统
  3. 打开终端
  4. 执行安装命令：
     Windows: "winget install Google.Chrome --accept-package-agreements"
     Linux:   "wget ... && sudo dpkg -i ..."
  5. 等待安装完成，定期截图监控
  6. 确认安装成功
  7. disconnect_device
```

**AI 报告**：

| 设备 | ID | 操作系统 | 结果 |
|------|-----|---------|------|
| 服务器 A | 111222333 | Windows 11 | ✅ Chrome 125 已安装 |
| 服务器 B | 444555666 | Windows 10 | ✅ Chrome 125 已安装 |
| 服务器 C | 777888999 | Ubuntu 22.04 | ✅ Chrome 125 已安装 |

> 批量操作完成：3/3 台设备更新成功。

---

## 场景 3：远程故障诊断

**目标**：AI Agent 诊断用户电脑运行缓慢的原因。

**Prompt**：使用内置 `diagnose_system_issue` 提示词模板。

### 对话示例

> **用户**：我的工作电脑今天特别卡。设备 ID：999888777，访问码：123456。

**AI Agent 工作流**：

```
步骤 1: connect_device → screenshot
  （看到桌面打开了很多窗口）

步骤 2: keyboard_hotkey(["ctrl", "shift", "esc"])
  （打开任务管理器）

步骤 3: screenshot
  （AI 读取：内存 15.2/16 GB，CPU 92%，磁盘 100%）

步骤 4: 点击"内存"列排序 → screenshot
  （chrome.exe — 8.3 GB, Teams — 2.1 GB, outlook.exe — 1.8 GB）

步骤 5: 点击"磁盘"列排序 → screenshot
  （SearchIndexer.exe — 100% 磁盘, MsMpEng.exe — 50%）
```

**AI 响应**：
> 我找到了问题原因。你的电脑卡顿是因为三个问题：
>
> 1. **内存**：Chrome 占用 8.3 GB（总内存的 52%），你开了 47 个标签页
> 2. **磁盘**：Windows 搜索索引器正在以 100% 的速度读写磁盘
> 3. **CPU**：Chrome + Teams + Outlook 合计占用 92% CPU
>
> 建议操作：
> - 关闭不用的 Chrome 标签页（当前 47 个）
> - 重启 Windows Search 服务，停止索引风暴
> - 重启 Teams（已知存在内存泄漏问题）
>
> 需要我执行这些修复操作吗？

---

## 场景 4：跨设备文件操作

**目标**：AI 将设备 A 桌面上的报告复制到设备 B 的文档目录。

**Prompt**：使用内置 `multi_device_workflow` 提示词模板。

### 对话示例

> **用户**：把办公电脑（ID: 111111111，访问码: 999999）桌面上的"Q4_Report.xlsx"复制到家里电脑（ID: 222222222，访问码: 888888）的文档目录。

**AI Agent 工作流**：

```
阶段 1：连接两台设备
  conn_办公 = connect_device("111111111", "999999", show_window=true)
  conn_家里 = connect_device("222222222", "888888", show_window=true)

阶段 2：在办公电脑上获取文件路径
  screenshot(conn_办公) → 找到 Q4_Report.xlsx
  右键 → "复制文件路径"
  get_clipboard(conn_办公) → "C:\Users\user\Desktop\Q4_Report.xlsx"

阶段 3：传输
  如果两台电脑在同一网络：
    PowerShell: Copy-Item '...' '\\home-pc\shared\'
  如果不在同一网络：
    利用剪贴板传输文本内容，或建议用户使用云存储中转

阶段 4：验证
  screenshot(conn_家里) → 导航到文档目录
  → 确认文件已到达

阶段 5：清理
  disconnect_device(conn_办公)
  disconnect_device(conn_家里)
```

---

## 场景 5：自动化 SOP 文档生成

**目标**：AI 执行操作流程，同时记录每一步生成 SOP 文档。

**Prompt**：使用内置 `document_procedure` 提示词模板。

### 对话示例

> **用户**：记录一下在生产服务器上重启 Web 服务的操作流程（ID: 333444555，访问码: 777777）。

**AI 生成的 SOP**：

```markdown
# 操作流程：重启生产服务器 Web 服务

## 前提条件
- 具有生产服务器的远程访问权限
- 管理员权限

## 操作步骤

### 步骤 1：打开服务管理器
**操作**：按 Win+R，输入 "services.msc"，按回车
**预期结果**：Windows 服务管理控制台打开
**耗时**：约 3 秒

### 步骤 2：定位 Web 服务
**操作**：输入 "W3SVC" 跳转到 World Wide Web Publishing Service
**预期结果**：服务在列表中高亮显示

### 步骤 3：重启服务
**操作**：右键 → 重新启动
**预期结果**：服务先停止后启动（状态变为"正在运行"）
**⚠️ 注意**：重启期间网站将短暂不可用

### 步骤 4：验证服务状态
**操作**：等待 10 秒，检查状态列
**预期结果**：状态显示"正在运行"
**如果失败**：查看事件查看器中的错误日志

### 步骤 5：验证网站
**操作**：打开浏览器 → 访问 https://localhost
**预期结果**：网站正常加载

## 故障排除
- 服务启动失败：检查事件查看器 → Windows 日志 → 应用程序
- 网站无法加载：检查 IIS 配置和端口绑定

## 总结
- 总步骤数：5
- 预计耗时：2 分钟
- 难度：简单
```

---

## 场景 6：屏幕安全审计

**目标**：AI 分析远程桌面上是否有敏感信息暴露。

**Prompt**：使用内置 `analyze_screen_content` 提示词模板。

### 对话示例

> **用户**：审计一下开发工作站上是否有敏感信息暴露（ID: 555666777，访问码: 444444）。

**AI 报告**：

> ## 屏幕内容分析
>
> **操作系统**：Windows 11，深色主题
> **打开的应用**：VS Code、Chrome（3 个标签页）、Slack、终端
>
> ### 安全发现
>
> **⚠️ 警告 — 检测到敏感信息：**
>
> 1. **VS Code**（活动窗口）：`.env` 文件已打开
>    - 第 3 行：`DATABASE_URL=postgres://admin:P@ssw0rd123@prod-db.company.com:5432/main`
>    - 第 5 行：`AWS_SECRET_KEY=AKIA...已脱敏...`
>    - **风险**：数据库凭据和 AWS 密钥以明文形式可见
>
> 2. **Chrome 标签页**："AWS Console" — 以 root 用户登录
>    - **风险**：使用 root 账户违反最小权限原则
>
> 3. **终端**：`git log` 输出显示提交信息 "added api key"
>    - **风险**：密钥可能已提交到版本控制
>
> **建议**：
> - 立即轮换暴露的数据库密码和 AWS 密钥
> - 使用密钥管理器（AWS Secrets Manager、HashiCorp Vault）
> - 设置 pre-commit hook 防止密钥提交（如 git-secrets）
> - 创建 IAM 用户，停止使用 root 账户

---

## 快速参考：场景 → Prompt 对照表

| 场景 | MCP Prompt | 主要使用的工具 |
|------|------------|---------------|
| 服务器健康检查 | `server_health_check` | `connect_device`, `screenshot`, `keyboard_type`, `keyboard_hotkey`, `get_clipboard` |
| 批量操作 N 台设备 | `batch_operation` | `connect_device(show_window=false)`, `screenshot`, `keyboard_type`, `disconnect_device` |
| 诊断电脑故障 | `diagnose_system_issue` | `connect_device`, `screenshot`, `keyboard_hotkey`, `mouse_click` |
| 跨设备工作流 | `multi_device_workflow` | `connect_device` ×N, `get_clipboard`, `set_clipboard` |
| 文档化操作流程 | `document_procedure` | `screenshot`（每步前后截图）, 所有输入工具 |
| 屏幕安全审计 | `analyze_screen_content` | `screenshot`（全分辨率） |
| 通用远程操作 | `operate_remote_desktop` | 所有工具 |
| 查找并点击元素 | `find_and_click` | `screenshot`, `get_screen_size`, `mouse_click` |
| 在终端运行命令 | `run_command` | `keyboard_hotkey`, `keyboard_type`, `screenshot` |
