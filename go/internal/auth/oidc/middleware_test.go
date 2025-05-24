package oidc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	tests := []struct {
		name           string
		path           string
		sessionCookie  string
		excludePaths   []string
		setupMock      func(*MockSessionStore)
		expectedStatus int
		expectedError  string
		checkHeaders   bool
	}{
		{
			name:           "Excluded path",
			path:           "/health",
			excludePaths:   []string{"/health", "/version"},
			setupMock:      func(m *MockSessionStore) {},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "No session cookie",
			path:           "/api/data",
			excludePaths:   []string{"/health"},
			setupMock:      func(m *MockSessionStore) {},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Authentication required",
		},
		{
			name:          "Invalid session",
			path:          "/api/data",
			sessionCookie: "invalid-session",
			excludePaths:  []string{"/health"},
			setupMock: func(m *MockSessionStore) {
				m.On("Get", mock.Anything, "invalid-session", mock.Anything).Return(assert.AnError)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid or expired session",
		},
		{
			name:          "Expired session",
			path:          "/api/data",
			sessionCookie: "expired-session",
			excludePaths:  []string{"/health"},
			setupMock: func(m *MockSessionStore) {
				m.On("Get", mock.Anything, "expired-session", mock.Anything).Run(func(args mock.Arguments) {
					userSession := args.Get(2).(*UserSession)
					userSession.ID = "user123"
					userSession.ExpiresAt = time.Now().Add(-time.Hour) // Expired
				}).Return(nil)
				m.On("Delete", mock.Anything, "expired-session").Return(nil)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Session expired",
		},
		{
			name:          "Valid session",
			path:          "/api/data",
			sessionCookie: "valid-session",
			excludePaths:  []string{"/health"},
			setupMock: func(m *MockSessionStore) {
				m.On("Get", mock.Anything, "valid-session", mock.Anything).Run(func(args mock.Arguments) {
					userSession := args.Get(2).(*UserSession)
					userSession.ID = "user123"
					userSession.Email = "test@example.com"
					userSession.Name = "Test User"
					userSession.ExpiresAt = time.Now().Add(time.Hour)
				}).Return(nil)
			},
			expectedStatus: http.StatusOK,
			checkHeaders:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := new(MockSessionStore)
			tt.setupMock(mockStore)

			// Create middleware
			middleware := AuthMiddleware(mockStore, logger, tt.excludePaths)

			// Create test context
			w := httptest.NewRecorder()
			c, router := gin.CreateTestContext(w)

			// Set up route with middleware
			router.Use(middleware)
			router.GET("/*path", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			// Create request
			c.Request = httptest.NewRequest("GET", tt.path, nil)
			if tt.sessionCookie != "" {
				c.Request.AddCookie(&http.Cookie{
					Name:  "session_id",
					Value: tt.sessionCookie,
				})
			}

			// Execute request
			router.ServeHTTP(w, c.Request)

			// Check response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response["error"], tt.expectedError)
			}

			if tt.checkHeaders {
				// Check that user headers were added
				assert.Equal(t, "user123", c.Request.Header.Get("X-User-ID"))
				assert.Equal(t, "test@example.com", c.Request.Header.Get("X-User-Email"))
				assert.Equal(t, "Test User", c.Request.Header.Get("X-User-Name"))
			}

			mockStore.AssertExpectations(t)
		})
	}
}

func TestOptionalAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	tests := []struct {
		name             string
		sessionCookie    string
		setupMock        func(*MockSessionStore)
		expectedStatus   int
		expectAuthHeader bool
	}{
		{
			name:           "No session cookie",
			sessionCookie:  "",
			setupMock:      func(m *MockSessionStore) {},
			expectedStatus: http.StatusOK,
		},
		{
			name:          "Invalid session",
			sessionCookie: "invalid-session",
			setupMock: func(m *MockSessionStore) {
				m.On("Get", mock.Anything, "invalid-session", mock.Anything).Return(assert.AnError)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:          "Expired session",
			sessionCookie: "expired-session",
			setupMock: func(m *MockSessionStore) {
				m.On("Get", mock.Anything, "expired-session", mock.Anything).Run(func(args mock.Arguments) {
					userSession := args.Get(2).(*UserSession)
					userSession.ExpiresAt = time.Now().Add(-time.Hour)
				}).Return(nil)
				m.On("Delete", mock.Anything, "expired-session").Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:          "Valid session",
			sessionCookie: "valid-session",
			setupMock: func(m *MockSessionStore) {
				m.On("Get", mock.Anything, "valid-session", mock.Anything).Run(func(args mock.Arguments) {
					userSession := args.Get(2).(*UserSession)
					userSession.ID = "user123"
					userSession.Email = "test@example.com"
					userSession.Name = "Test User"
					userSession.ExpiresAt = time.Now().Add(time.Hour)
				}).Return(nil)
			},
			expectedStatus:   http.StatusOK,
			expectAuthHeader: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := new(MockSessionStore)
			tt.setupMock(mockStore)

			// Create middleware
			middleware := OptionalAuthMiddleware(mockStore, logger)

			// Create test context
			w := httptest.NewRecorder()
			c, router := gin.CreateTestContext(w)

			// Set up route with middleware
			router.Use(middleware)
			router.GET("/test", func(c *gin.Context) {
				authenticated, _ := c.Get("authenticated")
				c.JSON(http.StatusOK, gin.H{
					"authenticated": authenticated,
				})
			})

			// Create request
			c.Request = httptest.NewRequest("GET", "/test", nil)
			if tt.sessionCookie != "" {
				c.Request.AddCookie(&http.Cookie{
					Name:  "session_id",
					Value: tt.sessionCookie,
				})
			}

			// Execute request
			router.ServeHTTP(w, c.Request)

			// Check response
			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectAuthHeader {
				assert.True(t, response["authenticated"].(bool))
				assert.Equal(t, "user123", c.Request.Header.Get("X-User-ID"))
				assert.Equal(t, "test@example.com", c.Request.Header.Get("X-User-Email"))
				assert.Equal(t, "Test User", c.Request.Header.Get("X-User-Name"))
			} else {
				// authenticated should be nil or false
				assert.NotEqual(t, true, response["authenticated"])
			}

			mockStore.AssertExpectations(t)
		})
	}
}