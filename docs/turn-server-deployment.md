# TURN/STUN Server Deployment Guide (coturn)

## Overview

When two QuickDesk clients cannot establish a direct WebRTC P2P connection (e.g., both behind symmetric NATs), all media traffic must be relayed through a TURN server. This guide explains how to deploy **coturn** — the most widely used open-source TURN/STUN implementation.

### Credential Modes

coturn supports two authentication mechanisms. **Choose one — they are mutually exclusive:**

| Mode | Config Directives | Use Case |
|------|-------------------|----------|
| **Long-Term Credentials** | `lt-cred-mech` + `user=` | Static users, simple setups |
| **REST API / HMAC Temporary Credentials** | `use-auth-secret` + `static-auth-secret=` | Integration with QuickDesk signaling server; credentials are time-limited and generated on demand |

For production use with the QuickDesk signaling server, **REST API mode is recommended** — the signaling server's `/v1/ice-config` endpoint automatically generates short-lived TURN credentials so no static passwords are exposed to clients.

---

## System Requirements

- CentOS 7/8/Stream, Rocky Linux 8+, or Ubuntu 20.04+
- 1GB+ RAM
- Public IP address (required for TURN relaying)
- Open UDP/TCP ports (see firewall section)

---

## 1. Install coturn

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

### Verify installation

```bash
turnserver --version
```

---

## 2. Configure coturn

The default configuration file is `/etc/coturn/turnserver.conf` (Debian/Ubuntu) or `/etc/turnserver.conf` (CentOS).

```bash
sudo cp /etc/coturn/turnserver.conf /etc/coturn/turnserver.conf.bak  # backup
sudo tee /etc/coturn/turnserver.conf > /dev/null << 'EOF'
# =============================================================================
# coturn Configuration for QuickDesk
# =============================================================================

# Public IP address of this server.
# Replace with your actual server's external/public IP.
# Required for proper TURN relaying behind NAT.
external-ip=115.190.196.189

# If the server has a specific local (internal) NIC IP, specify it here.
# Format: external-ip=<public-ip>/<private-ip>
# Example: external-ip=115.190.196.189/10.0.0.5

# ---------------------------------------------------------------------------
# Network Ports
# ---------------------------------------------------------------------------

# STUN/TURN listening port (default 3478)
listening-port=3478

# TLS listening port (default 5349) — only needed when TLS is enabled
# tls-listening-port=5349

# UDP relay port range for media traffic.
# Each active TURN allocation uses one port from this range.
# Ensure these ports are open in your firewall.
min-port=49152
max-port=65535

# ---------------------------------------------------------------------------
# Identity / Realm
# ---------------------------------------------------------------------------

# Realm is used in TURN authentication challenges.
# Typically set to your domain name.
realm=turn.example.com

# Server name (used in some client handshakes, defaults to realm if unset)
# server-name=turn.example.com

# ---------------------------------------------------------------------------
# TLS (optional but recommended for production)
# ---------------------------------------------------------------------------

# Disable TLS — comment these out if you want TLS
no-tls
no-dtls

# To enable TLS, comment out the two lines above and configure:
# cert=/etc/letsencrypt/live/turn.example.com/fullchain.pem
# pkey=/etc/letsencrypt/live/turn.example.com/privkey.pem

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------

# Do not print logs to stdout (recommended for systemd-managed services)
no-stdout-log

# Log file path
log-file=/var/log/coturn/turn.log

# Uncomment for detailed debug logging (very verbose, use only for debugging)
# verbose

# Add fingerprint to TURN messages (recommended for compatibility)
fingerprint

# ---------------------------------------------------------------------------
# Authentication — Choose ONE of the two modes below
# ---------------------------------------------------------------------------

# ~~~ MODE 1: Long-Term Static Credentials ~~~
# Use this for simple setups with fixed usernames/passwords.
# Add as many `user=` lines as needed.
#
# lt-cred-mech
# user=alice:password1
# user=bob:password2

# ~~~ MODE 2: REST API / HMAC Temporary Credentials (Recommended) ~~~
# Use this to integrate with the QuickDesk signaling server.
# The signaling server uses this shared secret to generate time-limited
# TURN credentials on behalf of clients via /v1/ice-config.
# Credentials expire after a short window (typically 1 hour), so even if
# intercepted they cannot be reused indefinitely.
#
use-auth-secret
static-auth-secret=your-strong-secret-here

# ---------------------------------------------------------------------------
# Relay Protocol Options
# ---------------------------------------------------------------------------

# Uncomment to disable UDP relay (forces TCP relay only)
# no-udp-relay

# Uncomment to disable TCP relay (forces UDP relay only)
# no-tcp-relay

# Uncomment to act as a STUN-only server (no TURN relaying at all)
# stun-only

EOF
```

> **Security note:** Replace `your-strong-secret-here` with a random string of at least 32 characters. Generate one with:
> ```bash
> openssl rand -hex 32
> ```

---

## 3. Create Log Directory

```bash
sudo mkdir -p /var/log/coturn
sudo chown turnserver:turnserver /var/log/coturn  # CentOS/Rocky
# or
sudo chown turnserver:turnserver /var/log/coturn  # Ubuntu
```

---

## 4. Enable and Start the Service

### CentOS / Rocky Linux

```bash
# Enable the service
sudo systemctl enable coturn
sudo systemctl start coturn

# Check status
sudo systemctl status coturn
```

### Ubuntu / Debian

```bash
# Enable the service (must also set TURNSERVER_ENABLED=1)
sudo sed -i 's/^#TURNSERVER_ENABLED=1/TURNSERVER_ENABLED=1/' /etc/default/coturn

sudo systemctl enable coturn
sudo systemctl start coturn

# Check status
sudo systemctl status coturn
```

---

## 5. Configure Firewall

```bash
# Open STUN/TURN control port (UDP + TCP)
sudo firewall-cmd --permanent --add-port=3478/udp
sudo firewall-cmd --permanent --add-port=3478/tcp

# Open TLS port if TLS is enabled
# sudo firewall-cmd --permanent --add-port=5349/udp
# sudo firewall-cmd --permanent --add-port=5349/tcp

# Open UDP relay port range (must match min-port/max-port in config)
sudo firewall-cmd --permanent --add-port=49152-65535/udp

sudo firewall-cmd --reload

# Verify
sudo firewall-cmd --list-ports
```

For Ubuntu ufw:

```bash
sudo ufw allow 3478/udp
sudo ufw allow 3478/tcp
sudo ufw allow 49152:65535/udp
sudo ufw reload
```

---

## 6. Integrate with QuickDesk Signaling Server (REST API Mode)

When using **REST API mode** (`use-auth-secret`), configure the same `static-auth-secret` in the signaling server's `.env`:

```bash
# /opt/quickdesk/signaling/.env
TURN_AUTH_SECRET=your-strong-secret-here
TURN_URLS=turn:115.190.196.189:3478
STUN_URLS=stun:115.190.196.189:3478
```

The signaling server will then respond to `/v1/ice-config` requests with short-lived credentials:

```json
{
  "iceServers": [
    {
      "urls": "turn:115.190.196.189:3478",
      "username": "1700000000:user",
      "credential": "<hmac-sha1-of-username>"
    }
  ]
}
```

Clients use these temporary credentials — they are valid only for a limited time window (typically 1 hour) even if intercepted.

---

## 7. Verify Deployment

### Check service status

```bash
sudo systemctl status coturn

# Tail logs
sudo tail -f /var/log/coturn/turn.log
```

### Test STUN connectivity

```bash
# Install stun-client or use an online STUN tester
# Quick test with nc (should get a response on UDP 3478)
echo -n "" | nc -u 115.190.196.189 3478 -w 2
```

### Online TESTER

Use https://webrtc.github.io/samples/src/content/peerconnection/trickle-ice/ — enter your TURN server URL and credentials to verify allocations succeed.

---

## 8. View Logs

```bash
# Follow live logs
sudo tail -f /var/log/coturn/turn.log

# View systemd journal
sudo journalctl -u coturn -f

# Check for allocation errors
sudo grep -i "error\|fail\|denied" /var/log/coturn/turn.log
```

---

## 9. Log Rotation

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

## Troubleshooting

### Allocations Fail with 401 Unauthorized

- **Long-term mode:** Check that the `user=` credentials in the config exactly match what the client sends.
- **REST API mode:** Ensure `static-auth-secret` in coturn and the signaling server are identical. Verify system clocks are synchronized (NTP) — HMAC credentials are time-sensitive.

```bash
# Check time synchronization
timedatectl status
sudo systemctl enable --now chronyd  # or ntpd
```

### No UDP Relay Traffic

- Verify the `min-port`/`max-port` range is open in the firewall.
- Check `external-ip` is set to the correct public IP (not a private/internal IP).

### Service Fails to Start

```bash
# Check config syntax
sudo turnserver -c /etc/coturn/turnserver.conf --check-config

# View startup errors
sudo journalctl -u coturn -n 50 --no-pager
```

### Port Already in Use

```bash
sudo lsof -i:3478
```

---

## Security Recommendations

1. **Use REST API mode in production** — static user passwords never reach clients
2. **Set `denied-peer-ip`** — prevents SSRF attacks targeting internal services
3. **Enable TLS** (`no-tls` / `no-dtls` removed, certificate configured) for encrypted TURN traffic
4. **Rotate `static-auth-secret` periodically** — update both coturn and the signaling server
5. **Firewall the relay port range** — only open UDP 49152-65535, not all ports
6. **Monitor allocations** — excessive allocations may indicate abuse
