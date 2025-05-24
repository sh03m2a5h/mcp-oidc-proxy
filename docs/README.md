# MCP OIDC Proxy Go実装 - 設計ドキュメント

このディレクトリには、MCP OIDC ProxyのGo実装に関する設計ドキュメントが含まれています。

## ドキュメント一覧

### 1. [アーキテクチャ概要](./01-architecture-overview.md)
- システム全体の設計思想
- コンポーネント構成
- データフロー
- セキュリティ考慮事項
- 非機能要件

### 2. [API仕様とConfiguration設計](./02-api-configuration.md)
- RESTful APIエンドポイント仕様
- 設定ファイル形式（YAML）
- 環境変数マッピング
- コマンドライン引数
- エラーレスポンス形式

### 3. [モジュール構造と依存関係](./03-module-structure.md)
- ディレクトリ構造
- 主要な外部依存関係（go.mod）
- 内部モジュールの詳細設計
- インターフェース定義
- ビルドとテスト戦略

### 4. [実装ロードマップ](./04-implementation-roadmap.md)
- 6つの実装フェーズ
- 各フェーズの詳細タスク
- マイルストーンと期限
- リスクと対策
- 成功指標

### 5. [デプロイメント戦略](./05-deployment-strategies.md)
- 5つのデプロイメントオプション
- 設定管理戦略
- モニタリングとアラート
- バックアップとリカバリー
- アップグレード戦略

## 設計の特徴

### 🎯 主要な設計目標
- **単一バイナリ**: 依存関係を最小限に抑えた簡単なデプロイ
- **高パフォーマンス**: 1000+ 同時接続、< 10ms レイテンシ
- **互換性**: 既存のNginx実装との機能互換性
- **拡張性**: プラグイン可能なセッション管理とOIDCプロバイダー

### 🔧 技術スタック
- **言語**: Go 1.22+
- **Webフレームワーク**: Gin
- **OIDC**: coreos/go-oidc
- **セッション**: メモリ/Redis
- **設定**: Viper + Cobra
- **ログ**: Zap
- **メトリクス**: Prometheus
- **トレーシング**: OpenTelemetry

### 📊 アーキテクチャの利点

| 特徴 | Nginx版 | Go版 |
|-----|---------|------|
| デプロイ | Docker必須 | 単一バイナリ |
| メモリ使用量 | ~200MB | < 100MB |
| 起動時間 | 10-20秒 | < 1秒 |
| 設定変更 | 再起動必要 | ホットリロード可能 |
| モニタリング | 外部ツール必要 | 組み込み |

## クイックスタート（実装後）

```bash
# バイナリをダウンロード
wget https://github.com/sh03m2a5h/mcp-oidc-proxy-go/releases/download/v1.0.0/mcp-oidc-proxy

# 実行権限を付与
chmod +x mcp-oidc-proxy

# 設定ファイルで起動
./mcp-oidc-proxy --config config.yaml

# または環境変数で起動
MCP_TARGET_HOST=localhost \
MCP_TARGET_PORT=3000 \
OIDC_DISCOVERY_URL=https://your-domain.auth0.com/.well-known/openid-configuration \
OIDC_CLIENT_ID=your-client-id \
OIDC_CLIENT_SECRET=your-client-secret \
./mcp-oidc-proxy
```

## 設計レビューチェックリスト

- [x] セキュリティ要件を満たしているか（PKCE、セッション管理）
- [x] パフォーマンス目標が達成可能か
- [x] 既存実装との互換性が保たれているか
- [x] 運用性が考慮されているか（ログ、メトリクス、ヘルスチェック）
- [x] テスト戦略が明確か
- [x] デプロイメントオプションが十分か

## 貢献方法

設計に関するフィードバックや提案がある場合は、以下の方法で貢献できます：

1. Issueを作成して議論を開始
2. Pull Requestで設計ドキュメントの改善を提案
3. 実装時の経験を基にドキュメントを更新

## ライセンス

このプロジェクトはMITライセンスの下で公開されています。
