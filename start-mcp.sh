#!/bin/bash

# .envファイルのチェック
if [ ! -f .env ]; then
    echo ".envファイルが見つかりません。"
    if [ -f .env.sample ]; then
        echo ".env.sampleをコピーして.envを作成します..."
        cp .env.sample .env
        echo ".envファイルを作成しました。必要に応じて設定を更新してください。"
    else
        echo "エラー: .env.sampleファイルも見つかりません。"
        exit 1
    fi
fi

# .envファイルを読み込み
export $(grep -v '^#' .env | xargs)

# MCP認証プロキシの起動
echo "MCP認証プロキシを起動します..."
docker compose up -d

# コンテナの起動を待つ
echo "コンテナの起動を待っています..."
for i in {1..30}; do
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        echo "ヘルスチェック成功！"
        break
    fi
    echo -n "."
    sleep 1
done
echo ""

echo "======================="
echo "MCP認証プロキシが起動しました！"
echo "プロキシ: http://localhost:8080"
echo "ヘルスチェック: http://localhost:8080/health"
echo ""
echo "現在の設定:"
echo "- 認証モード: ${AUTH_MODE:-bypass}"
echo "- MCPサーバー: ${MCP_TARGET_HOST:-localhost}:${MCP_TARGET_PORT:-3000}"
if [ "${AUTH_MODE}" = "oidc" ]; then
    if [ -n "${OIDC_DISCOVERY_URL}" ]; then
        echo "- OIDC Provider: ${OIDC_DISCOVERY_URL}"
    elif [ -n "${AUTH0_DOMAIN}" ]; then
        echo "- Auth0 Domain: ${AUTH0_DOMAIN}"
    fi
fi
echo "======================="