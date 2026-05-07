# QuickDesk Signaling Server — all-in-one Docker image
# Includes: PostgreSQL 15 + Redis 7 + signaling server binary
#
# Build:   docker build -t quickdesk-signaling .
# Run:     docker run -d -p 8000:8000 --env-file .env quickdesk-signaling
# Custom:  docker run -d -p 8000:8000 --env-file /path/to/your.env quickdesk-signaling

# ---- Stage 1: Build frontend ----
FROM node:20-alpine AS frontend
WORKDIR /build/web
COPY web/package*.json ./
RUN npm install --no-audit
COPY web/ ./
RUN npm run build

# ---- Stage 2: Build Go binary ----
FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /build/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -a -ldflags="-s -w" -o /signaling ./cmd/signaling

# ---- Stage 3: Runtime (all-in-one) ----
FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
    postgresql-15 redis-server \
    ca-certificates procps gosu curl locales \
    && rm -rf /var/lib/apt/lists/* \
    && sed -i '/C.UTF-8/s/^# //' /etc/locale.gen && locale-gen

ENV LANG=C.UTF-8
ENV LC_ALL=C.UTF-8

COPY --from=builder /signaling /opt/quickdesk/signaling

RUN mkdir -p /data/postgres /data/redis /opt/quickdesk/logs \
    && chown -R postgres:postgres /data/postgres

COPY start.sh /opt/quickdesk/start.sh
RUN chmod +x /opt/quickdesk/start.sh

# No hardcoded ENV for application config.
# All configuration is passed via --env-file at runtime.
# The Go server has sensible defaults via viper.SetDefault() for
# cases where no env file is provided (DB at localhost, port 8000, etc.).

EXPOSE 8000
VOLUME ["/data"]

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD curl -sf http://127.0.0.1:8000/health || exit 1

CMD ["/opt/quickdesk/start.sh"]
