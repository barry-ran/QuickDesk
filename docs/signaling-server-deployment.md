# QuickDesk Signaling Server Deployment Guide

## Quick Deploy (Recommended)

Use the all-in-one Docker image (PostgreSQL + Redis + signaling server):

```bash
git clone git@github.com:barry-ran/QuickDesk.git
cd QuickDesk/SignalingServer

# Copy and edit your configuration
cp .env .env.production
vim .env.production

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
| `DB_NAME` | quickdesk | Database name |
| `REDIS_HOST` | localhost | Redis host |
| `REDIS_PORT` | 6379 | Redis port |
| `REDIS_PASSWORD` | (empty) | Redis password |
| `ADMIN_USER` | admin | Initial admin username (first deploy only) |
| `ADMIN_PASSWORD` | admin | Initial admin password (first deploy only) |

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

---

# One-click deploy with your config file
chmod +x deploy.sh
./deploy.sh --env .env.production

# With domain and Nginx
./deploy.sh --env .env.production --port 8000 --domain your-domain.com
```

Or manually:

```bash
# Build image
docker build -t quickdesk-signaling .

# Run with --env-file
docker run -d \
    --name quickdesk-signaling \
    --restart=always \
    -p 8000:8000 \
    -v /data/quickdesk:/data \
    --env-file .env.production \
    quickdesk-signaling
```

### Post-Deployment Setup

After deployment, log in to the admin panel to complete the following:

1. **Change admin password**: Admin Panel → Admin Users → Edit, change username/password
2. **Configure ICE servers**: Admin Panel → Settings → ICE / TURN / STUN, add your TURN/STUN servers
3. **Configure security**: Set API Key and Allowed Origins as needed
4. **Configure SMS** (optional): Admin Panel → Settings → Aliyun SMS, fill in AccessKey, signature and template to enable phone number verification

These settings take effect **immediately** without restarting the server.

To update infrastructure config (port, database, etc.), edit `.env` and re-deploy:

```bash
vim .env.production
./deploy.sh --env .env.production
```

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
docker run -d --name quickdesk-redis \
  --restart=always \
  -p 6379:6379 \
  -v /data/quickdesk/redis:/data \
  redis:7 redis-server --appendonly yes

# Verify running status
docker ps
```

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

> **Note:** Database tables are automatically created/updated by GORM AutoMigrate when the signaling server starts. No manual SQL execution is required.
> Reference SQL can be found in `SignalingServer/migrations/001_init.sql`.

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

The Go server automatically reads the `.env` file from its working directory.

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

    # WebSocket endpoints (long-lived connections)
    # The signaling server sends WebSocket Ping frames every 60s.
    # Set proxy_read_timeout > ping interval; 300s (5 min) gives ample margin.
    location /signal/ {
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

    # HTTP API and static files
    location / {
        proxy_pass http://signaling;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 30s;
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

    location /signal/ {
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

# View preset configuration
curl http://localhost:8000/api/v1/admin/preset

# Update preset configuration (can also be done via admin dashboard at /admin/)
curl -X PUT http://localhost:8000/api/v1/admin/preset \
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
vim .env.production

# 3. Re-deploy
./deploy.sh --env .env.production
```

## Access URLs

After deployment, the following URLs are available:

- **HTTP**: `http://your-domain.com`
- **HTTPS**: `https://your-domain.com`
- **WebSocket**: `wss://your-domain.com/signal/:device_id?access_code=xxx`
- **API**: `https://your-domain.com/api/v1/devices/register`
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
