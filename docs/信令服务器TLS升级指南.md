# 信令服务器 HTTPS/WSS 升级指南

生产环境的 QuickDesk 信令入口必须使用 HTTPS/WSS。Docker 中的 Go 服务仅
监听宿主机 loopback 地址，公网客户端通过 Nginx 的 443 端口访问：

```text
Qt / WebClient / Android / Remoting
        https:// 或 wss://
                 |
              Nginx :443
                 |
       http://127.0.0.1:<backend-port>
                 |
          Docker signaling-server
```

不要向公网开放 Docker 后端端口。部署脚本会将其绑定为
`127.0.0.1:<backend-port>:8000`。

## 正式部署

准备一个已解析到服务器公网 IP 的域名，并先取得证书。使用 Let's Encrypt 时：

```bash
sudo certbot certonly --standalone -d signal.example.com
```

然后在 `QuickDesk/SignalingServer` 中部署。以下示例使用 Let’s Encrypt 默认
路径：

```bash
./deploy-build.sh --port 8000 --domain signal.example.com
```

使用其他 CA 或自建证书时，显式指定证书链和私钥：

```bash
./deploy-build.sh --port 8000 --domain signal.example.com \
  --cert-file /etc/ssl/quickdesk/fullchain.pem \
  --key-file /etc/ssl/quickdesk/privkey.pem
```

`deploy.sh`、`deploy-pull.sh` 和 `deploy-offline.sh` 也支持相同的
`--domain`、`--cert-file`、`--key-file` 参数。证书和私钥必须成对提供；未提供时
脚本使用 `/etc/letsencrypt/live/<domain>/fullchain.pem` 与 `privkey.pem`。

生产防火墙只需开放 80 和 443：

```bash
sudo firewall-cmd --permanent --add-service=http
sudo firewall-cmd --permanent --add-service=https
sudo firewall-cmd --permanent --remove-port=8000/tcp
sudo firewall-cmd --reload
```

部署脚本生成的 Nginx 配置会将 HTTP 80 重定向到 HTTPS 443，并为
`/v1/realtime/events` 与 `/v1/realtime/signal` 配置 WebSocket Upgrade。

## 验证

```bash
curl -fsS https://signal.example.com/health
curl -I http://signal.example.com/health  # 预期 301/308 到 HTTPS
```

客户端配置使用：

```text
wss://signal.example.com
```

客户端会将同一基址转换为 `https://signal.example.com/v1/...` 访问 REST API，
并使用 `wss://signal.example.com/v1/realtime/...` 访问实时事件和远控信令。

## 旧客户端临时兼容

新代码和默认部署脚本不会开启 HTTP/WS。若旧客户端仍固定连接
`ws://signal.example.com:8060`，可在升级窗口内手工创建独立 Nginx 配置
`/etc/nginx/conf.d/quickdesk-legacy.conf`：

```nginx
server {
    listen 8060;
    server_name signal.example.com;

    location /v1/realtime/ {
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
```

上述兼容配置依赖默认 `quickdesk.conf` 中已定义的 `$connection_upgrade`。
应用后仅在迁移窗口开放 8060：

```bash
sudo nginx -t && sudo systemctl reload nginx
sudo firewall-cmd --permanent --add-port=8060/tcp
sudo firewall-cmd --reload
```

8060 上的 HTTP/WS 是明文传输，不应长期保留。旧客户端升级完成后关闭它：

```bash
sudo rm /etc/nginx/conf.d/quickdesk-legacy.conf
sudo nginx -t && sudo systemctl reload nginx
sudo firewall-cmd --permanent --remove-port=8060/tcp
sudo firewall-cmd --reload
```