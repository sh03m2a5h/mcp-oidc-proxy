# API仕様とConfiguration設計

## API エンドポイント

### 認証関連

#### GET /
- **説明**: メインエントリーポイント
- **認証**: 必要
- **動作**: 
  - 認証済み: MCPサーバーへプロキシ
  - 未認証: OIDCプロバイダーへリダイレクト

#### GET /callback
- **説明**: OIDC認証コールバック
- **パラメータ**:
  - `code`: 認証コード
  - `state`: CSRF対策用状態
- **動作**: トークン交換とセッション作成

#### GET /logout
- **説明**: ログアウト処理
- **動作**: セッション削除とOIDCプロバイダーのログアウト

### 管理エンドポイント

#### GET /health
- **説明**: ヘルスチェック
- **レスポンス**:
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": 3600,
  "backend_status": "reachable"
}
```

**backend_status の可能な値:**
- `"reachable"`: バックエンドMCPサーバーに接続可能
- `"unreachable"`: バックエンドMCPサーバーに接続不可
- `"unknown"`: バックエンドの状態が不明（初期化中など）

#### GET /metrics
- **説明**: Prometheusメトリクス
- **形式**: Prometheus text format
- **メトリクス例**:
```
# HELP mcp_proxy_requests_total Total number of requests
# TYPE mcp_proxy_requests_total counter
mcp_proxy_requests_total{method="GET",status="200"} 1234

# HELP mcp_proxy_active_sessions Number of active sessions
# TYPE mcp_proxy_active_sessions gauge
mcp_proxy_active_sessions 42
```

#### GET /api/v1/session
- **説明**: 現在のセッション情報
- **認証**: 必要
- **レスポンス**:
```json
{
  "id": "session-id",
  "user": {
    "id": "user123",
    "email": "user@example.com",
    "name": "John Doe"
  },
  "expires_at": "2024-01-01T00:00:00Z"
}
```

## Configuration設計

### 設定の優先順位
1. コマンドライン引数
2. 環境変数
3. 設定ファイル
4. デフォルト値

### 設定ファイル形式

YAML形式（`config.yaml`）:
```yaml
# サーバー設定
server:
  host: "0.0.0.0"
  port: 8080
  tls:
    enabled: false
    cert_file: ""
    key_file: ""
  
  # タイムアウト設定
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "120s"

# プロキシ設定
proxy:
  target_host: "localhost"
  target_port: 3000
  target_scheme: "http"
  
  # リトライ設定
  retry:
    max_attempts: 3
    backoff: "100ms"
  
  # サーキットブレーカー
  circuit_breaker:
    threshold: 5
    timeout: "60s"

# OIDC設定
oidc:
  # プロバイダー設定
  discovery_url: "https://your-domain.auth0.com/.well-known/openid-configuration"
  client_id: "your-client-id"
  client_secret: "your-client-secret"
  
  # スコープとオプション
  scopes: ["openid", "email", "profile"]
  use_pkce: true
  
  # URLカスタマイズ
  redirect_url: "http://localhost:8080/callback"
  post_logout_redirect_url: "http://localhost:8080/"

# セッション設定
session:
  # ストアタイプ: memory | redis
  store: "memory"
  
  # セッションオプション
  ttl: "24h"
  cookie_name: "mcp_session"
  cookie_domain: ""
  cookie_path: "/"
  cookie_secure: false  # 本番環境（HTTPS）では true に設定してください
  cookie_http_only: true
  cookie_same_site: "lax"
  
  # Redis設定（store: redisの場合）
  redis:
    url: "redis://localhost:6379"
    password: ""
    db: 0
    key_prefix: "mcp:session:"

# 認証設定
auth:
  # モード: oidc | bypass
  mode: "oidc"
  
  # ヘッダー設定
  headers:
    user_id: "X-User-ID"
    user_email: "X-User-Email"
    user_name: "X-User-Name"
    user_groups: "X-User-Groups"
  
  # アクセス制御
  access_control:
    # 認証不要パス
    public_paths:
      - "/health"
      - "/metrics"
    
    # グループベースアクセス制御
    required_groups: []

# ロギング設定
logging:
  level: "info"  # debug, info, warn, error
  format: "json" # json, text
  output: "stdout" # stdout, stderr, file
  file:
    path: "/var/log/mcp-proxy.log"
    max_size: "100MB"
    max_age: "7d"
    max_backups: 5

# メトリクス設定
metrics:
  enabled: true
  path: "/metrics"
  
# トレーシング設定
tracing:
  enabled: false
  provider: "jaeger" # jaeger, zipkin
  endpoint: "http://localhost:14268/api/traces"
  service_name: "mcp-oidc-proxy"
  sample_rate: 0.1
```

### 環境変数マッピング

| 環境変数 | 設定パス | 説明 |
|---------|---------|------|
| `MCP_HOST` | `server.host` | リスンアドレス |
| `MCP_PORT` | `server.port` | リスンポート |
| `MCP_TARGET_HOST` | `proxy.target_host` | プロキシ先ホスト |
| `MCP_TARGET_PORT` | `proxy.target_port` | プロキシ先ポート |
| `AUTH_MODE` | `auth.mode` | 認証モード |
| `OIDC_DISCOVERY_URL` | `oidc.discovery_url` | OIDC Discovery URL |
| `OIDC_CLIENT_ID` | `oidc.client_id` | クライアントID |
| `OIDC_CLIENT_SECRET` | `oidc.client_secret` | クライアントシークレット |
| `SESSION_STORE` | `session.store` | セッションストア |
| `REDIS_URL` | `session.redis.url` | Redis URL |
| `LOG_LEVEL` | `logging.level` | ログレベル |

### コマンドライン引数

```bash
mcp-oidc-proxy [flags]

Flags:
  -c, --config string        設定ファイルパス (default "config.yaml")
  -h, --host string         リスンアドレス (default "0.0.0.0")
  -p, --port int           リスンポート (default 8080)
      --target-host string  プロキシ先ホスト
      --target-port int     プロキシ先ポート
      --auth-mode string    認証モード [oidc|bypass]
      --log-level string    ログレベル [debug|info|warn|error]
  -v, --version            バージョン表示
      --help               ヘルプ表示
```

## ヘッダー仕様

### リクエストヘッダー（MCPサーバーへ）
認証済みユーザー情報を以下のヘッダーで送信:

| ヘッダー名 | 説明 | 例 |
|-----------|------|-----|
| `X-User-ID` | ユーザーID | `auth0|123456` |
| `X-User-Email` | メールアドレス | `user@example.com` |
| `X-User-Name` | 表示名 | `John Doe` |
| `X-User-Groups` | グループ（カンマ区切り） | `admin,users` |
| `X-Auth-Provider` | 認証プロバイダー | `auth0` |
| `X-Request-ID` | リクエストID | `uuid-v4` |

### レスポンスヘッダー
| ヘッダー名 | 説明 |
|-----------|------|
| `X-Proxy-Version` | プロキシバージョン |
| `X-Request-ID` | リクエストID |

## エラーレスポンス

標準的なエラーレスポンス形式:
```json
{
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Authentication required",
    "details": {
      "redirect_url": "https://..."
    }
  },
  "request_id": "uuid-v4",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

エラーコード:
- `UNAUTHORIZED`: 認証が必要
- `FORBIDDEN`: アクセス権限なし
- `INVALID_SESSION`: セッション無効
- `OIDC_ERROR`: OIDC認証エラー
- `PROXY_ERROR`: プロキシエラー
- `INTERNAL_ERROR`: 内部エラー

## 次のドキュメント

→ [モジュール構造と依存関係](./03-module-structure.md)
