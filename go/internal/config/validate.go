package config

import (
	"fmt"
	"net/url"
	"strings"
)

// Validate validates the configuration
func Validate(config *Config) error {
	// Validate server config
	if err := validateServerConfig(&config.Server); err != nil {
		return fmt.Errorf("server config: %w", err)
	}

	// Validate proxy config
	if err := validateProxyConfig(&config.Proxy); err != nil {
		return fmt.Errorf("proxy config: %w", err)
	}

	// Validate auth config
	if err := validateAuthConfig(&config.Auth); err != nil {
		return fmt.Errorf("auth config: %w", err)
	}

	// Validate OIDC config if auth mode is oidc
	if config.Auth.Mode == "oidc" {
		if err := validateOIDCConfig(&config.OIDC); err != nil {
			return fmt.Errorf("oidc config: %w", err)
		}
	}

	// Validate session config
	if err := validateSessionConfig(&config.Session); err != nil {
		return fmt.Errorf("session config: %w", err)
	}

	// Validate logging config
	if err := validateLoggingConfig(&config.Logging); err != nil {
		return fmt.Errorf("logging config: %w", err)
	}

	// Validate tracing config if enabled
	if config.Tracing.Enabled {
		if err := validateTracingConfig(&config.Tracing); err != nil {
			return fmt.Errorf("tracing config: %w", err)
		}
	}

	return nil
}

func validateServerConfig(config *ServerConfig) error {
	if config.Port < 1 || config.Port > 65535 {
		return fmt.Errorf("invalid port: %d", config.Port)
	}

	if config.TLS.Enabled {
		if config.TLS.CertFile == "" {
			return fmt.Errorf("TLS cert file is required when TLS is enabled")
		}
		if config.TLS.KeyFile == "" {
			return fmt.Errorf("TLS key file is required when TLS is enabled")
		}
	}

	if config.ReadTimeout <= 0 {
		return fmt.Errorf("read timeout must be positive")
	}
	if config.WriteTimeout <= 0 {
		return fmt.Errorf("write timeout must be positive")
	}
	if config.IdleTimeout <= 0 {
		return fmt.Errorf("idle timeout must be positive")
	}

	return nil
}

func validateProxyConfig(config *ProxyConfig) error {
	if config.TargetHost == "" {
		return fmt.Errorf("target host is required")
	}

	if config.TargetPort < 1 || config.TargetPort > 65535 {
		return fmt.Errorf("invalid target port: %d", config.TargetPort)
	}

	if config.TargetScheme != "http" && config.TargetScheme != "https" {
		return fmt.Errorf("target scheme must be http or https")
	}

	if config.Retry.MaxAttempts < 0 {
		return fmt.Errorf("retry max attempts must be non-negative")
	}
	if config.Retry.Backoff < 0 {
		return fmt.Errorf("retry backoff must be non-negative")
	}

	if config.CircuitBreaker.Threshold < 0 {
		return fmt.Errorf("circuit breaker threshold must be non-negative")
	}
	if config.CircuitBreaker.Timeout < 0 {
		return fmt.Errorf("circuit breaker timeout must be non-negative")
	}

	return nil
}

func validateAuthConfig(config *AuthConfig) error {
	switch config.Mode {
	case "oidc", "bypass":
		// Valid modes
	default:
		return fmt.Errorf("invalid auth mode: %s (must be 'oidc' or 'bypass')", config.Mode)
	}

	// Validate header names
	if config.Headers.UserID == "" {
		return fmt.Errorf("user ID header name is required")
	}
	if config.Headers.UserEmail == "" {
		return fmt.Errorf("user email header name is required")
	}
	if config.Headers.UserName == "" {
		return fmt.Errorf("user name header name is required")
	}
	if config.Headers.UserGroups == "" {
		return fmt.Errorf("user groups header name is required")
	}

	return nil
}

func validateOIDCConfig(config *OIDCConfig) error {
	if config.DiscoveryURL == "" {
		return fmt.Errorf("discovery URL is required")
	}

	// Validate discovery URL
	parsedURL, err := url.Parse(config.DiscoveryURL)
	if err != nil {
		return fmt.Errorf("invalid discovery URL: %w", err)
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fmt.Errorf("invalid discovery URL: must be a valid URL with scheme and host")
	}

	if config.ClientID == "" {
		return fmt.Errorf("client ID is required")
	}

	if config.ClientSecret == "" {
		return fmt.Errorf("client secret is required")
	}

	if len(config.Scopes) == 0 {
		return fmt.Errorf("at least one scope is required")
	}

	// Validate redirect URLs
	if config.RedirectURL == "" {
		return fmt.Errorf("redirect URL is required")
	}
	if _, err := url.Parse(config.RedirectURL); err != nil {
		return fmt.Errorf("invalid redirect URL: %w", err)
	}

	if config.PostLogoutRedirectURL != "" {
		if _, err := url.Parse(config.PostLogoutRedirectURL); err != nil {
			return fmt.Errorf("invalid post logout redirect URL: %w", err)
		}
	}

	return nil
}

func validateSessionConfig(config *SessionConfig) error {
	switch config.Store {
	case "memory", "redis":
		// Valid stores
	default:
		return fmt.Errorf("invalid session store: %s (must be 'memory' or 'redis')", config.Store)
	}

	if config.TTL <= 0 {
		return fmt.Errorf("session TTL must be positive")
	}

	if config.CookieName == "" {
		return fmt.Errorf("cookie name is required")
	}

	if config.CookiePath == "" {
		return fmt.Errorf("cookie path is required")
	}

	switch strings.ToLower(config.CookieSameSite) {
	case "strict", "lax", "none":
		// Valid values
	default:
		return fmt.Errorf("invalid cookie same site: %s (must be 'strict', 'lax', or 'none')", config.CookieSameSite)
	}

	// Validate Redis config if using Redis store
	if config.Store == "redis" {
		if config.Redis.URL == "" {
			return fmt.Errorf("redis URL is required when using redis store")
		}
		if _, err := url.Parse(config.Redis.URL); err != nil {
			return fmt.Errorf("invalid redis URL: %w", err)
		}
		if config.Redis.DB < 0 {
			return fmt.Errorf("redis DB must be non-negative")
		}
	}

	return nil
}

func validateLoggingConfig(config *LoggingConfig) error {
	switch strings.ToLower(config.Level) {
	case "debug", "info", "warn", "error":
		// Valid levels
	default:
		return fmt.Errorf("invalid log level: %s (must be 'debug', 'info', 'warn', or 'error')", config.Level)
	}

	switch strings.ToLower(config.Format) {
	case "json", "text":
		// Valid formats
	default:
		return fmt.Errorf("invalid log format: %s (must be 'json' or 'text')", config.Format)
	}

	switch strings.ToLower(config.Output) {
	case "stdout", "stderr", "file":
		// Valid outputs
	default:
		return fmt.Errorf("invalid log output: %s (must be 'stdout', 'stderr', or 'file')", config.Output)
	}

	// Validate file config if output is file
	if strings.ToLower(config.Output) == "file" {
		if config.File.Path == "" {
			return fmt.Errorf("log file path is required when output is 'file'")
		}
		if config.File.MaxBackups < 0 {
			return fmt.Errorf("log file max backups must be non-negative")
		}
	}

	return nil
}

func validateTracingConfig(config *TracingConfig) error {
	switch strings.ToLower(config.Provider) {
	case "jaeger", "zipkin":
		// Valid providers
	default:
		return fmt.Errorf("invalid tracing provider: %s (must be 'jaeger' or 'zipkin')", config.Provider)
	}

	if config.Endpoint == "" {
		return fmt.Errorf("tracing endpoint is required when tracing is enabled")
	}
	if _, err := url.Parse(config.Endpoint); err != nil {
		return fmt.Errorf("invalid tracing endpoint: %w", err)
	}

	if config.ServiceName == "" {
		return fmt.Errorf("service name is required when tracing is enabled")
	}

	if config.SampleRate < 0 || config.SampleRate > 1 {
		return fmt.Errorf("sample rate must be between 0 and 1")
	}

	return nil
}