# QuickDesk Signaling Server

QuickDesk 信令服务器，使用 Go + Gin + GORM 开发，提供设备 provisioning、用户认证、设备绑定、WebRTC 信令转发与实时事件推送。

> 完整 API 契约见 [`docs/user-api-docs.md`](docs/user-api-docs.md)。
> 设计原理与决策记录见 [`../docs/dev/信令服务器API重构方案.md`](../docs/dev/信令服务器API重构方案.md)。

## 功能

- 设备 provisioning（首次启动自动分配 9 位 device_id + device_secret）
- 用户注册/登录（密码 + SMS）；access/refresh token + family rotation
- 我的设备 / 收藏 / 连接历史 CRUD
- WebRTC 信令中继（host ↔ client，jingle 信封）
- 实时事件推送（snapshot + 增量；Redis stream resume）
- 管理员后台（CRUD + 2FA + 审计 + webhook）
- Redis 双信号 presence（hb + wsconn 派生 online；keyspace notifications）
- PostgreSQL 持久化

## 依赖

- Go 1.21+
- PostgreSQL 15+
- Redis 7+（**必须**启用 `notify-keyspace-events Ex`）

## 快速开始

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 启动数据库

使用 Docker：

```bash
# PostgreSQL
docker run -d -p 5432:5432 \
  -e POSTGRES_USER=quickdesk \
  -e POSTGRES_PASSWORD=quickdesk123 \
  -e POSTGRES_DB=quickdesk \
  postgres:15

# Redis（必须开启 keyspace notifications，host 心跳 TTL 过期靠它派发 device.online.changed 事件）
docker run -d -p 6379:6379 redis:7 redis-server --appendonly yes --notify-keyspace-events Ex
```

### 3. 配置环境变量

复制 `.env.example` 到 `.env` 并修改配置：

```bash
cp .env.example .env
```

### 4. 运行服务器

```bash
go run cmd/signaling/main.go
```

服务器将在 `http://localhost:8000` 启动。

## API 端点

服务器暴露的全部路由前缀都是 `/v1/*`（旧 `/api/v1/*`、`/signal/*`、`/host/*`、`/client/*` 在 v1 重构后已下线）。

主要分组：

| 分组 | 路径前缀 | 说明 |
|---|---|---|
| 公开 | `/health`、`/v1/preset`、`/v1/settings/public`、`/v1/features`、`/v1/verification-codes` | 无 user token，但需 `X-API-Key`（若服务端配置） |
| 认证 | `/v1/auth/*` | 注册、登录、SMS 登录、token 刷新、密码重置 |
| 当前用户 | `/v1/me/*` | profile、设备、连接、收藏、session 管理（Bearer access_token） |
| 设备侧 | `/v1/devices*`、`/v1/ice-config` | provision / heartbeat / signal-tokens / access-code:verify |
| 管理后台 | `/v1/admin/*` | 完整 CRUD + 2FA + 审计 + webhook |
| 实时 WebSocket | `/v1/realtime/events`、`/v1/realtime/signal` | 首帧 auth 握手；浏览器无 X-API-Key 走 Origin 白名单 |

完整路由 / 请求体 / 响应体 / 错误码契约请见 [`docs/user-api-docs.md`](docs/user-api-docs.md)。

## 项目结构

```
server/
├── cmd/
│   └── signaling/
│       └── main.go              # 程序入口
├── internal/
│   ├── config/                  # 配置管理
│   ├── models/                  # 数据模型
│   ├── repository/              # 数据访问层
│   ├── service/                 # 业务逻辑层
│   ├── handler/                 # HTTP 和 WebSocket 处理器
│   └── middleware/              # 中间件
├── migrations/                  # 数据库迁移文件
├── go.mod
├── go.sum
├── .env.example
└── README.md
```

## 开发

```bash
# 运行测试
go test ./...

# 构建
go build -o signaling cmd/signaling/main.go

# 运行
./signaling
```

## License

Copyright 2026 QuickDesk
