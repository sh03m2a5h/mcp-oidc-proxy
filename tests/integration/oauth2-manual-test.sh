#!/bin/bash

# OAuth2 Manual Testing Helper Script
# このスクリプトは手動テストの準備と実行を支援します

set -e

# カラー出力の定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 設定ファイルのディレクトリ
CONFIG_DIR="$(dirname "$0")/../../go/configs"
TEST_RESULTS_DIR="$(dirname "$0")/test-results"

# テスト結果ディレクトリの作成
mkdir -p "$TEST_RESULTS_DIR"

print_header() {
    echo -e "${BLUE}================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}================================${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# 使用方法の表示
show_usage() {
    echo "Usage: $0 [OPTIONS] COMMAND"
    echo ""
    echo "Commands:"
    echo "  setup          - テスト環境のセットアップ"
    echo "  start          - プロキシサーバーの起動"
    echo "  test           - インタラクティブテストの実行"
    echo "  validate       - 設定ファイルの検証"
    echo "  health         - ヘルスチェック"
    echo "  logs           - ログの表示"
    echo "  cleanup        - テスト環境のクリーンアップ"
    echo ""
    echo "Options:"
    echo "  -p, --provider PROVIDER   OIDCプロバイダー (google, azure, auth0, keycloak)"
    echo "  -c, --config CONFIG       設定ファイルのパス"
    echo "  -v, --verbose            詳細出力"
    echo "  -h, --help               このヘルプを表示"
}

# 設定ファイルの検証
validate_config() {
    local config_file="$1"
    
    print_info "設定ファイルを検証中: $config_file"
    
    if [[ ! -f "$config_file" ]]; then
        print_error "設定ファイルが見つかりません: $config_file"
        return 1
    fi
    
    # 必須フィールドの確認
    local required_fields=(
        "auth.mode"
        "auth.oidc.discovery_url"
        "auth.oidc.client_id"
        "auth.oidc.client_secret"
        "auth.oidc.redirect_url"
    )
    
    for field in "${required_fields[@]}"; do
        if ! grep -q "$(echo "$field" | sed 's/\./\\..*:/g')" "$config_file"; then
            print_warning "フィールドが見つかりません: $field"
        fi
    done
    
    print_success "設定ファイルの検証完了"
}

# 環境変数の確認
check_environment() {
    print_info "環境変数をチェック中..."
    
    local env_vars=()
    case "$PROVIDER" in
        google)
            env_vars=("GOOGLE_CLIENT_ID" "GOOGLE_CLIENT_SECRET")
            ;;
        azure)
            env_vars=("AZURE_CLIENT_ID" "AZURE_CLIENT_SECRET" "AZURE_TENANT_ID")
            ;;
        auth0)
            env_vars=("AUTH0_CLIENT_ID" "AUTH0_CLIENT_SECRET" "AUTH0_DOMAIN")
            ;;
        keycloak)
            env_vars=("KEYCLOAK_CLIENT_ID" "KEYCLOAK_CLIENT_SECRET" "KEYCLOAK_HOST" "KEYCLOAK_REALM")
            ;;
    esac
    
    local missing_vars=()
    for var in "${env_vars[@]}"; do
        if [[ -z "${!var}" ]]; then
            missing_vars+=("$var")
        fi
    done
    
    if [[ ${#missing_vars[@]} -gt 0 ]]; then
        print_error "以下の環境変数が設定されていません:"
        for var in "${missing_vars[@]}"; do
            echo "  - $var"
        done
        return 1
    fi
    
    print_success "環境変数チェック完了"
}

# テスト環境のセットアップ
setup_test_environment() {
    print_header "テスト環境セットアップ"
    
    # Go実行可能ファイルの確認
    if ! command -v go &> /dev/null; then
        print_error "Go がインストールされていません"
        return 1
    fi
    
    # プロジェクトディレクトリの確認
    if [[ ! -f "$(dirname "$0")/../../go/go.mod" ]]; then
        print_error "Goプロジェクトが見つかりません"
        return 1
    fi
    
    # 依存関係のインストール
    print_info "依存関係をインストール中..."
    cd "$(dirname "$0")/../../go"
    go mod download
    
    # バイナリのビルド
    print_info "プロキシバイナリをビルド中..."
    make build
    
    print_success "テスト環境セットアップ完了"
}

# プロキシサーバーの起動
start_proxy() {
    print_header "プロキシサーバー起動"
    
    local config_file="$CONFIG_FILE"
    local timestamp=$(date +%Y%m%d-%H%M%S)
    local log_file="$TEST_RESULTS_DIR/proxy-${PROVIDER}-${timestamp}.log"
    
    print_info "設定ファイル: $config_file"
    print_info "ログファイル: $log_file"
    
    # 既存のプロセスを確認
    if pgrep -f "mcp-oidc-proxy" > /dev/null; then
        print_warning "既存のプロキシプロセスが実行中です"
        read -p "停止しますか? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            pkill -f "mcp-oidc-proxy"
            sleep 2
        else
            return 1
        fi
    fi
    
    # プロキシの起動
    cd "$(dirname "$0")/../../go"
    export CONFIG_FILE="$config_file"
    
    print_info "プロキシを起動中..."
    ./bin/mcp-oidc-proxy 2>&1 | tee "$log_file" &
    local proxy_pid=$!
    
    # 起動確認
    sleep 3
    if ! kill -0 $proxy_pid 2>/dev/null; then
        print_error "プロキシの起動に失敗しました"
        return 1
    fi
    
    # ヘルスチェック
    local health_check_count=0
    while [[ $health_check_count -lt 10 ]]; do
        if curl -s http://localhost:8080/health > /dev/null; then
            print_success "プロキシが正常に起動しました (PID: $proxy_pid)"
            echo $proxy_pid > "$TEST_RESULTS_DIR/proxy.pid"
            return 0
        fi
        sleep 1
        ((health_check_count++))
    done
    
    print_error "プロキシのヘルスチェックに失敗しました"
    return 1
}

# インタラクティブテストの実行
run_interactive_test() {
    print_header "インタラクティブテスト実行"
    
    echo "以下のテストシナリオを実行してください:"
    echo ""
    echo "1. 基本認証フロー"
    echo "   - ブラウザで http://localhost:8080/ にアクセス"
    echo "   - OIDCプロバイダーでログイン"
    echo "   - バックエンドアクセス確認"
    echo ""
    echo "2. セッション継続性"
    echo "   - 複数回のリクエスト送信"
    echo "   - 再認証が不要なことを確認"
    echo ""
    echo "3. ログアウト機能"
    echo "   - POST http://localhost:8080/logout"
    echo "   - セッション削除を確認"
    echo ""
    
    read -p "テストを開始しますか? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        return 0
    fi
    
    # ブラウザを開く
    local url="http://localhost:8080/"
    if command -v xdg-open &> /dev/null; then
        xdg-open "$url"
    elif command -v open &> /dev/null; then
        open "$url"
    else
        print_info "ブラウザで以下のURLにアクセスしてください: $url"
    fi
    
    # テスト結果の記録
    local test_result_file="$TEST_RESULTS_DIR/test-result-$(date +%Y%m%d-%H%M%S).md"
    
    cat > "$test_result_file" << EOF
# OAuth2テスト結果

## テスト環境
- プロバイダー: $PROVIDER
- 実行日時: $(date)
- 設定ファイル: $CONFIG_FILE

## テスト結果

### 基本認証フロー
- [ ] リダイレクト正常
- [ ] 認証成功
- [ ] セッション作成
- [ ] バックエンドアクセス

### セッション管理
- [ ] セッション継続
- [ ] ログアウト機能

### エラーハンドリング
- [ ] 無効認証の処理
- [ ] タイムアウト処理

## 備考

EOF
    
    print_info "テスト結果ファイル: $test_result_file"
    print_info "テスト完了後、結果ファイルを更新してください"
}

# ヘルスチェック
health_check() {
    print_header "ヘルスチェック"
    
    # プロキシの稼働確認
    if curl -s http://localhost:8080/health > /dev/null; then
        print_success "プロキシサーバーは正常に稼働中"
        
        # 詳細情報の取得
        local health_info=$(curl -s http://localhost:8080/health | jq . 2>/dev/null || curl -s http://localhost:8080/health)
        echo "ヘルス情報:"
        echo "$health_info"
    else
        print_error "プロキシサーバーにアクセスできません"
        return 1
    fi
    
    # プロセス確認
    if [[ -f "$TEST_RESULTS_DIR/proxy.pid" ]]; then
        local pid=$(cat "$TEST_RESULTS_DIR/proxy.pid")
        if kill -0 $pid 2>/dev/null; then
            print_success "プロキシプロセス稼働中 (PID: $pid)"
        else
            print_error "プロキシプロセスが見つかりません"
        fi
    fi
}

# ログの表示
show_logs() {
    print_header "ログ表示"
    
    local latest_log=$(ls -t "$TEST_RESULTS_DIR"/proxy-*.log 2>/dev/null | head -1)
    if [[ -n "$latest_log" ]]; then
        print_info "最新のログファイル: $latest_log"
        tail -f "$latest_log"
    else
        print_error "ログファイルが見つかりません"
        return 1
    fi
}

# クリーンアップ
cleanup() {
    print_header "クリーンアップ"
    
    # プロキシプロセスの停止
    if [[ -f "$TEST_RESULTS_DIR/proxy.pid" ]]; then
        local pid=$(cat "$TEST_RESULTS_DIR/proxy.pid")
        if kill -0 $pid 2>/dev/null; then
            print_info "プロキシプロセスを停止中 (PID: $pid)"
            kill $pid
            sleep 2
            if kill -0 $pid 2>/dev/null; then
                kill -9 $pid
            fi
        fi
        rm -f "$TEST_RESULTS_DIR/proxy.pid"
    fi
    
    # その他のプロセスの確認
    if pgrep -f "mcp-oidc-proxy" > /dev/null; then
        print_warning "他のプロキシプロセスが残っています"
        pkill -f "mcp-oidc-proxy"
    fi
    
    print_success "クリーンアップ完了"
}

# コマンドライン引数の解析
PROVIDER="google"
CONFIG_FILE=""
VERBOSE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--provider)
            PROVIDER="$2"
            shift 2
            ;;
        -c|--config)
            CONFIG_FILE="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            COMMAND="$1"
            shift
            ;;
    esac
done

# デフォルト設定ファイルの設定
if [[ -z "$CONFIG_FILE" ]]; then
    CONFIG_FILE="$CONFIG_DIR/config-${PROVIDER}.yaml"
fi

# Verboseモードの設定
if [[ "$VERBOSE" == "true" ]]; then
    set -x
fi

# コマンドの実行
case "$COMMAND" in
    setup)
        setup_test_environment
        ;;
    start)
        check_environment
        validate_config "$CONFIG_FILE"
        start_proxy
        ;;
    test)
        run_interactive_test
        ;;
    validate)
        validate_config "$CONFIG_FILE"
        ;;
    health)
        health_check
        ;;
    logs)
        show_logs
        ;;
    cleanup)
        cleanup
        ;;
    *)
        show_usage
        exit 1
        ;;
esac