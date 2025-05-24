package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/pkg/version"
)

// HealthResponse represents health check response
type HealthResponse struct {
	Status        string `json:"status"`
	Version       string `json:"version"`
	Uptime        int64  `json:"uptime"`
	BackendStatus string `json:"backend_status"`
}

// VersionResponse represents version information response
type VersionResponse struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

var startTime = time.Now()

// handleHealth handles health check requests
func (s *Server) handleHealth(c *gin.Context) {
	uptime := int64(time.Since(startTime).Seconds())
	
	response := HealthResponse{
		Status:        "healthy",
		Version:       version.Version,
		Uptime:        uptime,
		BackendStatus: "unknown", // TODO: Implement backend health check
	}

	c.JSON(http.StatusOK, response)
}

// handleVersion handles version information requests
func (s *Server) handleVersion(c *gin.Context) {
	buildInfo := version.GetBuildInfo()
	
	response := VersionResponse{
		Version:   buildInfo.Version,
		GitCommit: buildInfo.GitCommit,
		BuildDate: buildInfo.BuildDate,
		GoVersion: buildInfo.GoVersion,
		Platform:  buildInfo.Platform,
	}

	c.JSON(http.StatusOK, response)
}