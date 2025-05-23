#!/bin/bash

# MCP認証プロキシの起動
echo "MCP認証プロキシを起動します..."
docker compose up -d

# 依存関係のインストール
echo "必要な依存関係をインストールします..."
sleep 5  # コンテナが完全に起動するのを待つ
./install-deps.sh

echo "======================="
echo "MCP認証プロキシが起動しました！"
echo "プロキシ: http://localhost:8080"
echo "ヘルスチェック: http://localhost:8080/health"
echo ""
echo "接続先MCPサーバーを設定してください:"
echo "MCP_TARGET_HOST=your-mcp-host MCP_TARGET_PORT=3000 docker compose up -d"
echo "======================="