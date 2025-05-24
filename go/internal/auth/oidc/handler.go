package oidc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/config"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/session"
	"go.uber.org/zap"
)

// Handler handles OIDC authentication
type Handler struct {
	client       *Client
	sessionStore session.Store
	config       *config.OIDCConfig
	logger       *zap.Logger
}

// NewHandler creates a new OIDC handler
func NewHandler(ctx context.Context, cfg *config.OIDCConfig, sessionStore session.Store, logger *zap.Logger) (*Handler, error) {
	// Validate configuration
	if cfg.DiscoveryURL == "" {
		return nil, fmt.Errorf("OIDC discovery URL is required")
	}
	if cfg.ClientID == "" {
		return nil, fmt.Errorf("OIDC client ID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, fmt.Errorf("OIDC client secret is required")
	}
	if cfg.RedirectURL == "" {
		return nil, fmt.Errorf("OIDC redirect URL is required")
	}

	// Create OIDC client
	client, err := NewClient(ctx, cfg.DiscoveryURL, cfg.ClientID, cfg.ClientSecret, cfg.RedirectURL, cfg.Scopes)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC client: %w", err)
	}

	return &Handler{
		client:       client,
		sessionStore: sessionStore,
		config:       cfg,
		logger:       logger,
	}, nil
}

// Authorize handles the authorization request
func (h *Handler) Authorize(c *gin.Context) {
	// Generate state for CSRF protection
	state, err := generateRandomString(32)
	if err != nil {
		h.logger.Error("Failed to generate state", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate state",
		})
		return
	}

	// Generate authorization URL with PKCE
	authURL, codeVerifier, _, err := h.client.AuthCodeURL(state)
	if err != nil {
		h.logger.Error("Failed to generate auth URL", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate authorization URL",
		})
		return
	}

	// Store state and PKCE verifier in session
	authSession := &AuthSession{
		State:        state,
		CodeVerifier: codeVerifier,
		CreatedAt:    time.Now(),
		RedirectURI:  c.Query("redirect_uri"),
	}

	// Create temporary session for auth flow
	sessionID, err := h.sessionStore.Create(c.Request.Context(), fmt.Sprintf("auth:%s", state), authSession, 10*time.Minute)
	if err != nil {
		h.logger.Error("Failed to create auth session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create session",
		})
		return
	}

	h.logger.Debug("Created auth session",
		zap.String("session_id", sessionID),
		zap.String("state", state),
	)

	// Redirect to authorization endpoint
	c.Redirect(http.StatusFound, authURL)
}

// Callback handles the authorization callback
func (h *Handler) Callback(c *gin.Context) {
	// Get state and code from query parameters
	state := c.Query("state")
	code := c.Query("code")
	errorParam := c.Query("error")
	errorDesc := c.Query("error_description")

	// Check for errors from authorization server
	if errorParam != "" {
		h.logger.Error("Authorization error",
			zap.String("error", errorParam),
			zap.String("description", errorDesc),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             errorParam,
			"error_description": errorDesc,
		})
		return
	}

	// Validate required parameters
	if state == "" || code == "" {
		h.logger.Error("Missing state or code parameter")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing required parameters",
		})
		return
	}

	// Retrieve auth session
	var authSession AuthSession
	sessionKey := fmt.Sprintf("auth:%s", state)
	err := h.sessionStore.Get(c.Request.Context(), sessionKey, &authSession)
	if err != nil {
		h.logger.Error("Failed to retrieve auth session",
			zap.Error(err),
			zap.String("state", state),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid or expired state",
		})
		return
	}

	// Delete auth session (one-time use)
	if err := h.sessionStore.Delete(c.Request.Context(), sessionKey); err != nil {
		h.logger.Warn("Failed to delete auth session", zap.Error(err), zap.String("key", sessionKey))
	}

	// Validate state
	if authSession.State != state {
		h.logger.Error("State mismatch",
			zap.String("expected", authSession.State),
			zap.String("received", state),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid or expired state",
		})
		return
	}

	// Exchange code for tokens
	tokenResp, err := h.client.Exchange(c.Request.Context(), code, authSession.CodeVerifier)
	if err != nil {
		h.logger.Error("Failed to exchange code for tokens", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to exchange authorization code",
		})
		return
	}

	// Extract user information from claims
	userID, _ := tokenResp.Claims["sub"].(string)
	email, _ := tokenResp.Claims["email"].(string)
	name, _ := tokenResp.Claims["name"].(string)
	
	// If email is not in ID token, try userinfo endpoint
	if email == "" && h.config.UseUserInfo {
		userInfo, err := h.client.UserInfo(c.Request.Context(), tokenResp.AccessToken)
		if err != nil {
			h.logger.Warn("Failed to fetch user info", zap.Error(err))
		} else {
			if e, ok := userInfo["email"].(string); ok {
				email = e
			}
			if n, ok := userInfo["name"].(string); ok && name == "" {
				name = n
			}
		}
	}

	// Create user session
	userSession := &UserSession{
		ID:           userID,
		Email:        email,
		Name:         name,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		IDToken:      tokenResp.IDToken,
		ExpiresAt:    tokenResp.Expiry,
		CreatedAt:    time.Now(),
		Claims:       tokenResp.Claims,
	}

	// Store user session
	sessionID, err := h.sessionStore.Create(c.Request.Context(), fmt.Sprintf("user:%s", userID), userSession, 0)
	if err != nil {
		h.logger.Error("Failed to create user session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create user session",
		})
		return
	}

	h.logger.Info("User authenticated successfully",
		zap.String("user_id", userID),
		zap.String("email", email),
		zap.String("session_id", sessionID),
	)

	// Set session cookie
	c.SetCookie(
		"session_id",
		sessionID,
		int(24*time.Hour/time.Second), // 24 hours
		"/",
		"", // Domain (empty = current domain)
		false, // Secure (set to true in production with HTTPS)
		true,  // HttpOnly
	)

	// Redirect to original URL or default
	redirectURI := authSession.RedirectURI
	if redirectURI == "" {
		redirectURI = "/"
	}
	c.Redirect(http.StatusFound, redirectURI)
}

// Logout handles user logout
func (h *Handler) Logout(c *gin.Context) {
	// Get session ID from cookie
	sessionID, err := c.Cookie("session_id")
	if err == nil && sessionID != "" {
		// Delete session from store
		if err := h.sessionStore.Delete(c.Request.Context(), sessionID); err != nil {
			h.logger.Warn("Failed to delete session", zap.Error(err), zap.String("session_id", sessionID))
		}
	}

	// Clear session cookie
	c.SetCookie(
		"session_id",
		"",
		-1, // Max age -1 = delete cookie
		"/",
		"",
		false,
		true,
	)

	// Check if OIDC provider supports end session endpoint
	if h.config.EndSessionEndpoint != "" {
		// Build logout URL
		logoutURL := fmt.Sprintf("%s?post_logout_redirect_uri=%s",
			h.config.EndSessionEndpoint,
			h.config.PostLogoutRedirectURI,
		)

		// If we have ID token, include it
		var userSession UserSession
		if err == nil && sessionID != "" {
			if err := h.sessionStore.Get(c.Request.Context(), sessionID, &userSession); err == nil && userSession.IDToken != "" {
				logoutURL += "&id_token_hint=" + userSession.IDToken
			}
		}

		c.Redirect(http.StatusFound, logoutURL)
		return
	}

	// Otherwise, redirect to post-logout URL or home
	redirectURL := h.config.PostLogoutRedirectURI
	if redirectURL == "" {
		redirectURL = "/"
	}
	c.Redirect(http.StatusFound, redirectURL)
}

// generateRandomString generates a random string of specified length
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// AuthSession represents temporary authentication session data
type AuthSession struct {
	State        string    `json:"state"`
	CodeVerifier string    `json:"code_verifier"`
	CreatedAt    time.Time `json:"created_at"`
	RedirectURI  string    `json:"redirect_uri"`
}

// UserSession represents authenticated user session data
type UserSession struct {
	ID           string                 `json:"id"`
	Email        string                 `json:"email"`
	Name         string                 `json:"name"`
	AccessToken  string                 `json:"access_token"`
	RefreshToken string                 `json:"refresh_token"`
	IDToken      string                 `json:"id_token"`
	ExpiresAt    time.Time              `json:"expires_at"`
	CreatedAt    time.Time              `json:"created_at"`
	Claims       map[string]interface{} `json:"claims"`
}