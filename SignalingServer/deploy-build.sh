#!/bin/bash
# QuickDesk Signaling Server — build from source and deploy
# Usage: ./deploy-build.sh [--port PORT] [--domain DOMAIN]
#
# Builds the Docker image locally from source using docker-compose,
# then starts the container. Use this when you need to customize the
# source code or when pre-built images are not available.

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

PORT=""
DOMAIN=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --port)   PORT="$2"; shift 2;;
        --domain) DOMAIN="$2"; shift 2;;
        -h|--help)
            echo "Usage: $0 [--port PORT] [--domain DOMAIN]"
            echo ""
            echo "  --port    Host port (default: SERVER_PORT from .env, or 8000)"
            echo "  --domain  Configure Nginx reverse proxy + optional SSL"
            exit 0;;
        *) echo "Unknown option: $1"; exit 1;;
    esac
done

# ---- .env check ----
if [ ! -f ".env" ]; then
    if [ -f ".env.example" ]; then
        echo "No .env file found."
        echo ""
        echo "Creating .env from .env.example — please review and edit it:"
        cp .env.example .env
        echo "  vim .env"
        echo ""
        echo "Then re-run: $0"
        exit 1
    else
        echo "ERROR: Neither .env nor .env.example found."
        exit 1
    fi
fi

if [ -z "$PORT" ]; then
    PORT=$(grep -E '^SERVER_PORT=' .env 2>/dev/null | cut -d= -f2 | tr -d '[:space:]')
    PORT="${PORT:-8000}"
fi

echo "=========================================="
echo " QuickDesk Signaling Server (Build Deploy)"
echo "=========================================="
echo "Port:     $PORT"
echo "Domain:   ${DOMAIN:-<none>}"
echo ""

export SERVER_PORT="$PORT"

# ---- 1. Build ----
echo "[1/3] Building Docker image from source..."
docker compose -f docker-compose.yml -f docker-compose.build.yml build

# ---- 2. Start ----
echo "[2/3] Starting services..."
docker compose -f docker-compose.yml -f docker-compose.build.yml up -d

# ---- 3. Health check ----
echo "[3/3] Waiting for server to become healthy..."
MAX_WAIT=120
WAITED=0
HEALTHY=false

while [ $WAITED -lt $MAX_WAIT ]; do
    if curl -sf "http://127.0.0.1:$PORT/health" > /dev/null 2>&1; then
        HEALTHY=true
        break
    fi
    sleep 2
    WAITED=$((WAITED + 2))
    printf "."
done
echo ""

if [ "$HEALTHY" = true ]; then
    echo "Server is healthy and ready."
else
    echo "ERROR: Server did not become healthy within ${MAX_WAIT}s!"
    echo ""
    echo "Container logs:"
    docker compose -f docker-compose.yml -f docker-compose.build.yml logs --tail 50
    exit 1
fi

# ---- Nginx (optional) ----
if [ -n "$DOMAIN" ]; then
    echo ""
    bash "$SCRIPT_DIR/setup-nginx.sh" "$DOMAIN" "$PORT"
fi

echo ""
echo "=========================================="
echo " Deployment complete!"
echo "=========================================="
echo ""
echo "  Health:  curl http://localhost:$PORT/health"
echo "  Admin:   http://localhost:$PORT/admin/"
echo "  Logs:    docker compose -f docker-compose.yml -f docker-compose.build.yml logs -f"
if [ -n "$DOMAIN" ]; then
    echo "  URL:     http://$DOMAIN"
fi
echo ""
echo "  To rebuild: ./deploy-build.sh"
echo "  To stop:    docker compose down"
