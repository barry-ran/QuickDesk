#!/bin/bash
# QuickDesk Signaling Server — build image locally and deploy
# Usage: ./deploy.sh [--port PORT] [--name NAME] [--domain DOMAIN]
#
# This script builds the Docker image directly (docker build), then runs
# the container. For docker-compose based deployment, use deploy-build.sh.
#
# Other deployment methods:
#   ./deploy-pull.sh     — Pull pre-built image from registry (recommended)
#   ./deploy-build.sh    — Build from source using docker-compose
#   ./deploy-offline.sh  — Load from offline image archive

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

PORT=""
DOMAIN=""
INSTANCE_NAME=""
CONTAINER_NAME="${CONTAINER_NAME:-}"
IMAGE_NAME="quickdesk-signaling"
DATA_DIR="${DATA_DIR:-}"

sanitize_name() {
    echo "$1" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9_.-]+/-/g; s/^-+//; s/-+$//'
}

default_data_dir() {
    echo "${HOME:-$SCRIPT_DIR}/.quickdesk/signaling/$INSTANCE_NAME"
}

while [[ $# -gt 0 ]]; do
    case $1 in
        --port)   PORT="$2"; shift 2;;
        --name)   INSTANCE_NAME="$2"; shift 2;;
        --domain) DOMAIN="$2"; shift 2;;
        -h|--help)
            echo "Usage: $0 [--port PORT] [--name NAME] [--domain DOMAIN]"
            echo ""
            echo "  --port    Host port (default: SERVER_PORT from .env, or 8000)"
            echo "  --name    Instance name (default: port-PORT; data: ~/.quickdesk/signaling/NAME)"
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

if [ -z "$INSTANCE_NAME" ]; then
    INSTANCE_NAME="port-$PORT"
fi
INSTANCE_NAME=$(sanitize_name "$INSTANCE_NAME")
if [ -z "$INSTANCE_NAME" ]; then
    echo "ERROR: Invalid instance name."
    exit 1
fi

DATA_DIR="${DATA_DIR:-$(default_data_dir)}"
CONTAINER_NAME="${CONTAINER_NAME:-quickdesk-signaling-$INSTANCE_NAME}"

echo "=========================================="
echo " QuickDesk Signaling Server (Direct Build)"
echo "=========================================="
echo "Port:     $PORT"
echo "Name:     $INSTANCE_NAME"
echo "Domain:   ${DOMAIN:-<none>}"
echo "Data:     $DATA_DIR"
echo "Container:$CONTAINER_NAME"
echo ""

# ---- 1. Build ----
echo "[1/4] Building Docker image..."
docker build -t "$IMAGE_NAME" "$SCRIPT_DIR"

# ---- 2. Stop old container ----
if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo "[2/4] Stopping old container..."
    docker stop "$CONTAINER_NAME" 2>/dev/null || true
    docker rm "$CONTAINER_NAME" 2>/dev/null || true
else
    echo "[2/4] No old container found."
fi

# ---- 3. Start ----
echo "[3/4] Starting container..."
mkdir -p "$DATA_DIR"

docker run -d \
    --name "$CONTAINER_NAME" \
    --restart=always \
    -p "$PORT:8000" \
    -v "$DATA_DIR:/data" \
    --env-file "$SCRIPT_DIR/.env" \
    "$IMAGE_NAME"

echo "Waiting for server to become healthy..."

MAX_WAIT=120
WAITED=0
HEALTHY=false

while [ $WAITED -lt $MAX_WAIT ]; do
    STATUS=$(docker inspect --format='{{.State.Status}}' "$CONTAINER_NAME" 2>/dev/null || echo "missing")
    if [ "$STATUS" = "exited" ] || [ "$STATUS" = "missing" ]; then
        echo ""
        echo "ERROR: Container exited unexpectedly!"
        docker logs --tail 30 "$CONTAINER_NAME"
        exit 1
    fi

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
    echo "Container status: $(docker inspect --format='{{.State.Status}}' "$CONTAINER_NAME" 2>/dev/null)"
    echo "Recent logs:"
    docker logs --tail 50 "$CONTAINER_NAME"
    exit 1
fi

# ---- 4. Nginx (optional) ----
if [ -n "$DOMAIN" ]; then
    echo "[4/4] Configuring Nginx..."
    bash "$SCRIPT_DIR/setup-nginx.sh" "$DOMAIN" "$PORT"
else
    echo "[4/4] Skipping Nginx (no --domain specified)."
fi

echo ""
echo "=========================================="
echo " Deployment complete!"
echo "=========================================="
echo ""
echo "  Health:  curl http://localhost:$PORT/health"
echo "  Admin:   http://localhost:$PORT/admin/"
echo "  Logs:    docker logs -f $CONTAINER_NAME"
echo "  Data:    $DATA_DIR"
if [ -n "$DOMAIN" ]; then
    echo "  URL:     http://$DOMAIN"
fi
echo ""
echo "  Other deployment methods:"
echo "    ./deploy-pull.sh     — Pull pre-built image (recommended)"
echo "    ./deploy-build.sh    — Build from source using docker-compose"
echo "    ./deploy-offline.sh  — Load from offline image archive"
