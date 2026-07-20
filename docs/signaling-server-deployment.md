# QuickDesk Signaling Server Deployment Guide

## Quick Deploy (Recommended)

All-in-one Docker image including PostgreSQL + Redis + signaling server. Three deployment methods are available:

### Prepare Configuration

```bash
git clone git@github.com:barry-ran/QuickDesk.git
cd QuickDesk/SignalingServer

# Copy and edit your configuration
cp .env.example .env
vim .env
```

Configuration reference:

**Required (infrastructure):**

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_HOST` | 0.0.0.0 | Listen address |
| `SERVER_PORT` | 8000 | Server port |
| `DB_HOST` | localhost | Database host |
| `DB_PORT` | 5432 | Database port |
| `DB_USER` | quickdesk | Database user |
| `DB_PASSWORD` | quickdesk123 | Database password |
| `DB_NAME` | quickdesk_signaling | Database name |
| `REDIS_HOST` | localhost | Redis host |
| `REDIS_PORT` | 6379 | Redis port |
| `REDIS_PASSWORD` | (empty) | Redis password |
| `ADMIN_USER` | admin | Initial admin username (first deploy only) |
| `ADMIN_PASSWORD` | admin | Initial admin password (first deploy only) |
| `RUN_AUTO_MIGRATE` | false | Run GORM AutoMigrate at startup (temporarily enable for first initialization/schema upgrades; keep false for stable production runs) |

> **Database initialization/upgrades:** `RUN_AUTO_MIGRATE` defaults to `false` to avoid scanning database metadata on every startup. For a fresh database, or when a release adds tables/columns, set `RUN_AUTO_MIGRATE=true` temporarily and start the server once. After it starts successfully, set it back to `false` and restart.

**Optional (can be configured in admin panel after deployment):**

These parameters can be preset in `.env` or configured later in the admin panel (Settings page). Changes take effect immediately without restart.

| Variable | Default | Description |
|----------|---------|-------------|
| `TURN_URLS` | (empty) | TURN server URLs (comma-separated) |
| `TURN_AUTH_SECRET` | (empty) | TURN shared secret (matches coturn static-auth-secret) |
| `TURN_CREDENTIAL_TTL` | 86400 | TURN credential TTL in seconds |
| `STUN_URLS` | (empty) | STUN server URLs (comma-separated) |
| `API_KEY` | (empty) | Client auth API key (empty=disabled) |
| `ALLOWED_ORIGINS` | (empty) | WebClient origin whitelist (comma-separated) |
| `ALIYUN_SMS_ACCESS_KEY_ID` | (empty) | Aliyun SMS AccessKey ID |
| `ALIYUN_SMS_ACCESS_KEY_SECRET` | (empty) | Aliyun SMS AccessKey Secret |
| `ALIYUN_SMS_SIGN_NAME` | (empty) | Aliyun SMS signature name |
| `ALIYUN_SMS_TEMPLATE_CODE` | (empty) | Aliyun SMS template code |

> **Note:** Optional `.env` values are only used to seed the database on first deployment. After that, all changes should be made through the admin panel.
> 
> **SMS:** Aliyun SMS enables phone number verification for login/register. SMS is auto-enabled when all four fields are set; leave any empty to disable.

### Persistent Data Directory

All Docker deployment scripts store user data in a host directory by default. Data is not stored inside the Docker image. The default directory is `~/.quickdesk/signaling/<instance-name>`, which is usually writable on Linux, macOS and Windows Git Bash/WSL. If `--name` is omitted, the instance name is derived from the port, for example port `8000` becomes `port-8000`:

| Data | Host directory | Container directory |
|------|----------------|---------------------|
| PostgreSQL | `~/.quickdesk/signaling/<instance-name>/postgres` | `/data/postgres` |
| Redis | `~/.quickdesk/signaling/<instance-name>/redis` | `/data/redis` |
| Signaling application logs | `~/.quickdesk/signaling/<instance-name>/logs` | `/data/logs` |

Rebuilding the image, restarting the container, or running `docker compose up -d` does not clear this directory. Data only disappears if the instance data directory is manually deleted, or if you switch to a different `DATA_DIR` and mount a new empty directory.

### Production Logs and Troubleshooting

The signaling service writes to both container stdout and `$DATA_DIR/logs/signaling.log`. The file rotates at 50MB by default and retains five rotated files (`signaling.log.1` through `.5`). Configure this in `.env`:

```bash
LOG_DIR=/data/logs          # Keep this for Docker; use an absolute path on bare metal
LOG_MAX_SIZE_MB=50
LOG_MAX_BACKUPS=5
```

Logs never include access/refresh tokens, device secrets, or plaintext access codes. Business diagnostics use `component=... event=... key=value`. Important entries include `component=event_bus event=session_revoked`, `[auth] refresh family revoked`, `component=auth event=session_created`, `component=device event=ownership_transferred`, `[realtime/signal]`, `[presence]`, and authorization/slow-request events.

```bash
tail -F "$DATA_DIR/logs/signaling.log"
grep 'event=session_revoked' "$DATA_DIR/logs/signaling.log"
grep 'request_id="..."' "$DATA_DIR/logs/signaling.log"
```

To customize the data directory, set `DATA_DIR` before running the deploy script. `DATA_DIR` should point to one instance's dedicated directory; do not share one directory across multiple instances. For Linux production, explicitly placing data under `/data/quickdesk/...` or a dedicated data disk is recommended for backups and operations:

```bash
DATA_DIR=/data/quickdesk/test ./deploy-build.sh --name test --port 8000
DATA_DIR=/data/quickdesk/prod ./deploy-pull.sh --name prod --port 9000
DATA_DIR=/data/quickdesk/offline ./deploy-offline.sh quickdesk-signaling-image.tar.gz --name offline --port 10000
```

### Multiple Instances on One Host

You can deploy multiple independent signaling servers on the same physical host. Use a different port for each instance. If `--name` is omitted, the script uses `port-PORT` as the instance name and automatically isolates the container name, Compose project name and data directory:

```bash
# Instance 1: default data directory ~/.quickdesk/signaling/port-8000
./deploy-build.sh --port 8000

# Instance 2: default data directory ~/.quickdesk/signaling/port-9000
./deploy-build.sh --port 9000
```

You can also specify explicit instance names:

```bash
./deploy-build.sh --name test --port 8000
./deploy-build.sh --name prod --port 9000
```

Data directories:

```text
/data/quickdesk/test/postgres
/data/quickdesk/test/redis
/data/quickdesk/prod/postgres
/data/quickdesk/prod/redis
```

To stop a Compose instance, use the `To stop` command printed by the deploy script to avoid stopping another instance by mistake.

### Option 1: Pull Pre-built Image (Recommended)

No local compilation needed — pull the pre-built image from GitHub Container Registry:

```bash
chmod +x deploy-pull.sh

# Deploy latest version
./deploy-pull.sh

# Deploy a specific version
./deploy-pull.sh v1.0.0

# Custom port
./deploy-pull.sh --port 9000

# Multiple instances (default independent data directory ~/.quickdesk/signaling/test)
./deploy-pull.sh --name test --port 9000
```

### Option 2: Build from Source

Build the Docker image locally. Use this when you need to customize the source code or can't pull images from the registry:

```bash
chmod +x deploy-build.sh
./deploy-build.sh

# Custom port
./deploy-build.sh --port 9000

# Multiple instances (default independent data directory ~/.quickdesk/signaling/test)
./deploy-build.sh --name test --port 9000
```

### Option 3: Offline Deploy

For servers without internet access. Download the offline image from GitHub Actions Artifacts or Releases, then load and deploy:

```bash
# 1. Download the offline image (.tar.gz) on a machine with internet
#    - GitHub Actions → SignalingServer Docker → Artifacts
#    - GitHub Releases (for tagged versions)

# 2. Transfer to the target server and deploy
chmod +x deploy-offline.sh
./deploy-offline.sh quickdesk-signaling-image.tar.gz

# Multiple instances (default independent data directory ~/.quickdesk/signaling/test)
./deploy-offline.sh quickdesk-signaling-image.tar.gz --name test --port 9000
```

### Legacy Deploy Script

The original one-click deploy script is still available, with Nginx reverse proxy and SSL support:

```bash
chmod +x deploy.sh
./deploy.sh

# With domain and Nginx
./deploy.sh --port 8000 --domain your-domain.com

# Multiple instances (default independent data directory ~/.quickdesk/signaling/test)
./deploy.sh --name test --port 9000
```

### Post-Deployment Setup

After deployment, log in to the admin panel to complete the following:

1. **Change admin password**: Admin Panel → Admin Users → Edit, change username/password
2. **Configure ICE servers**: Admin Panel → Settings → ICE / TURN / STUN, add your TURN/STUN servers
3. **Configure security**:
   - **API Key**: Admin Panel → Settings → API Key. When set, only clients carrying this key can connect to the signaling server, preventing unauthorized client access. Native clients (QuickDesk desktop) can configure the API Key in **Settings → Network → API Key** at runtime — no recompilation needed
   - **Allowed Origins**: Admin Panel → Settings → Allowed Origins. **Required when deploying the WebClient on a separate domain** (e.g. `https://web.quickdesk.cc`). Two reasons:
     1. CORS — browsers block cross-origin requests by default
     2. The `POST /v1/devices/:id/access-code:verify` endpoint authenticates browser callers via `Origin` whitelist (browsers can't attach `X-API-Key` to fetch). **If you forget this, the WebClient "Connect" button silently 403s.**
     
     Separate multiple domains with commas. Example: `https://web.quickdesk.cc,https://app.quickdesk.cc`
4. **Configure WebClient URL** (optional): Admin Panel → Preset → WebClient URL. Enter the WebClient deployment address (e.g. `https://web.quickdesk.cc`). The native client will display this link in its UI so users can quickly access the WebClient
5. **Configure SMS** (optional): Admin Panel → Settings → Aliyun SMS, fill in AccessKey, signature and template to enable phone number verification

These settings take effect **immediately** without restarting the server.

### Docker Compose Management

After deploying with `deploy-pull.sh`, `deploy-build.sh` or `deploy-offline.sh`, use standard docker compose commands:

```bash
# Check status
docker compose ps

# View logs
docker compose logs -f

# View recent logs (recommended to avoid following high-volume logs)
docker compose logs --tail 200

# Stop services
docker compose down

# Restart services
docker compose restart
```

The generated `docker-compose.yml` configures Docker `json-file` log rotation for the all-in-one container (50MB per file, 3 files retained). This prevents high-volume server logs from growing indefinitely and causing disk/IO pressure. Adjust `logging.options.max-size` / `max-file` if needed, then run `docker compose up -d`.

---

## Manual Deployment (Step by Step)

### System Requirements

- CentOS 7/8/Stream or Rocky Linux 8+
- 2GB+ RAM
- 10GB+ disk space
- Public IP and domain name (optional)

## 1. Install Docker

```bash
# Install Docker
sudo yum install -y yum-utils
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
sudo yum install -y docker-ce docker-ce-cli containerd.io
sudo systemctl start docker
sudo systemctl enable docker

# Verify installation
docker --version
```

## 2. Deploy Databases (Docker)

```bash
# Create data directories
sudo mkdir -p /data/quickdesk/{postgres,redis}

# Start PostgreSQL
docker run -d --name quickdesk-postgres \
  --restart=always \
  -p 5432:5432 \
  -e POSTGRES_USER=quickdesk \
  -e POSTGRES_PASSWORD=quickdesk123 \
  -e POSTGRES_DB=quickdesk \
  -v /data/quickdesk/postgres:/var/lib/postgresql/data \
  postgres:15

# Start Redis
# IMPORTANT: keyspace notifications must be enabled so that
# host heartbeat-TTL expiries surface as `device.online.changed`
# realtime events. Pass `--notify-keyspace-events Ex` (uppercase E,
# lowercase x) to redis-server, or set `notify-keyspace-events Ex`
# in redis.conf for a config-file deployment.
docker run -d --name quickdesk-redis \
  --restart=always \
  -p 6379:6379 \
  -v /data/quickdesk/redis:/data \
  redis:7 redis-server --appendonly yes --notify-keyspace-events Ex

# Verify running status
docker ps
```

> **Why this Redis flag matters** — the v1 design tracks device online/offline
> state from two signals (heartbeat + signaling-WS connection) stored as
> Redis keys with TTLs. When a host crashes, only the TTL expiry tells the
> server it has gone offline. Without `notify-keyspace-events Ex`, the server
> never sees the expiry and `online` stays stuck at `true` until the next
> client connect attempt. The default Redis image (`redis:7`) ships with
> notifications **disabled**.
>
> See `QuickDesk/docs/dev/信令服务器API重构方案.md` §2.4 / §2.17 for the
> full state-machine rationale.

## 3. Install Go

```bash
# Download Go 1.24
cd /tmp
wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz

# Extract and install
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz

# Configure environment variables
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify installation
go version
```

## 4. Install Node.js (for frontend build)

```bash
# Install Node.js 20 LTS
curl -fsSL https://rpm.nodesource.com/setup_20.x | sudo bash -
sudo yum install -y nodejs

# Verify installation
node --version
npm --version
```

## 5. Build the Signaling Server

> **Note:** Reference SQL can be found in `SignalingServer/migrations/001_init.sql`. GORM AutoMigrate is disabled by default; for first initialization or schema upgrades, temporarily set `RUN_AUTO_MIGRATE=true` in `.env` and start the server once. Set it back to `false` afterwards to avoid scanning `information_schema` on every production startup.

```bash
# Clone the code to your server
cd /opt
git clone git@github.com:barry-ran/QuickDesk.git
cd QuickDesk/SignalingServer

# Build frontend (admin dashboard + user portal, Vue 3 + Element Plus)
cd web
npm install
npm run build
cd ..

# Download Go dependencies
go mod tidy

# Build (frontend assets are embedded via go:embed)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -a -ldflags="-s -w -extldflags '-static'" \
  -o quickdesk_signaling ./cmd/signaling

# Create runtime directory and deploy
sudo mkdir -p /opt/quickdesk-signaling
sudo cp quickdesk_signaling /opt/quickdesk-signaling/
```

## 6. Configure and Run

```bash
# Copy configuration template to runtime directory
sudo cp .env /opt/quickdesk-signaling/.env

# Edit configuration as needed (database, TURN, API key, etc.)
sudo vim /opt/quickdesk-signaling/.env

# Create systemd service
sudo cat > /etc/systemd/system/quickdesk-signaling.service << 'EOF'
[Unit]
Description=QuickDesk Signaling Server
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/quickdesk-signaling
ExecStart=/opt/quickdesk-signaling/quickdesk_signaling
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Start the service
sudo systemctl daemon-reload
sudo systemctl start quickdesk-signaling
sudo systemctl enable quickdesk-signaling

# Check service status
sudo systemctl status quickdesk-signaling
```

The Go server reads `.env` from its working directory by default. You can also specify a custom config file:

```bash
# Specify config file path
/opt/quickdesk-signaling/quickdesk_signaling --env /etc/quickdesk/production.env
```

You can also modify `ExecStart` in the systemd service to add the `--env` flag.

To update configuration later, edit the file and restart:

```bash
sudo vim /opt/quickdesk-signaling/.env
sudo systemctl restart quickdesk-signaling
```

## 7. Configure Firewall

```bash
# Open ports
sudo firewall-cmd --permanent --add-port=8000/tcp
sudo firewall-cmd --permanent --add-port=80/tcp
sudo firewall-cmd --permanent --add-port=443/tcp
sudo firewall-cmd --reload

# Or disable firewall (not recommended for production)
# sudo systemctl stop firewalld
# sudo systemctl disable firewalld
```

## 8. Domain Access (Nginx Reverse Proxy)

```bash
# Install Nginx
sudo yum install -y nginx
sudo systemctl start nginx
sudo systemctl enable nginx

# Configure reverse proxy
sudo cat > /etc/nginx/conf.d/quickdesk.conf << 'EOF'
# Dynamic Connection header for WebSocket support
map $http_upgrade $connection_upgrade {
    default upgrade;
    ''      close;
}

upstream signaling {
    server 127.0.0.1:8000;
}

server {
    listen 80;
    server_name your-domain.com;  # Replace with your domain

    client_max_body_size 100M;

    # v1 realtime WebSocket endpoints (long-lived).
    #   /v1/realtime/events  — user device-state stream (Qt + WebClient)
    #   /v1/realtime/signal  — WebRTC SDP/ICE relay (host + client)
    # Server sends Ping every 60s; set proxy_read_timeout > Ping interval.
    # 300s gives ample margin for transient network slowdowns.
    location /v1/realtime/ {
        proxy_pass http://signaling;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
        proxy_buffering off;
    }

    # HTTP API (/v1/*) and static admin SPA (/admin/, /).
    location / {
        proxy_pass http://signaling;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 30s;

        # Pre-flight CORS — the WebClient lives on a different
        # subdomain in most deployments. Add the WebClient origin to
        # the admin-panel `allowed_origins` setting; the signaling
        # server emits the appropriate Access-Control-* headers.
    }
}
EOF

# Test configuration
sudo nginx -t

# Restart Nginx
sudo systemctl restart nginx
```

## 9. HTTPS Configuration (Optional)

```bash
# Install Certbot
sudo yum install -y epel-release
sudo yum install -y certbot python3-certbot-nginx

# Request certificate (auto-configures Nginx)
sudo certbot --nginx -d your-domain.com

# Test auto-renewal
sudo certbot renew --dry-run

# Certbot automatically adds a cron job, no manual configuration needed
```

After configuration, the Nginx config will be automatically updated to:

```nginx
server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;
    include /etc/letsencrypt/options-ssl-nginx.conf;
    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem;

    location /v1/realtime/ {
        proxy_pass http://127.0.0.1:8000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
        proxy_buffering off;
    }

    location / {
        proxy_pass http://127.0.0.1:8000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

server {
    listen 80;
    server_name your-domain.com;
    return 301 https://$server_name$request_uri;
}
```

## 10. Verify Deployment

```bash
# Check service status
sudo systemctl status quickdesk-signaling
sudo systemctl status nginx

# Check port listening
sudo netstat -tlnp | grep -E '8000|80|443'

# Test API (local)
curl http://localhost:8000/health

# Test API (domain)
curl http://your-domain.com/health
curl https://your-domain.com/health  # HTTPS
```

### Health endpoint reference

`GET /health` is the canonical liveness + readiness probe. It is **public**
(no auth) by design so Kubernetes / load-balancer health checks can hit it
without provisioning credentials. A healthy server returns:

```json
{
  "status":  "ok",
  "version": "v2.10.0",
  "components": {
    "postgres": "ok",
    "redis":    "ok"
  }
}
```

Behavior:

- HTTP `200` when every component reports `ok` — use this as your readiness
  probe.
- HTTP `503` with `status: "degraded"` if any component is non-ok (DB
  unreachable, Redis missing, …). The `components` map tells you which one.
- The `version` field comes from the `Version` ldflag at build time
  (`-X main.Version=$(git describe --tags --always)`). When unset it defaults
  to `"dev"`.

Recommended Kubernetes probes:

```yaml
livenessProbe:
  httpGet: { path: /health, port: 8000 }
  initialDelaySeconds: 5
  periodSeconds: 10
  failureThreshold: 6     # tolerate ~60s of degraded state before kill

readinessProbe:
  httpGet: { path: /health, port: 8000 }
  periodSeconds: 5
  failureThreshold: 2     # take out of LB rotation quickly
```

When you put nginx in front, `/health` is reachable at the public domain and
also serves as a quick smoke test from the outside:

```bash
# Expect 200 + JSON
curl -i https://your-domain.com/health
```

If your monitoring stack only consumes status codes, use:

```bash
curl -fsS -o /dev/null -w '%{http_code}\n' https://your-domain.com/health
# → 200 healthy / 503 degraded / connection error = service down
```

## 11. View Logs

```bash
# Signaling server logs
sudo journalctl -u quickdesk-signaling -f

# Nginx logs
sudo tail -f /var/log/nginx/access.log
sudo tail -f /var/log/nginx/error.log

# Docker container logs
docker logs -f quickdesk-postgres
docker logs -f quickdesk-redis
```

## 12. Common Operations

```bash
# Restart services
sudo systemctl restart quickdesk-signaling
sudo systemctl restart nginx

# Stop service
sudo systemctl stop quickdesk-signaling

# Access database
docker exec -it quickdesk-postgres psql -U quickdesk -d quickdesk

# Query device count
docker exec -it quickdesk-postgres psql -U quickdesk -d quickdesk -c "SELECT COUNT(*) FROM devices;"

# Backup database
docker exec quickdesk-postgres pg_dump -U quickdesk quickdesk > /backup/quickdesk_$(date +%Y%m%d).sql

# Restore database
cat backup.sql | docker exec -i quickdesk-postgres psql -U quickdesk -d quickdesk

# View preset configuration (you need an admin access_token)
curl http://localhost:8000/v1/admin/preset \
  -H "Authorization: Bearer $ADMIN_ACCESS_TOKEN"

# Update preset configuration (can also be done via admin dashboard at /admin/)
curl -X PUT http://localhost:8000/v1/admin/preset \
  -H "Authorization: Bearer $ADMIN_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d @test_preset.json
```

## Troubleshooting

### Service Fails to Start

```bash
# View detailed logs
sudo journalctl -u quickdesk-signaling -n 100 --no-pager

# Check port usage
sudo lsof -i:8000

# Check database connectivity
docker exec -it quickdesk-postgres psql -U quickdesk -d quickdesk -c "SELECT 1;"
docker exec -it quickdesk-redis redis-cli ping
```

### Nginx 502 Error

```bash
# Check if signaling service is running
sudo systemctl status quickdesk-signaling

# Check SELinux status (may block Nginx connections)
sudo setenforce 0  # Temporarily disable
sudo sed -i 's/SELINUX=enforcing/SELINUX=disabled/g' /etc/selinux/config  # Permanently disable

# Or configure SELinux to allow Nginx network connections
sudo setsebool -P httpd_can_network_connect 1
```

### Database Connection Failure

```bash
# Check Docker container status
docker ps -a

# Restart database containers
docker restart quickdesk-postgres quickdesk-redis

# Check database logs
docker logs quickdesk-postgres
```

## Security Recommendations

1. **Change default passwords**: Update PostgreSQL and Redis passwords in your `.env` file
2. **Configure firewall**: Only open necessary ports (80, 443)
3. **Enable HTTPS**: HTTPS is mandatory for production environments
4. **Regular backups**: Set up scheduled database backups
5. **Log monitoring**: Configure log collection and alerting
6. **Rate limiting**: Configure request rate limiting in Nginx
7. **Client authentication**: Configure API Key and Allowed Origins in the admin panel (Settings) to restrict client access

```nginx
# Nginx rate limiting example
http {
    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;
    
    server {
        location /api/ {
            limit_req zone=api_limit burst=20 nodelay;
        }
    }
}
```

## Production Configuration

```bash
# 1. Change database password (use a strong password)
# Stop container
docker stop quickdesk-postgres
docker rm quickdesk-postgres

# Recreate with new password
docker run -d --name quickdesk-postgres \
  --restart=always \
  -p 5432:5432 \
  -e POSTGRES_USER=quickdesk \
  -e POSTGRES_PASSWORD='your-strong-password-here' \
  -e POSTGRES_DB=quickdesk \
  -v /data/quickdesk/postgres:/var/lib/postgresql/data \
  postgres:15

# 2. Update DB_PASSWORD in your .env file
vim .env

# 3. Re-deploy
./deploy.sh
```

## Access URLs

After deployment, the following URLs are available:

- **HTTP**: `http://your-domain.com`
- **HTTPS**: `https://your-domain.com`
- **Health probe**: `https://your-domain.com/health`
- **WebSocket (realtime events)**: `wss://your-domain.com/v1/realtime/events` (first-frame auth — see user-api-docs §7.1)
- **WebSocket (signaling relay)**: `wss://your-domain.com/v1/realtime/signal` (first-frame auth — see user-api-docs §7.2)
- **API root**: `https://your-domain.com/v1/` (full route table in `SignalingServer/docs/user-api-docs.md`)
- **Admin Dashboard**: `https://your-domain.com/admin/` (devices, users, admin accounts, system settings)
- **WebClient**: Deployed independently, communicates with the signaling server via API

## Performance Tuning

```bash
# 1. Adjust Nginx worker count
# Edit /etc/nginx/nginx.conf
worker_processes auto;
worker_connections 4096;

# 2. Configure PostgreSQL connection pool
# Set appropriate pool size in code configuration

# 3. Redis persistence configuration
docker run -d --name quickdesk-redis \
  --restart=always \
  -p 6379:6379 \
  -v /data/quickdesk/redis:/data \
  redis:7 redis-server \
    --appendonly yes \
    --maxmemory 512mb \
    --maxmemory-policy allkeys-lru
```
