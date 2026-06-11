# TURN/STUN 服务器部署指南（coturn）

## 概述

当两个 QuickDesk 客户端无法建立 WebRTC P2P 直连时（例如双方均处于对称型 NAT 后），所有媒体流量都需要通过 TURN 服务器进行中继。本文档介绍如何部署 **coturn** —— 最广泛使用的开源 TURN/STUN 实现。

### 鉴权模式

coturn 支持两种鉴权机制，**二选一，不可同时使用：**

| 模式 | 配置指令 | 适用场景 |
|------|----------|----------|
| **长期静态凭据** | `lt-cred-mech` + `user=` | 静态用户，简单部署 |
| **REST API / HMAC 临时凭据** | `use-auth-secret` + `static-auth-secret=` | 与 QuickDesk 信令服务器集成；凭据有时效性、按需生成 |

在生产环境中与 QuickDesk 信令服务器配合使用时，**推荐 REST API 模式** —— 信令服务器通过 `/v1/ice-config` 接口自动生成短期 TURN 凭据，客户端无需持有静态密码。

---

## 系统要求

- CentOS 7/8/Stream、Rocky Linux 8+ 或 Ubuntu 20.04+
- 1GB+ 内存
- 公网 IP 地址（TURN 中继必须）
- 开放 UDP/TCP 端口（见防火墙章节）

---

## 1. 安装 coturn

### CentOS / Rocky Linux

```bash
sudo yum install -y epel-release
sudo yum install -y coturn
```

### Ubuntu / Debian

```bash
sudo apt update
sudo apt install -y coturn
```

### 验证安装

```bash
turnserver --version
```

---

## 2. 配置 coturn

默认配置文件路径：Debian/Ubuntu 为 `/etc/coturn/turnserver.conf`，CentOS 为 `/etc/turnserver.conf`。

```bash
sudo cp /etc/coturn/turnserver.conf /etc/coturn/turnserver.conf.bak  # 先备份
sudo tee /etc/coturn/turnserver.conf > /dev/null << 'EOF'
# =============================================================================
# coturn 配置文件（QuickDesk 使用）
# =============================================================================

# 服务器的公网 IP 地址。
# 替换为你服务器实际的外网 IP，TURN 中继正确工作所必需。
external-ip=115.190.196.189

# 如果服务器有特定的内网网卡 IP，可按如下格式同时指定：
# external-ip=<公网IP>/<内网IP>
# 示例：external-ip=115.190.196.189/10.0.0.5

# ---------------------------------------------------------------------------
# 监听端口
# ---------------------------------------------------------------------------

# STUN/TURN 监听端口（默认 3478）
listening-port=3478

# TLS 监听端口（默认 5349，仅在启用 TLS 时需要）
# tls-listening-port=5349

# UDP 中继端口范围，用于传输媒体数据。
# 每个活跃的 TURN allocation 会占用该范围内的一个端口。
# 请确保防火墙已开放这段端口。
min-port=49152
max-port=65535

# ---------------------------------------------------------------------------
# 域名 / Realm
# ---------------------------------------------------------------------------

# Realm 用于 TURN 鉴权质询，通常设为你的域名。
realm=turn.example.com

# server-name（部分客户端握手用到，默认与 realm 相同）
# server-name=turn.example.com

# ---------------------------------------------------------------------------
# TLS（可选，生产环境推荐）
# ---------------------------------------------------------------------------

# 禁用 TLS —— 如需启用 TLS，注释掉这两行
no-tls
no-dtls

# 启用 TLS 时，注释上面两行并配置证书：
# cert=/etc/letsencrypt/live/turn.example.com/fullchain.pem
# pkey=/etc/letsencrypt/live/turn.example.com/privkey.pem

# ---------------------------------------------------------------------------
# 日志
# ---------------------------------------------------------------------------

# 不向标准输出打印日志（systemd 管理的服务推荐关闭）
no-stdout-log

# 日志文件路径
log-file=/var/log/coturn/turn.log

# 开启详细调试日志（非常冗长，仅用于排错时启用）
# verbose

# 在 TURN 消息中添加 fingerprint（提升兼容性，推荐开启）
fingerprint

# ---------------------------------------------------------------------------
# 鉴权模式 —— 以下两种模式选择其一
# ---------------------------------------------------------------------------

# ~~~ 模式一：长期静态凭据 ~~~
# 适合固定用户名/密码的简单场景。
# 可添加多条 user= 行。
#
# lt-cred-mech
# user=alice:password1
# user=bob:password2

# ~~~ 模式二：REST API / HMAC 临时凭据（推荐）~~~
# 与 QuickDesk 信令服务器集成时使用此模式。
# 信令服务器利用这个共享密钥，通过 /v1/ice-config 接口
# 为每个客户端实时生成有时效限制的 TURN 凭据（通常 1 小时有效）。
# 即使凭据被截获，过期后也无法再使用，安全性更高。
#
use-auth-secret
static-auth-secret=your-strong-secret-here

# ---------------------------------------------------------------------------
# 中继协议选项
# ---------------------------------------------------------------------------

# 关闭 UDP 中继（仅使用 TCP 中继）
# no-udp-relay

# 关闭 TCP 中继（仅使用 UDP 中继）
# no-tcp-relay

# 仅作为 STUN 服务器运行（不提供 TURN 中继功能）
# stun-only

EOF
```

> **安全提示：** 将 `your-strong-secret-here` 替换为至少 32 位的随机字符串，生成命令：
> ```bash
> openssl rand -hex 32
> ```

---

## 3. 创建日志目录

```bash
sudo mkdir -p /var/log/coturn
sudo chown turnserver:turnserver /var/log/coturn
```

---

## 4. 启用并启动服务

### CentOS / Rocky Linux

```bash
sudo systemctl enable coturn
sudo systemctl start coturn

# 查看状态
sudo systemctl status coturn
```

### Ubuntu / Debian

```bash
# 需要先在 /etc/default/coturn 中启用服务
sudo sed -i 's/^#TURNSERVER_ENABLED=1/TURNSERVER_ENABLED=1/' /etc/default/coturn

sudo systemctl enable coturn
sudo systemctl start coturn

# 查看状态
sudo systemctl status coturn
```

---

## 5. 配置防火墙

```bash
# 开放 STUN/TURN 控制端口（UDP + TCP）
sudo firewall-cmd --permanent --add-port=3478/udp
sudo firewall-cmd --permanent --add-port=3478/tcp

# 如果启用了 TLS，额外开放 TLS 端口
# sudo firewall-cmd --permanent --add-port=5349/udp
# sudo firewall-cmd --permanent --add-port=5349/tcp

# 开放 UDP 中继端口范围（需与配置文件中 min-port/max-port 一致）
sudo firewall-cmd --permanent --add-port=49152-65535/udp

sudo firewall-cmd --reload

# 确认端口已开放
sudo firewall-cmd --list-ports
```

Ubuntu ufw 等效命令：

```bash
sudo ufw allow 3478/udp
sudo ufw allow 3478/tcp
sudo ufw allow 49152:65535/udp
sudo ufw reload
```

---

## 6. 与 QuickDesk 信令服务器集成（REST API 模式）

使用 **REST API 模式**（`use-auth-secret`）时，需要在信令服务器的 `.env` 中配置相同的 `static-auth-secret`：

```bash
# /opt/quickdesk/signaling/.env
TURN_AUTH_SECRET=your-strong-secret-here      # 与 coturn 中的 static-auth-secret 完全一致
TURN_URLS=turn:115.190.196.189:3478
STUN_URLS=stun:115.190.196.189:3478
```

配置完成后，信令服务器将通过 `/v1/ice-config` 接口为客户端返回短期凭据：

```json
{
  "iceServers": [
    {
      "urls": "turn:115.190.196.189:3478",
      "username": "1700000000:user",
      "credential": "<基于 HMAC-SHA1 的签名>"
    }
  ]
}
```

客户端使用这些临时凭据 —— 即使被截获，超过有效期（通常 1 小时）后也无法再次使用。

---

## 7. 验证部署

### 检查服务状态

```bash
sudo systemctl status coturn

# 实时跟踪日志
sudo tail -f /var/log/coturn/turn.log
```

### 测试 STUN 连通性

```bash
# 简单 UDP 连通测试（有响应说明端口可达）
echo -n "" | nc -u 115.190.196.189 3478 -w 2
```

### 在线测试工具

访问 https://webrtc.github.io/samples/src/content/peerconnection/trickle-ice/ —— 输入 TURN 服务器 URL 和凭据，确认 allocation 成功。

---

## 8. 查看日志

```bash
# 实时跟踪日志文件
sudo tail -f /var/log/coturn/turn.log

# 通过 systemd journal 查看
sudo journalctl -u coturn -f

# 筛选错误日志
sudo grep -i "error\|fail\|denied" /var/log/coturn/turn.log
```

---

## 9. 日志轮转配置

```bash
sudo tee /etc/logrotate.d/coturn > /dev/null << 'EOF'
/var/log/coturn/turn.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    postrotate
        /bin/kill -HUP $(cat /var/run/coturn.pid 2>/dev/null) 2>/dev/null || true
    endscript
}
EOF
```

---

## 故障排查

### 401 鉴权失败

- **长期凭据模式：** 检查配置文件中的 `user=` 与客户端发送的用户名/密码是否完全一致。
- **REST API 模式：** 确认 coturn 与信令服务器中的 `static-auth-secret` / `TURN_AUTH_SECRET` 字符串完全相同。同时检查系统时钟是否同步——HMAC 凭据对时间敏感，时钟偏差过大会导致鉴权失败。

```bash
# 检查 NTP 时间同步状态
timedatectl status
sudo systemctl enable --now chronyd  # 或 ntpd
```

### UDP 中继无流量

- 确认 `min-port`/`max-port` 范围已在防火墙中开放。
- 检查 `external-ip` 是否设置为正确的公网 IP（不能填内网 IP）。

### 服务无法启动

```bash
# 检查配置语法
sudo turnserver -c /etc/coturn/turnserver.conf --check-config

# 查看启动错误
sudo journalctl -u coturn -n 50 --no-pager
```

### 端口被占用

```bash
sudo lsof -i:3478
```

---

## 安全建议

1. **生产环境使用 REST API 模式** —— 静态密码永远不会下发给客户端
2. **配置 `denied-peer-ip`** —— 防止服务器被用于攻击内网服务（SSRF）
3. **启用 TLS**（删除 `no-tls`/`no-dtls`，配置证书）—— 加密 TURN 流量
4. **定期轮换 `static-auth-secret`** —— 同步更新 coturn 和信令服务器配置
5. **仅开放必要端口** —— 只开放 UDP 49152-65535，不要无限制地开放全部端口
6. **监控 allocation 数量** —— 异常高的 allocation 数量可能意味着服务器被滥用
