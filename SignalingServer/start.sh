#!/bin/bash
set -e

PG_DATA=/data/postgres
REDIS_DIR=/data/redis

# Ensure data directories exist with correct ownership (needed when
# the host mounts an empty volume over /data)
LOG_DIR=/opt/quickdesk/logs

mkdir -p "$PG_DATA" "$REDIS_DIR" "$LOG_DIR"
chown -R postgres:postgres "$PG_DATA" "$LOG_DIR"

# ---- PostgreSQL ----
if [ ! -f "$PG_DATA/PG_VERSION" ]; then
    echo "[quickdesk] Initializing PostgreSQL..."
    gosu postgres /usr/lib/postgresql/15/bin/initdb -D "$PG_DATA" --auth-local=trust --auth-host=md5 --encoding=UTF8 --locale=C.UTF-8
    echo "host all all 0.0.0.0/0 md5" >> "$PG_DATA/pg_hba.conf"
    echo "listen_addresses = '127.0.0.1'" >> "$PG_DATA/postgresql.conf"
fi
gosu postgres /usr/lib/postgresql/15/bin/pg_ctl -D "$PG_DATA" -l "$LOG_DIR/postgres.log" start
sleep 2

# Create user and database if needed
gosu postgres psql -tc "SELECT 1 FROM pg_roles WHERE rolname='${DB_USER:-quickdesk}'" | grep -q 1 || \
    gosu postgres psql -c "CREATE USER ${DB_USER:-quickdesk} WITH PASSWORD '${DB_PASSWORD:-quickdesk123}';"
gosu postgres psql -tc "SELECT 1 FROM pg_database WHERE datname='${DB_NAME:-quickdesk}'" | grep -q 1 || \
    gosu postgres psql -c "CREATE DATABASE ${DB_NAME:-quickdesk} OWNER ${DB_USER:-quickdesk} ENCODING 'UTF8' LC_COLLATE 'C.UTF-8' LC_CTYPE 'C.UTF-8' TEMPLATE template0;"

# ---- Redis ----
echo "[quickdesk] Starting Redis..."
redis-server --daemonize yes --dir "$REDIS_DIR" --appendonly yes --bind 127.0.0.1

# Wait for Redis to be ready
echo "[quickdesk] Waiting for Redis to be ready..."
REDIS_READY=0
for i in $(seq 1 30); do
    if redis-cli ping 2>/dev/null | grep -q PONG; then
        echo "[quickdesk] Redis is ready."
        REDIS_READY=1
        break
    fi
    sleep 1
done

# If Redis is stuck (likely corrupted AOF), reset persistence and restart
if [ "$REDIS_READY" -eq 0 ]; then
    echo "[quickdesk] WARNING: Redis not ready after 30s, likely corrupted AOF. Resetting persistence data..."
    redis-cli shutdown nosave 2>/dev/null || true
    rm -rf "$REDIS_DIR/appendonlydir" "$REDIS_DIR/dump.rdb"
    redis-server --daemonize yes --dir "$REDIS_DIR" --appendonly yes --bind 127.0.0.1
    sleep 2
    if redis-cli ping 2>/dev/null | grep -q PONG; then
        echo "[quickdesk] Redis restarted successfully with clean state."
    else
        echo "[quickdesk] ERROR: Redis still not ready after reset. Exiting."
        exit 1
    fi
fi

# ---- Signaling Server ----
echo "[quickdesk] Starting signaling server..."
cd /opt/quickdesk
exec ./signaling
