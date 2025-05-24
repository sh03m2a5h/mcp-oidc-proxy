package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/server"
	"github.com/spf13/viper"
)

// Config represents the complete application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Proxy    ProxyConfig    `mapstructure:"proxy"`
	OIDC     OIDCConfig     `mapstructure:"oidc"`
	Session  SessionConfig  `mapstructure:"session"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
	Tracing  TracingConfig  `mapstructure:"tracing"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	TLS          TLSConfig     `mapstructure:"tls"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

// ProxyConfig holds reverse proxy configuration
type ProxyConfig struct {
	TargetHost      string              `mapstructure:"target_host"`
	TargetPort      int                 `mapstructure:"target_port"`
	TargetScheme    string              `mapstructure:"target_scheme"`
	Retry           RetryConfig         `mapstructure:"retry"`
	CircuitBreaker  CircuitBreakerConfig `mapstructure:"circuit_breaker"`
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxAttempts int           `mapstructure:"max_attempts"`
	Backoff     time.Duration `mapstructure:"backoff"`
}

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	Threshold int           `mapstructure:"threshold"`
	Timeout   time.Duration `mapstructure:"timeout"`
}

// OIDCConfig holds OIDC provider configuration
type OIDCConfig struct {
	DiscoveryURL           string   `mapstructure:"discovery_url"`
	ClientID               string   `mapstructure:"client_id"`
	ClientSecret           string   `mapstructure:"client_secret"`
	Scopes                 []string `mapstructure:"scopes"`
	UsePKCE                bool     `mapstructure:"use_pkce"`
	RedirectURL            string   `mapstructure:"redirect_url"`
	PostLogoutRedirectURL  string   `mapstructure:"post_logout_redirect_url"`
	EndSessionEndpoint     string   `mapstructure:"end_session_endpoint"`
	PostLogoutRedirectURI  string   `mapstructure:"post_logout_redirect_uri"`
	UseUserInfo            bool     `mapstructure:"use_userinfo"`
}

// SessionConfig holds session management configuration
type SessionConfig struct {
	Store        string        `mapstructure:"store"`
	TTL          time.Duration `mapstructure:"ttl"`
	CookieName   string        `mapstructure:"cookie_name"`
	CookieDomain string        `mapstructure:"cookie_domain"`
	CookiePath   string        `mapstructure:"cookie_path"`
	CookieSecure bool          `mapstructure:"cookie_secure"`
	CookieHTTPOnly bool        `mapstructure:"cookie_http_only"`
	CookieSameSite string      `mapstructure:"cookie_same_site"`
	Redis        RedisConfig   `mapstructure:"redis"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	URL       string `mapstructure:"url"`
	Password  string `mapstructure:"password"`
	DB        int    `mapstructure:"db"`
	KeyPrefix string `mapstructure:"key_prefix"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Mode          string              `mapstructure:"mode"`
	Headers       HeadersConfig       `mapstructure:"headers"`
	AccessControl AccessControlConfig `mapstructure:"access_control"`
}

// HeadersConfig holds header configuration
type HeadersConfig struct {
	UserID     string `mapstructure:"user_id"`
	UserEmail  string `mapstructure:"user_email"`
	UserName   string `mapstructure:"user_name"`
	UserGroups string `mapstructure:"user_groups"`
}

// AccessControlConfig holds access control configuration
type AccessControlConfig struct {
	PublicPaths    []string `mapstructure:"public_paths"`
	RequiredGroups []string `mapstructure:"required_groups"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string          `mapstructure:"level"`
	Format string          `mapstructure:"format"`
	Output string          `mapstructure:"output"`
	File   FileLogConfig   `mapstructure:"file"`
}

// FileLogConfig holds file logging configuration
type FileLogConfig struct {
	Path       string `mapstructure:"path"`
	MaxSize    string `mapstructure:"max_size"`
	MaxAge     string `mapstructure:"max_age"`
	MaxBackups int    `mapstructure:"max_backups"`
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
}

// TracingConfig holds tracing configuration
type TracingConfig struct {
	Enabled     bool    `mapstructure:"enabled"`
	Provider    string  `mapstructure:"provider"`
	Endpoint    string  `mapstructure:"endpoint"`
	ServiceName string  `mapstructure:"service_name"`
	SampleRate  float64 `mapstructure:"sample_rate"`
}

// Load loads configuration from file, environment variables, and command line flags
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set config file
	if configPath != "" && configPath != "-" {
		v.SetConfigFile(configPath)
	} else if configPath == "" {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("/etc/mcp-proxy")
	}

	// Set defaults
	setDefaults(v)

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Bind environment variables
	v.SetEnvPrefix("MCP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Manual environment variable mappings for backward compatibility
	bindEnvVars(v)

	// Unmarshal config
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Apply legacy Auth0 environment variables if OIDC not configured
	applyLegacyAuth0Config(&config)

	// Validate config
	if err := Validate(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.idle_timeout", "120s")
	v.SetDefault("server.tls.enabled", false)

	// Proxy defaults
	v.SetDefault("proxy.target_host", "localhost")
	v.SetDefault("proxy.target_port", 3000)
	v.SetDefault("proxy.target_scheme", "http")
	v.SetDefault("proxy.retry.max_attempts", 3)
	v.SetDefault("proxy.retry.backoff", "100ms")
	v.SetDefault("proxy.circuit_breaker.threshold", 5)
	v.SetDefault("proxy.circuit_breaker.timeout", "60s")

	// OIDC defaults
	v.SetDefault("oidc.scopes", []string{"openid", "email", "profile"})
	v.SetDefault("oidc.use_pkce", true)
	v.SetDefault("oidc.redirect_url", "http://localhost:8080/callback")
	v.SetDefault("oidc.post_logout_redirect_url", "http://localhost:8080/")

	// Session defaults
	v.SetDefault("session.store", "memory")
	v.SetDefault("session.ttl", "24h")
	v.SetDefault("session.cookie_name", "mcp_session")
	v.SetDefault("session.cookie_path", "/")
	v.SetDefault("session.cookie_secure", false)
	v.SetDefault("session.cookie_http_only", true)
	v.SetDefault("session.cookie_same_site", "lax")
	v.SetDefault("session.redis.url", "redis://localhost:6379")
	v.SetDefault("session.redis.db", 0)
	v.SetDefault("session.redis.key_prefix", "mcp:session:")

	// Auth defaults
	v.SetDefault("auth.mode", "oidc")
	v.SetDefault("auth.headers.user_id", "X-User-ID")
	v.SetDefault("auth.headers.user_email", "X-User-Email")
	v.SetDefault("auth.headers.user_name", "X-User-Name")
	v.SetDefault("auth.headers.user_groups", "X-User-Groups")
	v.SetDefault("auth.access_control.public_paths", []string{"/health", "/metrics"})

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.output", "stdout")

	// Metrics defaults
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.path", "/metrics")

	// Tracing defaults
	v.SetDefault("tracing.enabled", false)
	v.SetDefault("tracing.provider", "jaeger")
	v.SetDefault("tracing.service_name", "mcp-oidc-proxy")
	v.SetDefault("tracing.sample_rate", 0.1)
}

// bindEnvVars manually binds environment variables for better control
func bindEnvVars(v *viper.Viper) {
	// Server bindings
	v.BindEnv("server.host", "MCP_HOST")
	v.BindEnv("server.port", "MCP_PORT")

	// Proxy bindings
	v.BindEnv("proxy.target_host", "MCP_TARGET_HOST")
	v.BindEnv("proxy.target_port", "MCP_TARGET_PORT")

	// Auth bindings
	v.BindEnv("auth.mode", "AUTH_MODE")

	// OIDC bindings
	v.BindEnv("oidc.discovery_url", "OIDC_DISCOVERY_URL")
	v.BindEnv("oidc.client_id", "OIDC_CLIENT_ID")
	v.BindEnv("oidc.client_secret", "OIDC_CLIENT_SECRET")
	v.BindEnv("oidc.scope", "OIDC_SCOPE")
	v.BindEnv("oidc.use_pkce", "OIDC_USE_PKCE")

	// Session bindings
	v.BindEnv("session.store", "SESSION_STORE")
	v.BindEnv("session.redis.url", "REDIS_URL")

	// Logging bindings
	v.BindEnv("logging.level", "LOG_LEVEL")
}

// applyLegacyAuth0Config applies legacy Auth0 environment variables
func applyLegacyAuth0Config(config *Config) {
	// Check environment directly since viper might not have these bound
	auth0Domain := os.Getenv("AUTH0_DOMAIN")
	auth0ClientID := os.Getenv("AUTH0_CLIENT_ID")
	auth0ClientSecret := os.Getenv("AUTH0_CLIENT_SECRET")

	// If Auth0 legacy vars are set and OIDC is not configured
	if auth0Domain != "" && config.OIDC.DiscoveryURL == "" {
		config.OIDC.DiscoveryURL = fmt.Sprintf("https://%s/.well-known/openid-configuration", auth0Domain)
		if auth0ClientID != "" {
			config.OIDC.ClientID = auth0ClientID
		}
		if auth0ClientSecret != "" {
			config.OIDC.ClientSecret = auth0ClientSecret
		}
	}
}

// ToServerConfig converts ServerConfig to internal server.Config
func (c *ServerConfig) ToServerConfig() *server.Config {
	return &server.Config{
		Host:         c.Host,
		Port:         c.Port,
		ReadTimeout:  c.ReadTimeout,
		WriteTimeout: c.WriteTimeout,
		IdleTimeout:  c.IdleTimeout,
	}
}

