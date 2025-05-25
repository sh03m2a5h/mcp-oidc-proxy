# MCP OIDC Proxy

Model Context Protocol (MCP) サーバー用の本番環境対応 OAuth 2.1/OIDC 認証プロキシ。モダンな認証機能でMCPエンドポイントを保護する単一のGoバイナリです。

> **🤖 注意**: このプロジェクトは主にAI（Claude、GitHub Copilot、Gemini Code Assist）によって開発・保守されています。人間による直接的な開発作業は最小限に抑えられており、コード品質の確保はAIレビューツールによって行われています。

## 🚀 クイックスタート

```bash
# インストール (Linux/macOS)
curl -sSL https://raw.githubusercontent.com/sh03m2a5h/mcp-oidc-proxy/main/install.sh | bash

# OIDC設定 (Auth0の例)
export OIDC_DISCOVERY_URL="https://your-domain.auth0.com/.well-known/openid-configuration"
export OIDC_CLIENT_ID="your-client-id"
export OIDC_CLIENT_SECRET="your-client-secret"

# 実行
mcp-oidc-proxy
```

`localhost:3000`のMCPサーバーが`localhost:8080`でOIDC認証により保護されます！

## 🎯 何をするものか

任意のMCPサーバーにエンタープライズグレードの認証を追加：

```
[インターネット] → [Cloudflare] → [MCP OIDC Proxy :8080] → [あなたのMCPサーバー :3000]
                                           ↓
                                   [OIDCプロバイダー]
                                (Auth0/Google/Azure)
```

## ✨ 機能

- 🔐 **汎用OIDC対応**: Auth0、Google、Microsoft、GitHub、その他のOIDCプロバイダーに対応
- 🚀 **単一バイナリ**: Docker不要、依存関係なし - ダウンロードして実行するだけ
- 🛡️ **モダンなセキュリティ**: OAuth 2.1 + PKCE、セキュアセッション、CSPヘッダー
- 📊 **本番環境対応**: Prometheusメトリクス、ヘルスチェック、OpenTelemetryトレーシング
- 🔄 **完全なプロトコルサポート**: HTTP、SSE/WebSocketストリーミング、MCPプロトコル
- ⚡ **高パフォーマンス**: オーバーヘッド10ms未満、1000以上の同時接続に対応

## 📦 インストール

### バイナリリリース（推奨）
```bash
# ワンライナーインストール
curl -sSL https://raw.githubusercontent.com/sh03m2a5h/mcp-oidc-proxy/main/install.sh | bash

# または直接ダウンロード
wget https://github.com/sh03m2a5h/mcp-oidc-proxy/releases/latest/download/mcp-oidc-proxy-$(uname -s)-$(uname -m)
```

### ソースからビルド
```bash
git clone https://github.com/sh03m2a5h/mcp-oidc-proxy.git
cd mcp-oidc-proxy/go
make build
```

## ⚙️ 設定

### 環境変数
```bash
# 必須
export OIDC_DISCOVERY_URL="https://your-provider/.well-known/openid-configuration"
export OIDC_CLIENT_ID="your-client-id"
export OIDC_CLIENT_SECRET="your-client-secret"

# オプション
export MCP_TARGET_HOST="localhost"     # デフォルト: localhost
export MCP_TARGET_PORT="3000"          # デフォルト: 3000
export SERVER_PORT="8080"              # デフォルト: 8080
export LOG_LEVEL="info"                # デフォルト: info
export SESSION_STORE="memory"          # memory または redis
```

### 設定ファイル (YAML)
```yaml
server:
  host: 0.0.0.0
  port: 8080

proxy:
  target_host: localhost
  target_port: 3000
  retry:
    max_attempts: 3
    backoff: 100ms

auth:
  mode: oidc  # または 'bypass' (開発用)

oidc:
  discovery_url: https://your-domain.auth0.com/.well-known/openid-configuration
  client_id: your-client-id
  client_secret: your-client-secret
  redirect_url: http://localhost:8080/callback
```

## 🚀 本番環境へのデプロイ

### Cloudflare Tunnels（推奨）
```bash
# MCP OIDCプロキシを起動
./mcp-oidc-proxy &

# Cloudflareトンネルで公開
cloudflared tunnel --url http://localhost:8080
```

### Systemdサービス
```bash
# サービスファイルをコピー
sudo cp mcp-oidc-proxy.service /etc/systemd/system/

# サービスを有効化して起動
sudo systemctl enable mcp-oidc-proxy
sudo systemctl start mcp-oidc-proxy
```

## 📊 モニタリング

- **ヘルスチェック**: `GET /health`
- **メトリクス**: `GET /metrics` (Prometheus形式)
- **トレーシング**: OpenTelemetry対応

## 🔄 最近の更新

### v0.5.0（最新）
- **SSE/WebSocketストリーミングサポート**: ストリーミングプロトコルでのパニック問題を修正
- **バイパスモード**: 認証をバイパスする開発/テストモードを追加
- **安定性の向上**: 長時間接続のエラーハンドリングを改善
- **AI支援開発**: CopilotとGeminiレビューによるコード品質向上

### v0.4.0
- **モニタリングと可観測性**: Prometheusメトリクスとトレーシング
- **ヘルスチェック**: サブシステムステータス付き組み込みヘルスエンドポイント
- **サーキットブレーカー**: 自動バックエンド障害保護

## 🤝 コントリビューション

プルリクエストを歓迎します！気軽に提出してください。

## 📜 ライセンス

MITライセンス - 詳細は[LICENSE](LICENSE)ファイルを参照してください。

## 🙏 謝辞

- [Model Context Protocol](https://modelcontextprotocol.io)エコシステム向けに構築
- シンプルで安全なMCPサーバーデプロイメントの必要性から着想
- [mcp-proxy](https://github.com/sparfenyuk/mcp-proxy)互換性のためのSSE/WebSocketストリーミングサポート
- **開発**: このプロジェクトは主にClaude (Anthropic)、GitHub Copilot、Gemini Code Assistによって開発されています

---

### レガシー実装

オリジナルのNginx/Lua実装は[legacy/nginx-implementation](legacy/nginx-implementation/)ディレクトリで利用可能です。Go実装が現在の主要かつ推奨バージョンです。

## 🤖 AI駆動開発について

このプロジェクトは、人間の開発者による最小限の介入で、AIツールによって開発・保守されている実験的なプロジェクトです：

- **コード生成**: Claude (Anthropic) が主要な開発を担当
- **コードレビュー**: GitHub CopilotとGemini Code Assistによる自動レビュー
- **テスト**: AIが生成したテストケースとAIによるテスト実装
- **ドキュメント**: このREADMEを含む全てのドキュメントがAI生成

人間の役割は主にプロジェクトの方向性の決定とAIツール間の調整に限定されています。