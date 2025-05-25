package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/auth/bypass"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/auth/oidc"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/config"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/metrics"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/middleware"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/proxy"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/session"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/server"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/tracing"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/pkg/version"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// App represents the main application
type App struct {
	config         *config.Config
	logger         *zap.Logger
	server         *server.Server
	proxy          *proxy.Proxy
	oidcHandler    *oidc.Handler
	sessionStore   session.Store
	tracingShutdown func(context.Context) error
}

// New creates a new application instance
func New(configPath string) (*App, error) {
	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Setup logger
	logger, err := setupLogger(&cfg.Logging)
	if err != nil {
		return nil, fmt.Errorf("failed to setup logger: %w", err)
	}

	// Initialize tracing
	ctx := context.Background()
	tracingShutdown, err := tracing.Initialize(ctx, &cfg.Tracing)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracing: %w", err)
	}

	// Initialize metrics build info
	metrics.SetBuildInfo(version.Version, version.GitCommit, version.BuildDate)

	// Create session store
	factory := session.NewFactory(logger)
	sessionStore, err := factory.CreateStore(&cfg.Session)
	if err != nil {
		return nil, fmt.Errorf("failed to create session store: %w", err)
	}

	// Create OIDC handler only if not in bypass mode
	var oidcHandler *oidc.Handler
	if cfg.Auth.Mode != "bypass" {
		oidcHandler, err = oidc.NewHandler(ctx, &cfg.OIDC, &cfg.Session, sessionStore, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create OIDC handler: %w", err)
		}
	}

	// Create reverse proxy
	proxyConfig := &proxy.Config{
		TargetHost:     cfg.Proxy.TargetHost,
		TargetPort:     cfg.Proxy.TargetPort,
		TargetScheme:   cfg.Proxy.TargetScheme,
		Retry:          proxy.RetryConfig(cfg.Proxy.Retry),
		CircuitBreaker: proxy.CircuitBreakerConfig(cfg.Proxy.CircuitBreaker),
	}
	reverseProxy, err := proxy.New(proxyConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create reverse proxy: %w", err)
	}

	// Create HTTP server
	httpServer := server.New(cfg.Server.ToServerConfig(), logger)

	app := &App{
		config:          cfg,
		logger:          logger,
		server:          httpServer,
		proxy:           reverseProxy,
		oidcHandler:     oidcHandler,
		sessionStore:    sessionStore,
		tracingShutdown: tracingShutdown,
	}

	// Setup routes
	app.setupRoutes()

	return app, nil
}

// setupRoutes configures the application routes
func (a *App) setupRoutes() {
	router := a.server.Router()

	// Apply security headers (first for all responses)
	router.Use(middleware.SecurityHeadersMiddleware())

	// Apply tracing middleware (capture everything)
	if a.config.Tracing.Enabled {
		router.Use(middleware.TracingMiddleware(a.config.Tracing.ServiceName))
	}

	// Apply metrics middleware
	router.Use(middleware.MetricsMiddleware())

	// Apply structured logging middleware
	router.Use(middleware.StructuredLoggingMiddleware(a.logger))
	router.Use(middleware.RequestContextMiddleware())

	// Health check endpoint (public)
	router.GET("/health", a.healthHandler)

	// Metrics endpoint (public)
	if a.config.Metrics.Enabled {
		router.GET(a.config.Metrics.Path, gin.WrapH(promhttp.Handler()))
	}

	// Setup auth based on mode
	var authMiddleware gin.HandlerFunc
	
	if a.config.Auth.Mode == "bypass" {
		// Bypass mode - no login/logout routes needed
		authMiddleware = bypass.AuthMiddleware(a.logger, []string{"/health", a.config.Metrics.Path})
	} else {
		// OIDC mode - setup authentication routes
		router.GET("/login", a.oidcHandler.Authorize)
		router.GET("/callback", a.oidcHandler.Callback)
		router.POST("/logout", a.oidcHandler.Logout)
		
		authMiddleware = oidc.AuthMiddleware(a.sessionStore, a.logger, []string{"/health", "/login", "/callback", a.config.Metrics.Path})
	}
	
	// Session management route (with auth)
	router.GET("/session", authMiddleware, a.sessionHandler)
	
	// Proxy all other requests to the target (with auth)
	router.NoRoute(authMiddleware, gin.WrapH(a.proxy))
}

// Run starts the application
func (a *App) Run() error {
	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		a.logger.Info("Starting server",
			zap.String("host", a.config.Server.Host),
			zap.Int("port", a.config.Server.Port),
		)
		
		if err := a.server.Run(); err != nil && err != http.ErrServerClosed {
			serverErr <- fmt.Errorf("server failed to start: %w", err)
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErr:
		return err
	case sig := <-quit:
		a.logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
	}

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return a.shutdown(shutdownCtx)
}

// shutdown gracefully shuts down the application
func (a *App) shutdown(ctx context.Context) error {
	a.logger.Info("Shutting down application...")

	// Shutdown HTTP server
	if err := a.server.Shutdown(ctx); err != nil {
		a.logger.Error("Failed to shutdown HTTP server", zap.Error(err))
	}

	// Close session store
	if err := a.sessionStore.Close(); err != nil {
		a.logger.Error("Failed to close session store", zap.Error(err))
	}

	// Shutdown tracing
	if a.tracingShutdown != nil {
		if err := a.tracingShutdown(ctx); err != nil {
			a.logger.Error("Failed to shutdown tracing", zap.Error(err))
		}
	}

	a.logger.Info("Application shutdown complete")
	return nil
}

// healthHandler handles health check requests
func (a *App) healthHandler(c *gin.Context) {
	ctx := c.Request.Context()
	
	// Initialize health response
	health := gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   version.Version,
		"checks":    gin.H{},
	}
	
	overallHealthy := true
	
	// Check proxy target health
	proxyHealth := gin.H{"status": "healthy"}
	if err := a.proxy.Health(ctx); err != nil {
		a.logger.Warn("Proxy target health check failed", zap.Error(err))
		proxyHealth["status"] = "unhealthy"
		proxyHealth["error"] = err.Error()
		overallHealthy = false
	}
	health["checks"].(gin.H)["proxy_target"] = proxyHealth
	
	// Check session store health
	sessionHealth := gin.H{"status": "healthy"}
	if stats, err := a.sessionStore.Stats(ctx); err != nil {
		a.logger.Warn("Session store health check failed", zap.Error(err))
		sessionHealth["status"] = "unhealthy"
		sessionHealth["error"] = err.Error()
		overallHealthy = false
	} else {
		if s, ok := stats.(*session.Stats); ok {
			sessionHealth["active_sessions"] = s.ActiveSessions
			sessionHealth["store_type"] = s.StoreType
		}
	}
	health["checks"].(gin.H)["session_store"] = sessionHealth
	
	// Set overall status
	if !overallHealthy {
		health["status"] = "degraded"
		c.JSON(http.StatusServiceUnavailable, health)
		return
	}
	
	c.JSON(http.StatusOK, health)
}

// sessionHandler handles session info requests
func (a *App) sessionHandler(c *gin.Context) {
	// Get session info from context (set by auth middleware)
	userID := c.GetString("user_id")
	userEmail := c.GetString("user_email")
	userName := c.GetString("user_name")

	c.JSON(http.StatusOK, gin.H{
		"user_id":    userID,
		"user_email": userEmail,
		"user_name":  userName,
		"authenticated": userID != "",
	})
}

// setupLogger creates and configures the logger
func setupLogger(config *config.LoggingConfig) (*zap.Logger, error) {
	var zapConfig zap.Config

	switch config.Level {
	case "debug":
		zapConfig = zap.NewDevelopmentConfig()
	case "info", "warn", "error":
		zapConfig = zap.NewProductionConfig()
	default:
		zapConfig = zap.NewProductionConfig()
	}

	// Set log level
	switch config.Level {
	case "debug":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	}

	// Set encoding based on format
	if config.Format == "console" {
		zapConfig.Encoding = "console"
		zapConfig.EncoderConfig = zap.NewDevelopmentEncoderConfig()
		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		zapConfig.Encoding = "json"
		zapConfig.EncoderConfig = zap.NewProductionEncoderConfig()
		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// Add caller information for debug and development
	if config.Level == "debug" || config.Format == "console" {
		zapConfig.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
		zapConfig.Development = true
	}

	// Set output paths
	zapConfig.OutputPaths = []string{config.Output}
	zapConfig.ErrorOutputPaths = []string{config.Output}

	// Add service name and version to all logs
	zapConfig.InitialFields = map[string]interface{}{
		"service": "mcp-oidc-proxy",
		"version": version.Version,
	}

	return zapConfig.Build()
}