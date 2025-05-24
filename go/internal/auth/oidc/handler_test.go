package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockSessionStore is a mock implementation of session.Store
type MockSessionStore struct {
	mock.Mock
}

func (m *MockSessionStore) Create(ctx context.Context, key string, data interface{}, ttl time.Duration) (string, error) {
	args := m.Called(ctx, key, data, ttl)
	return args.String(0), args.Error(1)
}

func (m *MockSessionStore) Get(ctx context.Context, key string, data interface{}) error {
	args := m.Called(ctx, key, data)
	return args.Error(0)
}

func (m *MockSessionStore) Update(ctx context.Context, key string, data interface{}) error {
	args := m.Called(ctx, key, data)
	return args.Error(0)
}

func (m *MockSessionStore) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockSessionStore) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockSessionStore) Refresh(ctx context.Context, key string, ttl time.Duration) error {
	args := m.Called(ctx, key, ttl)
	return args.Error(0)
}

func (m *MockSessionStore) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestNewHandler(t *testing.T) {
	logger := zap.NewNop()
	mockStore := new(MockSessionStore)

	tests := []struct {
		name          string
		config        *config.OIDCConfig
		expectError   bool
		errorContains string
	}{
		{
			name:          "Missing discovery URL",
			config:        &config.OIDCConfig{},
			expectError:   true,
			errorContains: "OIDC discovery URL is required",
		},
		{
			name: "Missing client ID",
			config: &config.OIDCConfig{
				DiscoveryURL: "https://example.com/.well-known/openid-configuration",
			},
			expectError:   true,
			errorContains: "OIDC client ID is required",
		},
		{
			name: "Missing client secret",
			config: &config.OIDCConfig{
				DiscoveryURL: "https://example.com/.well-known/openid-configuration",
				ClientID:     "test-client",
			},
			expectError:   true,
			errorContains: "OIDC client secret is required",
		},
		{
			name: "Missing redirect URL",
			config: &config.OIDCConfig{
				DiscoveryURL: "https://example.com/.well-known/openid-configuration",
				ClientID:     "test-client",
				ClientSecret: "test-secret",
			},
			expectError:   true,
			errorContains: "OIDC redirect URL is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			handler, err := NewHandler(ctx, tt.config, mockStore, logger)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, handler)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, handler)
			}
		})
	}
}

func TestAuthorize(t *testing.T) {
	// Create test server
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	// Create mock OIDC provider
	var oidcServer *httptest.Server
	oidcServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/.well-known/openid-configuration" {
			config := map[string]interface{}{
				"issuer":                 oidcServer.URL,
				"authorization_endpoint": oidcServer.URL + "/auth",
				"token_endpoint":         oidcServer.URL + "/token",
				"jwks_uri":               oidcServer.URL + "/jwks",
			}
			json.NewEncoder(w).Encode(config)
		} else if r.URL.Path == "/jwks" {
			jwks := map[string]interface{}{
				"keys": []interface{}{},
			}
			json.NewEncoder(w).Encode(jwks)
		}
	}))
	defer oidcServer.Close()

	cfg := &config.OIDCConfig{
		DiscoveryURL: oidcServer.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"openid", "email"},
	}

	mockStore := new(MockSessionStore)
	handler, err := NewHandler(context.Background(), cfg, mockStore, logger)
	require.NoError(t, err)

	// Set up expectation for session creation
	mockStore.On("Create", mock.Anything, mock.MatchedBy(func(key string) bool {
		return len(key) > 5 && key[:5] == "auth:"
	}), mock.Anything, 10*time.Minute).Return("session-123", nil)

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/auth/authorize?redirect_uri=/dashboard", nil)

	// Call handler
	handler.Authorize(c)

	// Check response
	assert.Equal(t, http.StatusFound, w.Code)
	location := w.Header().Get("Location")
	assert.NotEmpty(t, location)

	// Parse redirect URL
	u, err := url.Parse(location)
	require.NoError(t, err)
	
	// Check required parameters
	params := u.Query()
	assert.NotEmpty(t, params.Get("state"))
	assert.NotEmpty(t, params.Get("code_challenge"))
	assert.Equal(t, "S256", params.Get("code_challenge_method"))
	assert.Equal(t, "test-client", params.Get("client_id"))
	assert.Equal(t, "code", params.Get("response_type"))
	assert.Contains(t, params.Get("scope"), "openid")
	assert.Contains(t, params.Get("scope"), "email")

	mockStore.AssertExpectations(t)
}

func TestCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	tests := []struct {
		name           string
		queryParams    map[string]string
		setupMock      func(*MockSessionStore)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Authorization error",
			queryParams: map[string]string{
				"error":             "access_denied",
				"error_description": "User denied access",
			},
			setupMock:      func(m *MockSessionStore) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "access_denied",
		},
		{
			name: "Missing state",
			queryParams: map[string]string{
				"code": "test-code",
			},
			setupMock:      func(m *MockSessionStore) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Missing required parameters",
		},
		{
			name: "Missing code",
			queryParams: map[string]string{
				"state": "test-state",
			},
			setupMock:      func(m *MockSessionStore) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Missing required parameters",
		},
		{
			name: "Invalid state",
			queryParams: map[string]string{
				"state": "test-state",
				"code":  "test-code",
			},
			setupMock: func(m *MockSessionStore) {
				m.On("Get", mock.Anything, "auth:test-state", mock.Anything).Return(fmt.Errorf("not found"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid or expired state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := new(MockSessionStore)
			tt.setupMock(mockStore)

			// Create dummy handler (won't actually use OIDC client in these tests)
			handler := &Handler{
				sessionStore: mockStore,
				logger:       logger,
			}

			// Create test request
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			
			// Build query string
			values := url.Values{}
			for k, v := range tt.queryParams {
				values.Add(k, v)
			}
			c.Request = httptest.NewRequest("GET", "/callback?"+values.Encode(), nil)

			// Call handler
			handler.Callback(c)

			// Check response
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], tt.expectedError)
			}

			mockStore.AssertExpectations(t)
		})
	}
}

func TestLogout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	tests := []struct {
		name               string
		sessionCookie      string
		endSessionEndpoint string
		expectedLocation   string
		setupMock          func(*MockSessionStore)
	}{
		{
			name:             "No session cookie",
			sessionCookie:    "",
			expectedLocation: "/",
			setupMock:        func(m *MockSessionStore) {},
		},
		{
			name:             "With session cookie",
			sessionCookie:    "session-123",
			expectedLocation: "/",
			setupMock: func(m *MockSessionStore) {
				m.On("Delete", mock.Anything, "session-123").Return(nil)
			},
		},
		{
			name:               "With end session endpoint",
			sessionCookie:      "session-123",
			endSessionEndpoint: "https://example.com/logout",
			expectedLocation:   "https://example.com/logout?post_logout_redirect_uri=http://localhost:8080",
			setupMock: func(m *MockSessionStore) {
				m.On("Delete", mock.Anything, "session-123").Return(nil)
				m.On("Get", mock.Anything, "session-123", mock.Anything).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := new(MockSessionStore)
			tt.setupMock(mockStore)

			postLogoutURI := ""
			if tt.endSessionEndpoint != "" {
				postLogoutURI = "http://localhost:8080"
			}
			handler := &Handler{
				sessionStore: mockStore,
				logger:       logger,
				config: &config.OIDCConfig{
					EndSessionEndpoint:    tt.endSessionEndpoint,
					PostLogoutRedirectURI: postLogoutURI,
				},
			}

			// Create test request
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/logout", nil)

			// Set session cookie if provided
			if tt.sessionCookie != "" {
				c.Request.AddCookie(&http.Cookie{
					Name:  "session_id",
					Value: tt.sessionCookie,
				})
			}

			// Call handler
			handler.Logout(c)

			// Check response
			assert.Equal(t, http.StatusFound, w.Code)
			assert.Equal(t, tt.expectedLocation, w.Header().Get("Location"))

			// Check cookie was cleared
			cookies := w.Result().Cookies()
			for _, cookie := range cookies {
				if cookie.Name == "session_id" {
					assert.Equal(t, -1, cookie.MaxAge)
				}
			}

			mockStore.AssertExpectations(t)
		})
	}
}

func TestGenerateRandomString(t *testing.T) {
	// Test multiple times to ensure randomness
	seen := make(map[string]bool)
	for i := 0; i < 10; i++ {
		str, err := generateRandomString(32)
		assert.NoError(t, err)
		assert.NotEmpty(t, str)
		
		// Check for uniqueness
		assert.False(t, seen[str], "Generated duplicate random string")
		seen[str] = true
	}
}