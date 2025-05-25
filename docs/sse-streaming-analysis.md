# SSE/WebSocket Streaming Analysis

## 問題の概要

現在の実装では、`httputil.ReverseProxy`と`ResponseRecorder`を組み合わせて使用していますが、これがSSE（Server-Sent Events）やWebSocketなどのストリーミングプロトコルで問題を引き起こしています。

## 根本原因

### 1. ResponseRecorderの使用
```go
// proxy.go:257
recorder := NewResponseRecorder()
p.reverseProxy.ServeHTTP(recorder, r)
```

- ResponseRecorderは全てのレスポンスをメモリにバッファリング
- SSEは無限ストリームのため、メモリ枯渇やタイムアウトが発生
- `net/http: abort Handler` panicが発生

### 2. リトライロジックとの不整合
- ストリーミング接続はリトライできない
- 一度開始したストリームは中断できない

## 解決策

### アプローチ1: コンテンツタイプベースの分岐
```go
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // SSE/WebSocketの検出
    if isStreamingRequest(r) {
        p.handleStreaming(w, r)
        return
    }
    
    // 通常のHTTPリクエスト
    p.handleHTTP(w, r)
}
```

### アプローチ2: カスタムResponseWriter
- ストリーミング対応のResponseWriterを実装
- Flushメソッドを適切にプロキシ
- バッファリングを無効化

### アプローチ3: 専用ストリーミングプロキシ
- SSE/WebSocket専用の軽量プロキシ実装
- リトライやサーキットブレーカーを除外
- 直接ストリーミング

## 実装計画

1. **ストリーミング検出関数の実装**
   - Accept: text/event-stream
   - Connection: Upgrade
   - Upgrade: websocket

2. **専用ハンドラーの実装**
   - handleSSE()
   - handleWebSocket()

3. **テストケースの追加**
   - SSEストリーミングテスト
   - WebSocket接続テスト

## 考慮事項

- Cloudflare Tunnelとの互換性
- メトリクスの収集方法
- エラーハンドリング
- タイムアウト設定