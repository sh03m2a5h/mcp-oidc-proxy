#!/bin/bash

# OpenResty コンテナ内で lua-resty-openidc と必要な依存関係をインストールするスクリプト
docker exec mcp-openresty sh -c '
apk update && \
apk add --no-cache git build-base pcre-dev openssl-dev zlib-dev luarocks && \
luarocks install lua-resty-openidc && \
luarocks install lua-resty-session && \
luarocks install lua-resty-http && \
luarocks install lua-cjson && \
echo "依存関係のインストールが完了しました"
'
