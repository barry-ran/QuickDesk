#!/bin/bash
# QuickDesk Signaling Server — deploy from pre-built Docker image
# Usage: ./deploy-pull.sh [VERSION] [--port PORT] [--name NAME] [--domain DOMAIN]
#
# Examples:
#   ./deploy-pull.sh                       # Pull latest and deploy
#   ./deploy-pull.sh v1.0.0                # Deploy a specific version
#   ./deploy-pull.sh --port 9000           # Custom port
#   ./deploy-pull.sh --name test --port 9000
#   ./deploy-pull.sh --domain example.com  # With Nginx reverse proxy

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

IMAGE_BASE="ghcr.io/barry-ran/quickdesk-signaling"
VERSION="latest"
PORT=""
DOMAIN=""
INSTANCE_NAME=""
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
            echo "Usage: $0 [VERSION] [--port PORT] [--name NAME] [--domain DOMAIN]"
            echo ""
            echo "  VERSION   Docker image tag (default: latest), e.g. v1.0.0"
            echo "  --port    Host port (default: SERVER_PORT from .env, or 8000)"
            echo "  --name    Instance name (default: port-PORT; data: ~/.quickdesk/signaling/NAME)"
            echo "  --domain  Configure Nginx reverse proxy + optional SSL"
            exit 0;;
        -*)
            echo "Unknown option: $1"; exit 1;;
        *)
            VERSION="$1"; shift;;
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
        echo "Then re-run: $0 $VERSION"
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
COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-quickdesk-$INSTANCE_NAME}"
CONTAINER_NAME="${CONTAINER_NAME:-quickdesk-signaling-$INSTANCE_NAME}"

echo "=========================================="
echo " QuickDesk Signaling Server (Pull Deploy)"
echo "=========================================="
echo "Image:    ${IMAGE_BASE}:${VERSION}"
echo "Port:     $PORT"
echo "Name:     $INSTANCE_NAME"
echo "Domain:   ${DOMAIN:-<none>}"
echo "Data:     $DATA_DIR"
echo "Container:$CONTAINER_NAME"
echo ""

export SERVER_PORT="$PORT"
export IMAGE_TAG="$VERSION"
export DATA_DIR
export COMPOSE_PROJECT_NAME
export CONTAINER_NAME

echo "[1/3] Pulling image ${IMAGE_BASE}:${IMAGE_TAG}..."
docker compose pull

echo "[2/3] Starting services..."
mkdir -p "$DATA_DIR"
docker compose up -d --force-recreate

# ---- 3. Health check ----
echo "[3/3] Waiting for server to become healthy..."
MAX_WAIT=90
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
    docker compose logs --tail 50
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
echo "  Logs:    COMPOSE_PROJECT_NAME=$COMPOSE_PROJECT_NAME docker compose logs -f"
echo "  Data:    $DATA_DIR"
if [ -n "$DOMAIN" ]; then
    echo "  URL:     http://$DOMAIN"
fi
echo ""
echo "  To update: ./deploy-pull.sh [new-version] --port $PORT --name $INSTANCE_NAME"
echo "  To stop:   COMPOSE_PROJECT_NAME=$COMPOSE_PROJECT_NAME docker compose down"
