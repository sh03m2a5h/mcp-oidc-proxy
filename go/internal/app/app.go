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
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/auth/oidc"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/config"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/proxy"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/session"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/server"
	"go.uber.org/zap"
)

// App represents the main application
type App struct {
	config       *config.Config
	logger       *zap.Logger
	server       *server.Server
	proxy        *proxy.Proxy
	oidcHandler  *oidc.Handler
	sessionStore session.Store
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

	// Create session store
	factory := session.NewFactory(logger)
	sessionStore, err := factory.CreateStore(&cfg.Session)
	if err != nil {
		return nil, fmt.Errorf("failed to create session store: %w", err)
	}

	// Create OIDC handler
	ctx := context.Background()
	oidcHandler, err := oidc.NewHandler(ctx, &cfg.OIDC, sessionStore, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC handler: %w", err)
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
		config:       cfg,
		logger:       logger,
		server:       httpServer,
		proxy:        reverseProxy,
		oidcHandler:  oidcHandler,
		sessionStore: sessionStore,
	}

	// Setup routes
	app.setupRoutes()

	return app, nil
}

// setupRoutes configures the application routes
func (a *App) setupRoutes() {
	router := a.server.Router()

	// Health check endpoint (public)
	router.GET("/health", a.healthHandler)

	// OIDC authentication routes
	router.GET("/login", a.oidcHandler.Authorize)
	router.GET("/callback", a.oidcHandler.Callback)
	router.POST("/logout", a.oidcHandler.Logout)

	// Session management routes
	router.GET("/session", a.sessionHandler)

	// Apply authentication middleware to all other routes
	authMiddleware := oidc.AuthMiddleware(a.sessionStore, a.logger, []string{"/health", "/login", "/callback"})
	
	// Protected routes group
	protected := router.Group("/")
	protected.Use(authMiddleware)
	
	// Proxy all other requests to the target
	protected.Any("/*path", gin.WrapH(a.proxy))
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

	a.logger.Info("Application shutdown complete")
	return nil
}

// healthHandler handles health check requests
func (a *App) healthHandler(c *gin.Context) {
	// Check proxy target health
	if err := a.proxy.Health(c.Request.Context()); err != nil {
		a.logger.Warn("Proxy target health check failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	// Check session store health
	if _, err := a.sessionStore.Stats(c.Request.Context()); err != nil {
		a.logger.Warn("Session store health check failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"timestamp": time.Now().UTC(),
		"version": "1.0.0",
	})
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

	// Set output format
	if config.Format == "console" {
		zapConfig.Encoding = "console"
		zapConfig.EncoderConfig = zap.NewDevelopmentEncoderConfig()
	} else {
		zapConfig.Encoding = "json"
		zapConfig.EncoderConfig = zap.NewProductionEncoderConfig()
	}

	return zapConfig.Build()
}