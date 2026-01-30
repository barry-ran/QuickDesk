# QuickDesk Signaling Server

QuickDesk 信令服务器，使用 Go 语言开发，提供设备注册、认证和 WebSocket 信令转发功能。

## 功能

- 设备注册（生成唯一9位设备ID）
- 临时访问码管理
- WebSocket 信令转发
- Redis 缓存
- PostgreSQL 持久化存储

## 依赖

- Go 1.21+
- PostgreSQL 15+
- Redis 7+

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

# Redis
docker run -d -p 6379:6379 redis:7
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

### HTTP API

- `POST /api/v1/devices/register` - 注册新设备
- `GET /api/v1/devices/:device_id` - 获取设备信息

### WebSocket

- `GET /signal/:device_id?access_code=xxx` - WebSocket 信令连接

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
