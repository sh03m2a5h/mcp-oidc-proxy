# MCP OAuth 2.1 認証プロキシ

このプロジェクトは、Model Context Protocol (MCP) サーバーへのリモートアクセスに OAuth 2.1/OIDC 認証を提供する汎用的なプロキシサーバーです。

## 機能

- **汎用OIDC対応**: Auth0、Google、Microsoft、GitHub等に対応
- **OAuth 2.1 + PKCE**: モダンなセキュリティ標準
- **認証バイパス**: 開発・テスト用の簡単モード
- **Redisセッション**: スケーラブルなセッション管理
- **WebSocketサポート**: MCPプロトコル完全対応
- **外部公開対応**: Cloudflare Tunnel/Ngrok等で簡単公開

## コンポーネント

- **OpenResty 1.27.1.2**: 認証プロキシサーバー（最新版）
- **Redis 7**: セッションストア
- **lua-resty-openidc 1.8.0**: OIDC認証ライブラリ（最新版）
- **lua-resty-session 4.0.5**: セッション管理ライブラリ

## クイックスタート

```bash
# プロキシを起動
./start-mcp.sh

# 接続先MCPサーバーを設定して再起動
MCP_TARGET_HOST=your-mcp-server.com MCP_TARGET_PORT=3000 docker compose up -d
```

アクセス先:
- **プロキシ**: http://localhost:8080
- **ヘルスチェック**: http://localhost:8080/health

## 認証プロバイダー設定

### 1. バイパスモード (デフォルト)
```bash
# 認証なしで簡単アクセス
AUTH_MODE=bypass ./start-mcp.sh
```

### 2. Auth0 (推奨)
```bash
# 新しいOIDC設定方式
OIDC_DISCOVERY_URL="https://your-domain.auth0.com/.well-known/openid-configuration" \
OIDC_CLIENT_ID="your-client-id" \
OIDC_CLIENT_SECRET="your-client-secret" \
AUTH_MODE=oidc ./start-mcp.sh

# レガシー設定方式もサポート
AUTH0_DOMAIN=your-domain.auth0.com \
AUTH0_CLIENT_ID=your-client-id \
AUTH0_CLIENT_SECRET=your-client-secret \
AUTH_MODE=oidc ./start-mcp.sh
```

**Auth0設定:**
- Application Type: Regular Web Applications
- Allowed Callback URLs: `http://localhost:8080/callback`
- Allowed Web Origins: `http://localhost:8080`
- Allowed Scopes: `openid email profile`

### 3. Google
```bash
OIDC_DISCOVERY_URL="https://accounts.google.com/.well-known/openid-configuration" \
OIDC_CLIENT_ID="your-google-client-id" \
OIDC_CLIENT_SECRET="your-google-client-secret" \
AUTH_MODE=oidc ./start-mcp.sh
```

**Google設定:**
- Google Cloud Console > APIs & Services > Credentials
- OAuth 2.0 Client IDs で Web Application を作成
- Authorized redirect URIs: `http://localhost:8080/callback`

### 4. Microsoft Azure AD
```bash
OIDC_DISCOVERY_URL="https://login.microsoftonline.com/{tenant-id}/v2.0/.well-known/openid-configuration" \
OIDC_CLIENT_ID="your-azure-client-id" \
OIDC_CLIENT_SECRET="your-azure-client-secret" \
AUTH_MODE=oidc ./start-mcp.sh
```

**Azure AD設定:**
- Azure Portal > App registrations
- Platform: Web, Redirect URI: `http://localhost:8080/callback`
- API permissions: `openid`, `email`, `profile`

## 環境変数一覧

| 変数名 | デフォルト | 説明 |
|------------|------------|--------|
| `AUTH_MODE` | `bypass` | 認証モード (`bypass` または `oidc`) |
| `OIDC_DISCOVERY_URL` | - | OIDCディスカバリーURL |
| `OIDC_CLIENT_ID` | - | OIDCクライアントID |
| `OIDC_CLIENT_SECRET` | - | OIDCクライアントシークレット |
| `OIDC_SCOPE` | `openid email profile` | 要求するスコープ |
| `OIDC_USE_PKCE` | `true` | PKCE使用有無 |
| `MCP_TARGET_HOST` | `localhost` | 接続先MCPサーバー |
| `MCP_TARGET_PORT` | `3000` | 接続先ポート |
| `AUTH0_DOMAIN` | - | レガシー: Auth0ドメイン |
| `AUTH0_CLIENT_ID` | - | レガシー: Auth0クライアントID |
| `AUTH0_CLIENT_SECRET` | - | レガシー: Auth0シークレット |

## 外部公開

### Cloudflare Tunnel
```bash
cloudflared tunnel --url http://localhost:8080
```

### Ngrok
```bash
ngrok http 8080
```

### カスタムドメインでの設定
外部公開時はコールバックURLを更新:
```
Callback URL: https://your-domain.com/callback
Web Origins: https://your-domain.com
```

## 使用例

### ローカルMCPサーバーへのプロキシ
```bash
MCP_TARGET_HOST=localhost MCP_TARGET_PORT=3000 docker compose up -d
```

### リモートMCPサーバーへのプロキシ
```bash
MCP_TARGET_HOST=mcp.example.com MCP_TARGET_PORT=443 docker compose up -d
```

### Auth0で本格運用
```bash
OIDC_DISCOVERY_URL="https://prod.auth0.com/.well-known/openid-configuration" \
OIDC_CLIENT_ID="prod-client-id" \
OIDC_CLIENT_SECRET="prod-secret" \
MCP_TARGET_HOST=prod-mcp.example.com \
MCP_TARGET_PORT=443 \
AUTH_MODE=oidc \
docker compose up -d
```

## トラブルシューティング

### ログ確認
```bash
docker logs mcp-proxy
docker logs mcp-redis
```

### ヘルスチェック
```bash
curl http://localhost:8080/health
```

### 認証テスト
```bash
# バイパスモードでのテスト
curl -H "MCP-Protocol-Version: 2025-03-26" http://localhost:8080/

# OIDC認証モードでのテスト
# ブラウザで http://localhost:8080 にアクセスして認証フローを確認
```

## 停止

```bash
./stop-mcp.sh
```

## アーキテクチャ

```
Client → [Cloudflare/Ngrok] → MCP Auth Proxy → MCP Server
                                      │
                                   Redis
                                      │
                              OIDC Provider
                           (Auth0/Google/etc)
```

## サポートされるOIDCプロバイダー

- **Auth0** (推奨): エンタープライズ向け認証サービス
- **Google**: Google Workspace / Gmailアカウント
- **Microsoft**: Azure AD / Microsoft 365
- **GitHub**: GitHubアカウント
- **その他**: OpenID Connect標準に準拠する任意のプロバイダー

## ライセンス

MIT License

## 注意事項

- このプロキシは開発・テスト環境向けです
- 本番環境では適切なセキュリティ設定を行ってください
- SSL/HTTPS機能は削除されています。外部公開時はトンネルサービスのSSL終端を利用してください