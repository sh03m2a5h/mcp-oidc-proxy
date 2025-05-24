package main

import (
	"fmt"
	"os"

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
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Implement server startup
		fmt.Println("Starting MCP OIDC Proxy...")
		fmt.Printf("Version: %s\n", version.Version)
		fmt.Printf("Config: %s\n", configFile)
		fmt.Printf("Listen: %s:%d\n", host, port)
	},
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

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}