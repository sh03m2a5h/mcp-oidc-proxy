package oidc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Client represents an OIDC client with PKCE support
type Client struct {
	provider     *oidc.Provider
	oauth2Config *oauth2.Config
	verifier     *oidc.IDTokenVerifier
	httpClient   *http.Client
}

// NewClient creates a new OIDC client with discovery support
func NewClient(ctx context.Context, discoveryURL, clientID, clientSecret, redirectURL string, scopes []string) (*Client, error) {
	// Create HTTP client with reasonable timeouts
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create custom context with HTTP client
	ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)

	// Discover OIDC provider configuration
	provider, err := oidc.NewProvider(ctx, discoveryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to discover OIDC provider: %w", err)
	}

	// Configure OAuth2 client
	oauth2Config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  redirectURL,
		Scopes:       scopes,
	}

	// Create ID token verifier
	verifier := provider.Verifier(&oidc.Config{
		ClientID: clientID,
	})

	return &Client{
		provider:     provider,
		oauth2Config: oauth2Config,
		verifier:     verifier,
		httpClient:   httpClient,
	}, nil
}

// AuthCodeURL generates the authorization URL with PKCE parameters
func (c *Client) AuthCodeURL(state string) (string, string, string, error) {
	// Generate PKCE code verifier
	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate code verifier: %w", err)
	}

	// Generate code challenge
	codeChallenge := generateCodeChallenge(codeVerifier)

	// Build authorization URL with PKCE parameters
	authURL := c.oauth2Config.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	return authURL, codeVerifier, codeChallenge, nil
}

// Exchange exchanges the authorization code for tokens using PKCE
func (c *Client) Exchange(ctx context.Context, code, codeVerifier string) (*TokenResponse, error) {
	// Exchange code for token with PKCE verifier
	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.httpClient)
	token, err := c.oauth2Config.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Extract ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no id_token in token response")
	}

	// Verify ID token
	idToken, err := c.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	// Extract claims
	var claims map[string]interface{}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to extract claims: %w", err)
	}

	return &TokenResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		IDToken:      rawIDToken,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry,
		Claims:       claims,
	}, nil
}

// RefreshToken refreshes the access token
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.httpClient)
	
	tokenSource := c.oauth2Config.TokenSource(ctx, &oauth2.Token{
		RefreshToken: refreshToken,
	})

	token, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	// Extract ID token if present
	rawIDToken, _ := token.Extra("id_token").(string)
	
	var claims map[string]interface{}
	if rawIDToken != "" {
		// Verify ID token
		idToken, err := c.verifier.Verify(ctx, rawIDToken)
		if err != nil {
			return nil, fmt.Errorf("failed to verify ID token: %w", err)
		}

		// Extract claims
		if err := idToken.Claims(&claims); err != nil {
			return nil, fmt.Errorf("failed to extract claims: %w", err)
		}
	}

	return &TokenResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		IDToken:      rawIDToken,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry,
		Claims:       claims,
	}, nil
}

// UserInfo fetches user information from the userinfo endpoint
func (c *Client) UserInfo(ctx context.Context, accessToken string) (map[string]interface{}, error) {
	userInfo, err := c.provider.UserInfo(ctx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: accessToken,
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}

	var claims map[string]interface{}
	if err := userInfo.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to extract user info claims: %w", err)
	}

	return claims, nil
}

// generateCodeVerifier generates a PKCE code verifier
func generateCodeVerifier() (string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Base64 URL encode without padding
	verifier := base64.RawURLEncoding.EncodeToString(bytes)
	return verifier, nil
}

// generateCodeChallenge generates a PKCE code challenge from the verifier
func generateCodeChallenge(verifier string) string {
	// SHA256 hash of the verifier
	hash := sha256.Sum256([]byte(verifier))
	// Base64 URL encode without padding
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// TokenResponse represents the response from token exchange
type TokenResponse struct {
	AccessToken  string
	RefreshToken string
	IDToken      string
	TokenType    string
	Expiry       time.Time
	Claims       map[string]interface{}
}