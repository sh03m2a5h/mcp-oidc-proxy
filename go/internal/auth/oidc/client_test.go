package oidc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCodeVerifier(t *testing.T) {
	verifier, err := generateCodeVerifier()
	require.NoError(t, err)
	assert.NotEmpty(t, verifier)
	assert.Len(t, verifier, 43) // Base64 URL encoded 32 bytes
}

func TestGenerateCodeChallenge(t *testing.T) {
	verifier := "test-verifier"
	challenge := generateCodeChallenge(verifier)
	assert.NotEmpty(t, challenge)
	// Should be base64 URL encoded SHA256 hash
	assert.True(t, strings.ContainsAny(challenge, "-_"))
}

func TestNewClient(t *testing.T) {
	// Create mock OIDC provider
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Mock server received request: %s", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/.well-known/openid-configuration" {
			config := map[string]interface{}{
				"issuer":                 server.URL,
				"authorization_endpoint": server.URL + "/auth",
				"token_endpoint":         server.URL + "/token",
				"userinfo_endpoint":      server.URL + "/userinfo",
				"jwks_uri":               server.URL + "/jwks",
			}
			if err := json.NewEncoder(w).Encode(config); err != nil {
				t.Logf("Failed to encode config: %v", err)
			}
		} else if r.URL.Path == "/jwks" {
			// Mock JWKS response
			jwks := map[string]interface{}{
				"keys": []interface{}{},
			}
			json.NewEncoder(w).Encode(jwks)
		} else {
			t.Logf("Unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tests := []struct {
		name          string
		discoveryURL  string
		clientID      string
		clientSecret  string
		redirectURL   string
		scopes        []string
		expectError   bool
		errorContains string
	}{
		{
			name:         "Valid configuration",
			discoveryURL: server.URL,
			clientID:     "test-client",
			clientSecret: "test-secret",
			redirectURL:  "http://localhost:8080/callback",
			scopes:       []string{"openid", "email", "profile"},
			expectError:  false,
		},
		{
			name:          "Invalid discovery URL",
			discoveryURL:  "http://invalid-url/.well-known/openid-configuration",
			clientID:      "test-client",
			clientSecret:  "test-secret",
			redirectURL:   "http://localhost:8080/callback",
			scopes:        []string{"openid"},
			expectError:   true,
			errorContains: "failed to discover OIDC provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client, err := NewClient(ctx, tt.discoveryURL, tt.clientID, tt.clientSecret, tt.redirectURL, tt.scopes)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.NotNil(t, client.provider)
				assert.NotNil(t, client.oauth2Config)
				assert.NotNil(t, client.verifier)
				assert.Equal(t, tt.clientID, client.oauth2Config.ClientID)
				assert.Equal(t, tt.clientSecret, client.oauth2Config.ClientSecret)
				assert.Equal(t, tt.redirectURL, client.oauth2Config.RedirectURL)
				assert.Equal(t, tt.scopes, client.oauth2Config.Scopes)
			}
		})
	}
}

func TestAuthCodeURL(t *testing.T) {
	// Create mock OIDC provider
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/.well-known/openid-configuration" {
			config := map[string]interface{}{
				"issuer":                 server.URL,
				"authorization_endpoint": server.URL + "/auth",
				"token_endpoint":         server.URL + "/token",
				"jwks_uri":               server.URL + "/jwks",
			}
			json.NewEncoder(w).Encode(config)
		} else if r.URL.Path == "/jwks" {
			jwks := map[string]interface{}{
				"keys": []interface{}{},
			}
			json.NewEncoder(w).Encode(jwks)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	client, err := NewClient(
		ctx,
		server.URL,
		"test-client",
		"test-secret",
		"http://localhost:8080/callback",
		[]string{"openid", "email"},
	)
	require.NoError(t, err)

	state := "test-state"
	authURL, codeVerifier, codeChallenge, err := client.AuthCodeURL(state)
	
	assert.NoError(t, err)
	assert.NotEmpty(t, authURL)
	assert.NotEmpty(t, codeVerifier)
	assert.NotEmpty(t, codeChallenge)
	
	// Check URL contains required parameters
	assert.Contains(t, authURL, "state="+state)
	assert.Contains(t, authURL, "code_challenge=")
	assert.Contains(t, authURL, "code_challenge_method=S256")
	assert.Contains(t, authURL, "client_id=test-client")
	assert.Contains(t, authURL, "redirect_uri=")
	assert.Contains(t, authURL, "response_type=code")
	assert.Contains(t, authURL, "scope=openid+email")
}

func TestTokenResponse(t *testing.T) {
	resp := &TokenResponse{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		IDToken:      "test-id-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
		Claims: map[string]interface{}{
			"sub":   "user123",
			"email": "test@example.com",
			"name":  "Test User",
		},
	}

	assert.Equal(t, "test-access-token", resp.AccessToken)
	assert.Equal(t, "test-refresh-token", resp.RefreshToken)
	assert.Equal(t, "test-id-token", resp.IDToken)
	assert.Equal(t, "Bearer", resp.TokenType)
	assert.True(t, resp.Expiry.After(time.Now()))
	assert.Equal(t, "user123", resp.Claims["sub"])
	assert.Equal(t, "test@example.com", resp.Claims["email"])
	assert.Equal(t, "Test User", resp.Claims["name"])
}