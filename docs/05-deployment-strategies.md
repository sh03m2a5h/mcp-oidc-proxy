# デプロイメント戦略

## デプロイメントオプション

### 1. スタンドアロンバイナリ

最もシンプルなデプロイメント方法。

```bash
# ダウンロードと実行
wget https://github.com/sh03m2a5h/mcp-oidc-proxy-go/releases/download/v1.0.0/mcp-oidc-proxy-linux-amd64
chmod +x mcp-oidc-proxy-linux-amd64
./mcp-oidc-proxy-linux-amd64 --config config.yaml
```

**メリット:**
- 依存関係なし
- 簡単なインストール
- 最小限のリソース使用

**デメリット:**
- 手動でのプロセス管理
- ログローテーションなし

### 2. Systemdサービス

本番環境での推奨方法。

```ini
# /etc/systemd/system/mcp-oidc-proxy.service
[Unit]
Description=MCP OIDC Proxy
After=network.target

[Service]
Type=simple
User=mcp-proxy
Group=mcp-proxy
ExecStart=/usr/local/bin/mcp-oidc-proxy --config /etc/mcp-proxy/config.yaml
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal
SyslogIdentifier=mcp-oidc-proxy

# セキュリティ設定
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/mcp-proxy

[Install]
WantedBy=multi-user.target
```

インストールスクリプト:
```bash
#!/bin/bash
# install.sh

# バイナリインストール
sudo cp mcp-oidc-proxy /usr/local/bin/
sudo chmod +x /usr/local/bin/mcp-oidc-proxy

# ユーザー作成
sudo useradd -r -s /bin/false mcp-proxy

# 設定ディレクトリ
sudo mkdir -p /etc/mcp-proxy
sudo cp config.yaml /etc/mcp-proxy/
sudo chown -R mcp-proxy:mcp-proxy /etc/mcp-proxy

# データディレクトリ
sudo mkdir -p /var/lib/mcp-proxy
sudo chown -R mcp-proxy:mcp-proxy /var/lib/mcp-proxy

# Systemdサービス
sudo cp mcp-oidc-proxy.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable mcp-oidc-proxy
sudo systemctl start mcp-oidc-proxy
```

### 3. Docker

```dockerfile
# deployments/docker/Dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o mcp-oidc-proxy ./cmd/mcp-oidc-proxy

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata

RUN adduser -D -g '' appuser
USER appuser

COPY --from=builder /build/mcp-oidc-proxy /usr/local/bin/
COPY --from=builder /build/configs/config.example.yaml /etc/mcp-proxy/config.yaml

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/mcp-oidc-proxy"]
CMD ["--config", "/etc/mcp-proxy/config.yaml"]
```

Docker Compose:
```yaml
# docker-compose.yaml
version: '3.8'

services:
  mcp-proxy:
    image: mcp-oidc-proxy:latest
    container_name: mcp-proxy
    ports:
      - "8080:8080"
    environment:
      - MCP_TARGET_HOST=host.docker.internal
      - MCP_TARGET_PORT=3000
      - OIDC_DISCOVERY_URL=https://your-domain.auth0.com/.well-known/openid-configuration
      - OIDC_CLIENT_ID=${OIDC_CLIENT_ID}
      - OIDC_CLIENT_SECRET=${OIDC_CLIENT_SECRET}
    volumes:
      - ./config.yaml:/etc/mcp-proxy/config.yaml:ro
    restart: unless-stopped
    
  redis:
    image: redis:7-alpine
    container_name: mcp-redis
    volumes:
      - redis_data:/data
    restart: unless-stopped

volumes:
  redis_data:
```

### 4. Kubernetes

```yaml
# deployments/kubernetes/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mcp-oidc-proxy
  namespace: mcp-system
spec:
  replicas: 3
  selector:
    matchLabels:
      app: mcp-oidc-proxy
  template:
    metadata:
      labels:
        app: mcp-oidc-proxy
    spec:
      serviceAccountName: mcp-oidc-proxy
      containers:
      - name: proxy
        image: mcp-oidc-proxy:v1.0.0
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: metrics
        env:
        - name: SESSION_STORE
          value: "redis"
        - name: REDIS_URL
          value: "redis://mcp-redis:6379"
        envFrom:
        - secretRef:
            name: mcp-oidc-secrets
        - configMapRef:
            name: mcp-oidc-config
        livenessProbe:
          httpGet:
            path: /health
            port: http
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: http
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "500m"
        securityContext:
          runAsNonRoot: true
          runAsUser: 1000
          readOnlyRootFilesystem: true
          capabilities:
            drop:
            - ALL
---
apiVersion: v1
kind: Service
metadata:
  name: mcp-oidc-proxy
  namespace: mcp-system
spec:
  selector:
    app: mcp-oidc-proxy
  ports:
  - name: http
    port: 80
    targetPort: http
  - name: metrics
    port: 9090
    targetPort: metrics
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: mcp-oidc-config
  namespace: mcp-system
data:
  MCP_TARGET_HOST: "mcp-server.mcp-system.svc.cluster.local"
  MCP_TARGET_PORT: "3000"
  AUTH_MODE: "oidc"
---
apiVersion: v1
kind: Secret
metadata:
  name: mcp-oidc-secrets
  namespace: mcp-system
type: Opaque
stringData:
  OIDC_CLIENT_ID: "your-client-id"
  OIDC_CLIENT_SECRET: "your-client-secret"
```

Helm Chart構造:
```
helm/mcp-oidc-proxy/
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── configmap.yaml
│   ├── secret.yaml
│   ├── hpa.yaml
│   └── ingress.yaml
```

### 5. サーバーレス（AWS Lambda）

API Gateway + Lambda構成:

```go
// cmd/lambda/main.go
package main

import (
    "github.com/aws/aws-lambda-go/lambda"
    "github.com/awslabs/aws-lambda-go-api-proxy/gin"
)

var ginLambda *ginadapter.GinLambda

func init() {
    // Gin appの初期化
    app := setupApp()
    ginLambda = ginadapter.New(app)
}

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    return ginLambda.ProxyWithContext(ctx, req)
}

func main() {
    lambda.Start(Handler)
}
```

## 設定管理戦略

### 1. 環境別設定

```
configs/
├── base.yaml          # 共通設定
├── development.yaml   # 開発環境
├── staging.yaml      # ステージング環境
└── production.yaml   # 本番環境
```

### 2. シークレット管理

**オプション1: 環境変数**
```bash
export OIDC_CLIENT_SECRET=$(vault kv get -field=secret secret/mcp-proxy/oidc)
```

**オプション2: HashiCorp Vault**
```yaml
# Vault統合
vault:
  enabled: true
  address: "https://vault.example.com"
  auth_method: "kubernetes"
  role: "mcp-proxy"
  secrets:
    - path: "secret/data/mcp-proxy/oidc"
      key: "client_secret"
      env: "OIDC_CLIENT_SECRET"
```

**オプション3: AWS Secrets Manager**
```go
// AWS Secrets Manager統合
func loadSecretsFromAWS() error {
    sess := session.Must(session.NewSession())
    svc := secretsmanager.New(sess)
    
    result, err := svc.GetSecretValue(&secretsmanager.GetSecretValueInput{
        SecretId: aws.String("mcp-proxy/oidc"),
    })
    // ...
}
```

## モニタリングとアラート

### Prometheus設定
```yaml
# prometheus.yaml
scrape_configs:
  - job_name: 'mcp-oidc-proxy'
    static_configs:
      - targets: ['mcp-proxy:9090']
    metric_relabel_configs:
      - source_labels: [__name__]
        regex: 'go_.*'
        action: drop
```

### Grafanaダッシュボード
- リクエストレート
- レスポンスタイム
- エラー率
- アクティブセッション数
- バックエンド可用性

### アラートルール
```yaml
groups:
- name: mcp-proxy
  rules:
  - alert: HighErrorRate
    expr: rate(mcp_proxy_requests_total{status=~"5.."}[5m]) > 0.05
    for: 5m
    annotations:
      summary: "High error rate detected"
      
  - alert: BackendDown
    expr: mcp_proxy_backend_up == 0
    for: 1m
    annotations:
      summary: "Backend is unreachable"
```

## バックアップとリカバリー

### セッションデータ
- Redisの定期バックアップ
- レプリケーション設定
- フェイルオーバー戦略

### 設定のバックアップ
```bash
# 設定バックアップスクリプト
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
tar -czf "mcp-proxy-config-${DATE}.tar.gz" /etc/mcp-proxy/
aws s3 cp "mcp-proxy-config-${DATE}.tar.gz" s3://backup-bucket/mcp-proxy/
```

## アップグレード戦略

### ローリングアップデート
1. 新バージョンのヘルスチェック
2. 1インスタンスずつ更新
3. ヘルスチェック確認
4. 次のインスタンスへ

### Blue-Greenデプロイメント
1. 新環境（Green）の構築
2. テストとバリデーション
3. トラフィック切り替え
4. 旧環境（Blue）の削除

### カナリアデプロイメント
```yaml
# Istio/Flaggerを使用した例
apiVersion: flagger.app/v1beta1
kind: Canary
metadata:
  name: mcp-oidc-proxy
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: mcp-oidc-proxy
  progressDeadlineSeconds: 60
  service:
    port: 80
  analysis:
    interval: 30s
    threshold: 5
    maxWeight: 50
    stepWeight: 10
    metrics:
    - name: request-success-rate
      thresholdRange:
        min: 99
      interval: 1m
```

## トラブルシューティング

### 一般的な問題と解決策

1. **認証ループ**
   - Cookie設定の確認
   - リダイレクトURLの確認

2. **セッション喪失**
   - Redis接続の確認
   - セッションTTLの確認

3. **プロキシエラー**
   - バックエンド到達性
   - タイムアウト設定

### デバッグツール
```bash
# ヘルスチェック
curl http://localhost:8080/health

# メトリクス確認
curl http://localhost:8080/metrics | grep mcp_proxy

# ログ確認
journalctl -u mcp-oidc-proxy -f

# セッション確認（Redis）
redis-cli
> KEYS mcp:session:*
> TTL mcp:session:xxxxx
```

## まとめ

このドキュメントでは、MCP OIDC ProxyのGo実装における様々なデプロイメント戦略を説明しました。環境や要件に応じて適切な方法を選択してください。

重要なポイント:
- シンプルさを求めるなら単一バイナリ
- 本番環境ではSystemdまたはKubernetes
- 高可用性が必要ならKubernetesでのマルチレプリカ
- 適切なモニタリングとアラートの設定
- セキュアなシークレット管理
