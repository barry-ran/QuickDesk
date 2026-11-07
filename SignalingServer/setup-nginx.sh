#!/bin/bash
# QuickDesk Signaling Server — Nginx reverse proxy setup
# Usage: ./setup-nginx.sh <DOMAIN> <BACKEND_PORT> [--cert-file FILE --key-file FILE]
#
# The public endpoint uses HTTPS/WSS on port 443.

set -e

DOMAIN="$1"
PORT="$2"
shift 2

CERT_FILE=""
KEY_FILE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --cert-file) CERT_FILE="$2"; shift 2;;
        --key-file) KEY_FILE="$2"; shift 2;;
        -h|--help)
            echo "Usage: $0 <DOMAIN> <BACKEND_PORT> [--cert-file FILE --key-file FILE]"
            exit 0;;
        *) echo "Unknown option: $1"; exit 1;;
    esac
done

if [ -z "$DOMAIN" ] || [ -z "$PORT" ]; then
    echo "Usage: $0 <DOMAIN> <BACKEND_PORT> [--cert-file FILE --key-file FILE]"
    echo "  e.g. $0 example.com 8000"
    exit 1
fi

if ! [[ "$DOMAIN" =~ ^[A-Za-z0-9.-]+$ ]] || ! [[ "$PORT" =~ ^[0-9]+$ ]]; then
    echo "ERROR: Domain and ports contain unsupported characters."
    exit 1
fi

if [ -n "$CERT_FILE" ] || [ -n "$KEY_FILE" ]; then
    if [ -z "$CERT_FILE" ] || [ -z "$KEY_FILE" ]; then
        echo "ERROR: --cert-file and --key-file must be provided together."
        exit 1
    fi
else
    CERT_DIR="/etc/letsencrypt/live/$DOMAIN"
    CERT_FILE="$CERT_DIR/fullchain.pem"
    KEY_FILE="$CERT_DIR/privkey.pem"
fi

if [ ! -r "$CERT_FILE" ] || [ ! -r "$KEY_FILE" ]; then
    echo "ERROR: Certificate or private key cannot be read."
    echo "Certificate: $CERT_FILE"
    echo "Private key:  $KEY_FILE"
    echo "To use Let's Encrypt defaults, obtain a certificate first, for example:"
    echo "  sudo certbot certonly --standalone -d $DOMAIN"
    exit 1
fi

echo "Configuring HTTPS/WSS proxy for $DOMAIN → 127.0.0.1:$PORT ..."

# ---- Install Nginx if missing ----
if ! command -v nginx &>/dev/null; then
    echo "Installing Nginx..."
    if command -v dnf &>/dev/null; then
        sudo dnf install -y nginx
    elif command -v yum &>/dev/null; then
        sudo yum install -y nginx
    elif command -v apt-get &>/dev/null; then
        sudo apt-get update && sudo apt-get install -y nginx
    fi
fi

if ! command -v nginx &>/dev/null; then
    echo "ERROR: Failed to install Nginx. Please install it manually and re-run."
    exit 1
fi

# ---- Locate nginx.conf (system, BT Panel, custom) ----
NGINX_CONF=""
for candidate in \
    /etc/nginx/nginx.conf \
    /www/server/nginx/conf/nginx.conf \
    /usr/local/nginx/conf/nginx.conf \
    /opt/nginx/conf/nginx.conf; do
    if [ -f "$candidate" ]; then
        NGINX_CONF="$candidate"
        break
    fi
done

if [ -z "$NGINX_CONF" ]; then
    NGINX_CONF=$(nginx -t 2>&1 | grep -oP 'file \K\S+(?= syntax)' || true)
fi

if [ -z "$NGINX_CONF" ] || [ ! -f "$NGINX_CONF" ]; then
    NGINX_CONF="/etc/nginx/nginx.conf"
    echo "Creating minimal $NGINX_CONF ..."
    sudo mkdir -p /etc/nginx/conf.d /var/log/nginx
    sudo tee "$NGINX_CONF" > /dev/null << 'MINCONF'
user nginx;
worker_processes auto;
error_log /var/log/nginx/error.log;
pid /run/nginx.pid;

events {
    worker_connections 1024;
}

http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;
    sendfile      on;
    keepalive_timeout 65;
    include /etc/nginx/conf.d/*.conf;
}
MINCONF
fi

echo "Using nginx config: $NGINX_CONF"

# ---- Detect vhost include directory ----
NGINX_CONF_DIR=""
while IFS= read -r inc_path; do
    inc_dir="$(dirname "$inc_path")"
    if [ -d "$inc_dir" ]; then
        NGINX_CONF_DIR="$inc_dir"
        break
    fi
done < <(grep -E '^\s*include\s+' "$NGINX_CONF" \
    | grep -v 'mime' | grep -v 'lua' | grep -v 'proxy' | grep -v 'php' | grep -v 'enable-' | grep -v 'fastcgi' | grep -v '/tcp/' | grep -v '/stream/' \
    | grep -oP 'include\s+\K[^;]+' \
    | grep '\*\.conf' \
    | tail -n 5)

if [ -z "$NGINX_CONF_DIR" ]; then
    NGINX_CONF_DIR="$(dirname "$NGINX_CONF")/conf.d"
fi

echo "Using vhost directory: $NGINX_CONF_DIR"
sudo mkdir -p "$NGINX_CONF_DIR"

# ---- Write QuickDesk vhost config ----
sudo tee "$NGINX_CONF_DIR/quickdesk.conf" > /dev/null << NGINX_EOF
map \$http_upgrade \$connection_upgrade {
    default upgrade;
    ''      close;
}

upstream quickdesk_signaling {
    server 127.0.0.1:$PORT;
}

server {
    listen 80;
    server_name $DOMAIN;

    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }

    location / {
        return 301 https://\$host\$request_uri;
    }
}

server {
    listen 443 ssl;
    server_name $DOMAIN;

    ssl_certificate $CERT_FILE;
    ssl_certificate_key $KEY_FILE;
    ssl_protocols TLSv1.2 TLSv1.3;

    client_max_body_size 100M;

    # WebSocket (long-lived connections, server pings every 60s)
    location /v1/realtime/ {
        proxy_pass http://quickdesk_signaling;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection \$connection_upgrade;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;

        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
        proxy_buffering off;
    }

    # HTTP API and static files
    location / {
        proxy_pass http://quickdesk_signaling;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_read_timeout 30s;
    }
}

NGINX_EOF

# ---- Ensure nginx.conf includes the vhost directory ----
NGINX_CONF_DIR_ESCAPED=$(echo "$NGINX_CONF_DIR" | sed 's/[\/&]/\\&/g')
if ! grep -q "include.*${NGINX_CONF_DIR_ESCAPED}" "$NGINX_CONF" 2>/dev/null; then
    echo "Adding include for $NGINX_CONF_DIR to $NGINX_CONF ..."
    if grep -qP 'http\s*\{' "$NGINX_CONF"; then
        sudo sed -i '/http\s*{/a \    include '"$NGINX_CONF_DIR"'/*.conf;' "$NGINX_CONF"
    elif grep -q '^http$' "$NGINX_CONF"; then
        sudo sed -i '/^http$/,/\{/{/\{/a \    include '"$NGINX_CONF_DIR"'/*.conf;
        }' "$NGINX_CONF"
    else
        echo "WARNING: Could not auto-add include. Please add manually:"
        echo "  include $NGINX_CONF_DIR/*.conf;"
    fi
fi

# ---- Test and reload ----
echo "Testing nginx config..."
if ! sudo nginx -t; then
    echo "ERROR: Nginx config test failed!"
    exit 1
fi

if sudo systemctl is-active nginx &>/dev/null; then
    sudo systemctl reload nginx
elif pgrep -x nginx &>/dev/null; then
    sudo nginx -s reload
else
    sudo nginx
    sudo systemctl enable nginx 2>/dev/null || true
fi
echo "Nginx configured for https://$DOMAIN → 127.0.0.1:$PORT"
