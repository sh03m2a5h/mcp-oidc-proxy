package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_DefaultConfig(t *testing.T) {
	// Clear environment variables
	clearEnvVars()
	
	// Set auth mode to bypass to avoid OIDC validation
	os.Setenv("AUTH_MODE", "bypass")
	defer os.Unsetenv("AUTH_MODE")

	cfg, err := Load("")
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Check default values
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, 30*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, "localhost", cfg.Proxy.TargetHost)
	assert.Equal(t, 3000, cfg.Proxy.TargetPort)
	assert.Equal(t, "memory", cfg.Session.Store)
	assert.Equal(t, "bypass", cfg.Auth.Mode) // We set it to bypass
	assert.Equal(t, "info", cfg.Logging.Level)
}

func TestLoad_FromFile(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	
	configContent := `
server:
  host: "127.0.0.1"
  port: 9090
  read_timeout: "10s"
proxy:
  target_host: "example.com"
  target_port: 8080
auth:
  mode: "bypass"
logging:
  level: "debug"
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load config
	cfg, err := Load(configFile)
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify loaded values
	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, 10*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, "example.com", cfg.Proxy.TargetHost)
	assert.Equal(t, 8080, cfg.Proxy.TargetPort)
	assert.Equal(t, "bypass", cfg.Auth.Mode)
	assert.Equal(t, "debug", cfg.Logging.Level)
}

func TestLoad_EnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("MCP_HOST", "192.168.1.1")
	os.Setenv("MCP_PORT", "3000")
	os.Setenv("MCP_TARGET_HOST", "backend.local")
	os.Setenv("MCP_TARGET_PORT", "5000")
	os.Setenv("AUTH_MODE", "bypass")
	os.Setenv("LOG_LEVEL", "warn")
	defer clearEnvVars()

	cfg, err := Load("")
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify environment variable overrides
	assert.Equal(t, "192.168.1.1", cfg.Server.Host)
	assert.Equal(t, 3000, cfg.Server.Port)
	assert.Equal(t, "backend.local", cfg.Proxy.TargetHost)
	assert.Equal(t, 5000, cfg.Proxy.TargetPort)
	assert.Equal(t, "bypass", cfg.Auth.Mode)
	assert.Equal(t, "warn", cfg.Logging.Level)
}

func TestLoad_LegacyAuth0Config(t *testing.T) {
	// Set legacy Auth0 environment variables
	os.Setenv("AUTH0_DOMAIN", "test.auth0.com")
	os.Setenv("AUTH0_CLIENT_ID", "legacy-client-id")
	os.Setenv("AUTH0_CLIENT_SECRET", "legacy-secret")
	os.Setenv("AUTH_MODE", "oidc")
	defer clearEnvVars()

	cfg, err := Load("")
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify legacy config conversion
	assert.Equal(t, "https://test.auth0.com/.well-known/openid-configuration", cfg.OIDC.DiscoveryURL)
	assert.Equal(t, "legacy-client-id", cfg.OIDC.ClientID)
	assert.Equal(t, "legacy-secret", cfg.OIDC.ClientSecret)
}

func TestLoad_OIDCConfigOverridesLegacy(t *testing.T) {
	// Set both OIDC and legacy Auth0 variables
	os.Setenv("OIDC_DISCOVERY_URL", "https://modern.provider.com/.well-known/openid-configuration")
	os.Setenv("OIDC_CLIENT_ID", "modern-client-id")
	os.Setenv("OIDC_CLIENT_SECRET", "modern-secret")
	os.Setenv("AUTH0_DOMAIN", "test.auth0.com")
	os.Setenv("AUTH0_CLIENT_ID", "legacy-client-id")
	os.Setenv("AUTH_MODE", "oidc")
	defer clearEnvVars()

	cfg, err := Load("")
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify OIDC config takes precedence
	assert.Equal(t, "https://modern.provider.com/.well-known/openid-configuration", cfg.OIDC.DiscoveryURL)
	assert.Equal(t, "modern-client-id", cfg.OIDC.ClientID)
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		Proxy: ProxyConfig{
			TargetHost:   "localhost",
			TargetPort:   3000,
			TargetScheme: "http",
			Retry: RetryConfig{
				MaxAttempts: 3,
				Backoff:     100 * time.Millisecond,
			},
			CircuitBreaker: CircuitBreakerConfig{
				Threshold: 5,
				Timeout:   60 * time.Second,
			},
		},
		Auth: AuthConfig{
			Mode: "bypass",
			Headers: HeadersConfig{
				UserID:     "X-User-ID",
				UserEmail:  "X-User-Email",
				UserName:   "X-User-Name",
				UserGroups: "X-User-Groups",
			},
		},
		Session: SessionConfig{
			Store:          "memory",
			TTL:            24 * time.Hour,
			CookieName:     "session",
			CookiePath:     "/",
			CookieSameSite: "lax",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
	}

	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidate_ServerConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  ServerConfig
		wantErr string
	}{
		{
			name: "invalid port",
			config: ServerConfig{
				Port:         0,
				ReadTimeout:  time.Second,
				WriteTimeout: time.Second,
				IdleTimeout:  time.Second,
			},
			wantErr: "invalid port",
		},
		{
			name: "TLS without cert",
			config: ServerConfig{
				Port:         8080,
				ReadTimeout:  time.Second,
				WriteTimeout: time.Second,
				IdleTimeout:  time.Second,
				TLS: TLSConfig{
					Enabled: true,
					KeyFile: "key.pem",
				},
			},
			wantErr: "TLS cert file is required",
		},
		{
			name: "negative timeout",
			config: ServerConfig{
				Port:         8080,
				ReadTimeout:  -1 * time.Second,
				WriteTimeout: time.Second,
				IdleTimeout:  time.Second,
			},
			wantErr: "read timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateServerConfig(&tt.config)
			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidate_OIDCConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  OIDCConfig
		wantErr string
	}{
		{
			name:    "missing discovery URL",
			config:  OIDCConfig{},
			wantErr: "discovery URL is required",
		},
		{
			name: "invalid discovery URL",
			config: OIDCConfig{
				DiscoveryURL: "not-a-url",
				ClientID:     "test",
				ClientSecret: "secret",
				Scopes:       []string{"openid"},
				RedirectURL:  "http://localhost/callback",
			},
			wantErr: "invalid discovery URL: must be a valid URL",
		},
		{
			name: "missing client ID",
			config: OIDCConfig{
				DiscoveryURL: "https://example.com/.well-known/openid-configuration",
				ClientSecret: "secret",
				Scopes:       []string{"openid"},
				RedirectURL:  "http://localhost/callback",
			},
			wantErr: "client ID is required",
		},
		{
			name: "empty scopes",
			config: OIDCConfig{
				DiscoveryURL: "https://example.com/.well-known/openid-configuration",
				ClientID:     "test",
				ClientSecret: "secret",
				Scopes:       []string{},
				RedirectURL:  "http://localhost/callback",
			},
			wantErr: "at least one scope is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOIDCConfig(&tt.config)
			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidate_SessionConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  SessionConfig
		wantErr string
	}{
		{
			name: "invalid store",
			config: SessionConfig{
				Store:          "invalid",
				TTL:            time.Hour,
				CookieName:     "session",
				CookiePath:     "/",
				CookieSameSite: "lax",
			},
			wantErr: "invalid session store",
		},
		{
			name: "redis without URL",
			config: SessionConfig{
				Store:          "redis",
				TTL:            time.Hour,
				CookieName:     "session",
				CookiePath:     "/",
				CookieSameSite: "lax",
			},
			wantErr: "redis URL is required",
		},
		{
			name: "invalid same site",
			config: SessionConfig{
				Store:          "memory",
				TTL:            time.Hour,
				CookieName:     "session",
				CookiePath:     "/",
				CookieSameSite: "invalid",
			},
			wantErr: "invalid cookie same site",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSessionConfig(&tt.config)
			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestToServerConfig(t *testing.T) {
	cfg := &ServerConfig{
		Host:         "127.0.0.1",
		Port:         9090,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 20 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	serverCfg := cfg.ToServerConfig()
	assert.NotNil(t, serverCfg)
	assert.Equal(t, cfg.Host, serverCfg.Host)
	assert.Equal(t, cfg.Port, serverCfg.Port)
	assert.Equal(t, cfg.ReadTimeout, serverCfg.ReadTimeout)
	assert.Equal(t, cfg.WriteTimeout, serverCfg.WriteTimeout)
	assert.Equal(t, cfg.IdleTimeout, serverCfg.IdleTimeout)
}

// clearEnvVars clears all test environment variables
func clearEnvVars() {
	envVars := []string{
		"MCP_HOST", "MCP_PORT",
		"MCP_TARGET_HOST", "MCP_TARGET_PORT",
		"AUTH_MODE", "LOG_LEVEL",
		"OIDC_DISCOVERY_URL", "OIDC_CLIENT_ID", "OIDC_CLIENT_SECRET",
		"AUTH0_DOMAIN", "AUTH0_CLIENT_ID", "AUTH0_CLIENT_SECRET",
		"SESSION_STORE", "REDIS_URL",
	}
	for _, env := range envVars {
		os.Unsetenv(env)
	}
}