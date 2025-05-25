-- Generic OIDC Integration Module for MCP
-- Provides helper functions for OAuth 2.1/OIDC integration
-- Supports Auth0, Google, Microsoft, GitHub, and other OIDC providers

local _M = {}
local cjson = require("cjson")
local http = require("resty.http")

-- OIDC設定の取得
function _M.get_oidc_config()
    local config = {
        discovery_url = os.getenv("OIDC_DISCOVERY_URL"),
        client_id = os.getenv("OIDC_CLIENT_ID"),
        client_secret = os.getenv("OIDC_CLIENT_SECRET"),
        scope = os.getenv("OIDC_SCOPE") or "openid email profile",
        use_pkce = (os.getenv("OIDC_USE_PKCE") or "true") == "true",
        -- 以下はレガシー対応（Auth0互換）
        auth0_domain = os.getenv("AUTH0_DOMAIN"),
        auth0_client_id = os.getenv("AUTH0_CLIENT_ID"),
        auth0_client_secret = os.getenv("AUTH0_CLIENT_SECRET")
    }
    
    -- Auth0レガシー設定からの変換
    if config.auth0_domain and not config.discovery_url then
        config.discovery_url = "https://" .. config.auth0_domain .. "/.well-known/openid-configuration"
        config.client_id = config.auth0_client_id
        config.client_secret = config.auth0_client_secret
        ngx.log(ngx.INFO, "Using Auth0 legacy configuration")
    end
    
    if not config.discovery_url or not config.client_id or not config.client_secret then
        ngx.log(ngx.ERR, "OIDC configuration is incomplete. Required: OIDC_DISCOVERY_URL, OIDC_CLIENT_ID, OIDC_CLIENT_SECRET")
        return nil
    end
    
    return config
end

-- OpenID設定オブジェクトの構築
function _M.build_oidc_opts(redirect_uri)
    local oidc_config = _M.get_oidc_config()
    if not oidc_config then
        return nil
    end
    
    return {
        discovery = oidc_config.discovery_url,
        client_id = oidc_config.client_id,
        client_secret = oidc_config.client_secret,
        redirect_uri = redirect_uri or "http://localhost:8080/callback",
        redirect_uri_scheme = "http",
        scope = oidc_config.scope,
        use_pkce = oidc_config.use_pkce,
        discovery_expires_in = 86400,
        token_endpoint_auth_method = "client_secret_basic",
        refresh_session_interval = 600,
        log_level = "debug",
        ssl_verify = "no"
    }
end

-- セッション設定の取得
function _M.get_session_opts()
    return {
        name = "mcp_session",
        secret = "change_me_to_a_random_string_32_bytes_long",
        cookie = {
            secure = false,  -- HTTPでも動作するようにfalseに変更
            httponly = true,
            samesite = "Lax",
            path = "/",
            lifetime = 3600 * 8 -- 8時間
        }
    }
end

-- ユーザープロファイル情報をMCPヘッダーに設定
function _M.set_user_headers(res)
    if not res or not res.id_token then
        return
    end
    
    -- 基本的なユーザー識別子
    ngx.req.set_header("X-USER", res.id_token.sub)
    
    -- メールアドレス
    if res.id_token.email then
        ngx.req.set_header("X-EMAIL", res.id_token.email)
    end
    
    -- 名前情報（複数のクレームを試行）
    local name = res.id_token.name or res.id_token.given_name or res.id_token.preferred_username
    if name then
        ngx.req.set_header("X-NAME", name)
    end
    
    -- ロール情報（プロバイダー固有のクレームを試行）
    local roles = res.id_token.roles or res.id_token.groups or res.id_token["https://example.com/roles"]
    if roles then
        local roles_str
        if type(roles) == "table" then
            roles_str = table.concat(roles, ",")
        else
            roles_str = tostring(roles)
        end
        ngx.req.set_header("X-ROLES", roles_str)
    end
    
    -- Bearer認証ヘッダー
    if res.access_token then
        ngx.req.set_header("Authorization", "Bearer " .. res.access_token)
    end
    
    -- プロバイダー情報（デバッグ用）
    local issuer = res.id_token.iss
    if issuer then
        ngx.req.set_header("X-ISSUER", issuer)
    end
end

-- OIDCメタデータを取得する
function _M.fetch_oidc_metadata()
    local oidc_config = _M.get_oidc_config()
    if not oidc_config then
        return nil, "OIDC configuration not found"
    end
    
    local httpc = http.new()
    local res, err = httpc:request_uri(oidc_config.discovery_url, {
        method = "GET",
        ssl_verify = true,
        timeout = 10000
    })
    
    if not res then
        return nil, "Failed to fetch OIDC metadata: " .. (err or "unknown error")
    end
    
    if res.status ~= 200 then
        return nil, "Failed to fetch OIDC metadata, status: " .. res.status
    end
    
    local metadata, decode_err = cjson.decode(res.body)
    if not metadata then
        return nil, "Failed to decode OIDC metadata: " .. (decode_err or "unknown error")
    end
    
    return metadata
end

-- プロバイダー別の設定ヘルパー
_M.providers = {
    auth0 = {
        discovery_template = "https://{domain}/.well-known/openid-configuration",
        scope = "openid email profile",
        note = "Replace {domain} with your Auth0 domain"
    },
    google = {
        discovery_url = "https://accounts.google.com/.well-known/openid-configuration",
        scope = "openid email profile",
        note = "Get Client ID/Secret from Google Cloud Console"
    },
    microsoft = {
        discovery_template = "https://login.microsoftonline.com/{tenant}/v2.0/.well-known/openid-configuration",
        scope = "openid email profile",
        note = "Replace {tenant} with your Azure AD tenant ID"
    },
    github = {
        discovery_url = "https://token.actions.githubusercontent.com/.well-known/openid-configuration",
        scope = "openid email profile",
        note = "GitHub OIDC is primarily for Actions, use OAuth for general auth"
    }
}

return _M