package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/config"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/server"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/pkg/version"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	configFile string
	host       string
	port       int
	targetHost string
	targetPort int
	authMode   string
	logLevel   string
)

var rootCmd = &cobra.Command{
	Use:   "mcp-oidc-proxy",
	Short: "MCP OIDC Proxy - OAuth 2.1/OIDC authentication proxy for MCP servers",
	Long: `MCP OIDC Proxy provides OAuth 2.1/OIDC authentication for Model Context Protocol (MCP) servers.
It supports multiple OIDC providers including Auth0, Google, Microsoft, and GitHub.`,
	Version: version.Version,
	RunE:    runServer,
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "config.yaml", "config file path")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")

	// Server flags
	rootCmd.Flags().StringVar(&host, "host", "0.0.0.0", "listen address")
	rootCmd.Flags().IntVarP(&port, "port", "p", 8080, "listen port")

	// Proxy flags
	rootCmd.Flags().StringVar(&targetHost, "target-host", "", "proxy target host")
	rootCmd.Flags().IntVar(&targetPort, "target-port", 3000, "proxy target port")

	// Auth flags
	rootCmd.Flags().StringVar(&authMode, "auth-mode", "oidc", "authentication mode (oidc, bypass)")
}

func runServer(cmd *cobra.Command, args []string) error {
	// Override environment variables with command line flags before loading config
	if cmd.Flags().Changed("host") {
		os.Setenv("MCP_HOST", host)
	}
	if cmd.Flags().Changed("port") {
		os.Setenv("MCP_PORT", fmt.Sprintf("%d", port))
	}
	if cmd.Flags().Changed("target-host") {
		os.Setenv("MCP_TARGET_HOST", targetHost)
	}
	if cmd.Flags().Changed("target-port") {
		os.Setenv("MCP_TARGET_PORT", fmt.Sprintf("%d", targetPort))
	}
	if cmd.Flags().Changed("auth-mode") {
		os.Setenv("AUTH_MODE", authMode)
	}
	if cmd.Flags().Changed("log-level") {
		os.Setenv("LOG_LEVEL", logLevel)
	}

	// Load configuration
	cfgPath := configFile
	if configFile == "config.yaml" {
		// If using default, check if file exists
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			cfgPath = "-" // Use no config file
		}
	}
	
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Setup logger
	logger, err := setupLogger(cfg.Logging.Level)
	if err != nil {
		return fmt.Errorf("failed to setup logger: %w", err)
	}
	defer logger.Sync()

	// Log configuration
	logger.Info("Configuration loaded",
		zap.String("auth_mode", cfg.Auth.Mode),
		zap.String("server_address", fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)),
		zap.String("target", fmt.Sprintf("%s://%s:%d", cfg.Proxy.TargetScheme, cfg.Proxy.TargetHost, cfg.Proxy.TargetPort)),
	)

	// Create and start server
	srv := server.New(cfg.Server.ToServerConfig(), logger)
	
	// Setup graceful shutdown
	errChan := make(chan error, 1)
	go func() {
		if err := srv.Run(); err != nil {
			errChan <- err
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	case sig := <-sigChan:
		logger.Info("Received signal", zap.String("signal", sig.String()))
		
		// Graceful shutdown with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		if err := srv.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown server: %w", err)
		}
		
		logger.Info("Server shutdown complete")
		return nil
	}
}

func setupLogger(level string) (*zap.Logger, error) {
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapLevel)
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	return config.Build()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}