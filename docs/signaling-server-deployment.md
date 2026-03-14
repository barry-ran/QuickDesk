# QuickDesk Signaling Server Deployment Guide

## System Requirements

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
# Clone or upload the code to your server
# Assuming the code is at /opt/quickdesk/SignalingServer

cd /opt/quickdesk/SignalingServer

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

# Create runtime directory
sudo mkdir -p /opt/quickdesk/signaling
sudo cp quickdesk_signaling /opt/quickdesk/signaling/
```

## 6. Configure the Service

```bash
# Create configuration file
sudo cat > /opt/quickdesk/signaling/.env << 'EOF'
SERVER_HOST=0.0.0.0
SERVER_PORT=8000
DB_HOST=localhost
DB_PORT=5432
DB_USER=quickdesk
DB_PASSWORD=quickdesk123
DB_NAME=quickdesk
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# API Key for client authentication (optional)
# When set, only clients with the correct API Key can connect to this server.
# Leave empty to disable API Key verification (any client can connect).
# Clients pass this value via the X-API-Key header.
API_KEY=

# WebClient allowed origins (optional, comma-separated)
# Browsers automatically send the Origin header which cannot be spoofed by JS.
# When set, only WebClient pages loaded from these origins can access the server.
# Native clients (Qt/Host/Client) use API Key; WebClient uses Origin whitelist.
# Example: https://web.quickdesk.cc,https://quickdesk.example.com
ALLOWED_ORIGINS=
EOF

# Create systemd service
sudo cat > /etc/systemd/system/quickdesk-signaling.service << 'EOF'
[Unit]
Description=QuickDesk Signaling Server
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/quickdesk/signaling
ExecStart=/opt/quickdesk/signaling/signaling
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
upstream signaling {
    server 127.0.0.1:8000;
}

server {
    listen 80;
    server_name your-domain.com;  # Replace with your domain

    client_max_body_size 100M;

    location / {
        proxy_pass http://signaling;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 300s;
        proxy_connect_timeout 75s;
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

    location / {
        proxy_pass http://127.0.0.1:8000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
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
curl http://localhost:8000/api/v1/health

# Test API (domain)
curl http://your-domain.com/api/v1/health
curl https://your-domain.com/api/v1/health  # HTTPS
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

1. **Change default passwords**: Update PostgreSQL and Redis passwords
2. **Configure firewall**: Only open necessary ports (80, 443)
3. **Enable HTTPS**: HTTPS is mandatory for production environments
4. **Regular backups**: Set up scheduled database backups
5. **Log monitoring**: Configure log collection and alerting
6. **Rate limiting**: Configure request rate limiting in Nginx
7. **Client authentication**: Set `API_KEY` to restrict native client access, set `ALLOWED_ORIGINS` to restrict WebClient access

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

# 2. Update configuration file
sudo vi /opt/quickdesk/signaling/.env
# Update DB_PASSWORD

# 3. Restart service
sudo systemctl restart quickdesk-signaling
```

## Access URLs

After deployment, the following URLs are available:

- **HTTP**: `http://your-domain.com`
- **HTTPS**: `https://your-domain.com`
- **WebSocket**: `wss://your-domain.com/signal/:device_id?access_code=xxx`
- **API**: `https://your-domain.com/api/v1/devices/register`
- **Admin Dashboard**: `https://your-domain.com/admin/` (devices, users, admin accounts, system settings)
- **User Portal**: `https://your-domain.com/user-login` (user registration, login, device binding)
- **WebClient**: `https://your-domain.com/webclient/remote.html` (browser-based remote desktop)

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
