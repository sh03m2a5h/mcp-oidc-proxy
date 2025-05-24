package main

import (
	"fmt"
	"os"

	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/app"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/pkg/version"
	"github.com/spf13/cobra"
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

	// Determine config file path
	cfgPath := configFile
	if configFile == "config.yaml" {
		// If using default, check if file exists
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			cfgPath = "-" // Use no config file
		}
	}

	// Create and run application
	application, err := app.New(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to create application: %w", err)
	}

	return application.Run()
}


func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}