# モジュール構造と依存関係

## ディレクトリ構造

```
mcp-oidc-proxy-go/
├── cmd/
│   └── mcp-oidc-proxy/
│       └── main.go              # エントリーポイント
├── internal/
│   ├── config/
│   │   ├── config.go           # 設定構造体と読み込み
│   │   ├── validate.go         # 設定検証
│   │   └── defaults.go         # デフォルト値
│   ├── server/
│   │   ├── server.go           # HTTPサーバー実装
│   │   ├── middleware.go       # 共通ミドルウェア
│   │   └── routes.go           # ルーティング定義
│   ├── auth/
│   │   ├── oidc/
│   │   │   ├── handler.go      # OIDC認証ハンドラー
│   │   │   ├── provider.go     # OIDCプロバイダー抽象化
│   │   │   └── pkce.go         # PKCE実装
│   │   └── bypass/
│   │       └── handler.go      # バイパスモード実装
│   ├── session/
│   │   ├── manager.go          # セッションマネージャーインターフェース
│   │   ├── memory/
│   │   │   └── store.go        # メモリセッションストア
│   │   └── redis/
│   │       └── store.go        # Redisセッションストア
│   ├── proxy/
│   │   ├── handler.go          # リバースプロキシハンドラー
│   │   ├── websocket.go        # WebSocketプロキシ
│   │   └── circuit_breaker.go  # サーキットブレーカー
│   ├── middleware/
│   │   ├── logging.go          # ロギングミドルウェア
│   │   ├── metrics.go          # メトリクスミドルウェア
│   │   ├── tracing.go          # トレーシングミドルウェア
│   │   └── recovery.go         # パニックリカバリー
│   └── utils/
│       ├── logger.go           # ロガー実装
│       ├── errors.go           # エラー定義
│       └── crypto.go           # 暗号化ユーティリティ
├── pkg/
│   └── version/
│       └── version.go          # バージョン情報
├── configs/
│   └── config.example.yaml     # 設定ファイルサンプル
├── scripts/
│   ├── build.sh               # ビルドスクリプト
│   └── release.sh             # リリーススクリプト
├── deployments/
│   ├── docker/
│   │   └── Dockerfile         # Dockerイメージ定義
│   └── kubernetes/
│       ├── deployment.yaml    # Kubernetesデプロイメント
│       └── service.yaml       # Kubernetesサービス
├── go.mod                     # Goモジュール定義
├── go.sum                     # 依存関係ロックファイル
├── Makefile                   # ビルドタスク定義
└── README.md                  # プロジェクト説明
```

## 主要な依存関係

### go.mod

```go
module github.com/sh03m2a5h/mcp-oidc-proxy-go

go 1.22

require (
    // Web Framework & HTTP
    github.com/gin-gonic/gin v1.9.1
    github.com/gorilla/websocket v1.5.1
    
    // OIDC & OAuth2
    github.com/coreos/go-oidc/v3 v3.9.0
    golang.org/x/oauth2 v0.16.0
    
    // Session Management
    github.com/go-redis/redis/v8 v8.11.5
    
    // Configuration
    github.com/spf13/viper v1.18.2
    github.com/spf13/cobra v1.8.0
    
    // Logging
    go.uber.org/zap v1.26.0
    
    // Metrics & Tracing
    github.com/prometheus/client_golang v1.18.0
    go.opentelemetry.io/otel v1.21.0
    go.opentelemetry.io/otel/trace v1.21.0
    
    // Utilities
    github.com/google/uuid v1.5.0
    github.com/sony/gobreaker v0.5.0  // Circuit breaker
    golang.org/x/sync v0.5.0          // errgroup
    
    // Testing
    github.com/stretchr/testify v1.8.4
    github.com/golang/mock v1.6.0
)
```

## モジュール詳細設計

### 1. Config Module (`internal/config/`)

```go
// config.go
type Config struct {
    Server   ServerConfig   `mapstructure:"server"`
    Proxy    ProxyConfig    `mapstructure:"proxy"`
    OIDC     OIDCConfig     `mapstructure:"oidc"`
    Session  SessionConfig  `mapstructure:"session"`
    Auth     AuthConfig     `mapstructure:"auth"`
    Logging  LoggingConfig  `mapstructure:"logging"`
    Metrics  MetricsConfig  `mapstructure:"metrics"`
    Tracing  TracingConfig  `mapstructure:"tracing"`
}

func Load(configPath string) (*Config, error) {
    // 1. デフォルト値を設定
    // 2. 設定ファイルを読み込み
    // 3. 環境変数をオーバーライド
    // 4. コマンドライン引数をオーバーライド
    // 5. 検証
}
```

### 2. Server Module (`internal/server/`)

```go
// server.go
type Server struct {
    config      *config.ServerConfig
    router      *gin.Engine
    oidcHandler auth.Handler
    proxy       proxy.Handler
    session     session.Manager
}

func New(cfg *config.Config, deps Dependencies) *Server {
    // サーバー初期化
}

func (s *Server) Run() error {
    // サーバー起動
}
```

### 3. Auth Module (`internal/auth/`)

```go
// oidc/handler.go
type Handler interface {
    HandleLogin(c *gin.Context)
    HandleCallback(c *gin.Context)
    HandleLogout(c *gin.Context)
    Middleware() gin.HandlerFunc
}

type OIDCHandler struct {
    provider     *oidc.Provider
    oauth2Config *oauth2.Config
    session      session.Manager
}
```

### 4. Session Module (`internal/session/`)

```go
// manager.go
type Manager interface {
    Create(ctx context.Context, userInfo *UserInfo) (*Session, error)
    Get(ctx context.Context, sessionID string) (*Session, error)
    Update(ctx context.Context, session *Session) error
    Delete(ctx context.Context, sessionID string) error
}

type Session struct {
    ID        string
    UserInfo  *UserInfo
    CreatedAt time.Time
    ExpiresAt time.Time
}

type UserInfo struct {
    ID     string
    Email  string
    Name   string
    Groups []string
}
```

### 5. Proxy Module (`internal/proxy/`)

```go
// handler.go
type Handler interface {
    ServeHTTP(w http.ResponseWriter, r *http.Request)
}

type ReverseProxy struct {
    target         *url.URL
    httpProxy      *httputil.ReverseProxy
    circuitBreaker *gobreaker.CircuitBreaker
}

// WebSocketサポート
func (p *ReverseProxy) handleWebSocket(w http.ResponseWriter, r *http.Request) {
    // WebSocketプロキシ実装
}
```

## インターフェース設計

### 主要インターフェース

```go
// 認証ハンドラー
type AuthHandler interface {
    HandleLogin(c *gin.Context)
    HandleCallback(c *gin.Context)
    HandleLogout(c *gin.Context)
    Middleware() gin.HandlerFunc
}

// セッションマネージャー
type SessionManager interface {
    Create(ctx context.Context, userInfo *UserInfo) (*Session, error)
    Get(ctx context.Context, sessionID string) (*Session, error)
    Update(ctx context.Context, session *Session) error
    Delete(ctx context.Context, sessionID string) error
}

// プロキシハンドラー
type ProxyHandler interface {
    ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// ヘルスチェッカー
type HealthChecker interface {
    Check(ctx context.Context) error
}
```

## エラーハンドリング

```go
// internal/utils/errors.go
type ErrorCode string

const (
    ErrUnauthorized   ErrorCode = "UNAUTHORIZED"
    ErrForbidden      ErrorCode = "FORBIDDEN"
    ErrInvalidSession ErrorCode = "INVALID_SESSION"
    ErrOIDCError      ErrorCode = "OIDC_ERROR"
    ErrProxyError     ErrorCode = "PROXY_ERROR"
    ErrInternalError  ErrorCode = "INTERNAL_ERROR"
)

type AppError struct {
    Code    ErrorCode
    Message string
    Details map[string]interface{}
}

func (e *AppError) Error() string {
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
```

## ビルドとテスト

### Makefile

```makefile
.PHONY: all build test clean

VERSION ?= $(shell git describe --tags --always --dirty)
LDFLAGS := -X github.com/sh03m2a5h/mcp-oidc-proxy-go/pkg/version.Version=$(VERSION)

all: test build

build:
	go build -ldflags "$(LDFLAGS)" -o bin/mcp-oidc-proxy ./cmd/mcp-oidc-proxy

test:
	go test -v -race -coverprofile=coverage.out ./...

test-integration:
	go test -v -tags=integration ./tests/integration/...

lint:
	golangci-lint run

clean:
	rm -rf bin/ coverage.out

docker:
	docker build -t mcp-oidc-proxy:$(VERSION) -f deployments/docker/Dockerfile .

# クロスコンパイル
build-all:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/mcp-oidc-proxy-linux-amd64 ./cmd/mcp-oidc-proxy
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/mcp-oidc-proxy-darwin-amd64 ./cmd/mcp-oidc-proxy
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/mcp-oidc-proxy-windows-amd64.exe ./cmd/mcp-oidc-proxy
```

## Lambda用の追加依存関係（オプション）

AWS Lambdaでのデプロイメントをサポートする場合、以下の依存関係も追加：

```go
require (
    // AWS Lambda
    github.com/aws/aws-lambda-go v1.41.0
    github.com/awslabs/aws-lambda-go-api-proxy v0.16.0
)
```

## 次のドキュメント

→ [実装ロードマップ](./04-implementation-roadmap.md)
