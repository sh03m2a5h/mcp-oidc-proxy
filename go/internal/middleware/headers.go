package middleware

import (
	"crypto/rand"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/auth/oidc"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/config"
	"go.uber.org/zap"
)

// HeaderInjector handles custom header injection
type HeaderInjector struct {
	config *config.HeadersConfig
	logger *zap.Logger
}

// NewHeaderInjector creates a new header injector
func NewHeaderInjector(config *config.HeadersConfig, logger *zap.Logger) *HeaderInjector {
	return &HeaderInjector{
		config: config,
		logger: logger,
	}
}

// InjectHeaders injects custom headers into the request
func (hi *HeaderInjector) InjectHeaders(r *http.Request, sess *oidc.UserSession) {
	// Inject static custom headers
	hi.injectStaticHeaders(r)
	
	// Inject dynamic headers
	hi.injectDynamicHeaders(r, sess)
	
	// Inject user headers from session if available
	if sess != nil {
		hi.injectUserHeaders(r, sess)
	}
}

// injectStaticHeaders injects static custom headers from configuration
func (hi *HeaderInjector) injectStaticHeaders(r *http.Request) {
	if hi.config.Custom == nil {
		return
	}
	
	for name, value := range hi.config.Custom {
		if name != "" && value != "" {
			r.Header.Set(name, value)
			hi.logger.Debug("Injected static header",
				zap.String("header_name", name),
				zap.String("header_value", value),
			)
		}
	}
}

// injectDynamicHeaders injects dynamic headers based on request context
func (hi *HeaderInjector) injectDynamicHeaders(r *http.Request, sess *oidc.UserSession) {
	// Timestamp header
	if hi.config.Dynamic.Timestamp.Enabled && hi.config.Dynamic.Timestamp.HeaderName != "" {
		timestamp := hi.formatTimestamp(hi.config.Dynamic.Timestamp.Format)
		r.Header.Set(hi.config.Dynamic.Timestamp.HeaderName, timestamp)
		hi.logger.Debug("Injected timestamp header",
			zap.String("header_name", hi.config.Dynamic.Timestamp.HeaderName),
			zap.String("timestamp", timestamp),
		)
	}
	
	// Request ID header
	if hi.config.Dynamic.RequestID.Enabled && hi.config.Dynamic.RequestID.HeaderName != "" {
		requestID := hi.generateRequestID()
		r.Header.Set(hi.config.Dynamic.RequestID.HeaderName, requestID)
		hi.logger.Debug("Injected request ID header",
			zap.String("header_name", hi.config.Dynamic.RequestID.HeaderName),
			zap.String("request_id", requestID),
		)
	}
	
	// Client IP header
	if hi.config.Dynamic.ClientIP.Enabled && hi.config.Dynamic.ClientIP.HeaderName != "" {
		clientIP := hi.getClientIP(r)
		r.Header.Set(hi.config.Dynamic.ClientIP.HeaderName, clientIP)
		hi.logger.Debug("Injected client IP header",
			zap.String("header_name", hi.config.Dynamic.ClientIP.HeaderName),
			zap.String("client_ip", clientIP),
		)
	}
	
	// User Agent header
	if hi.config.Dynamic.UserAgent.Enabled && hi.config.Dynamic.UserAgent.HeaderName != "" {
		userAgent := r.UserAgent()
		if userAgent != "" {
			r.Header.Set(hi.config.Dynamic.UserAgent.HeaderName, userAgent)
			hi.logger.Debug("Injected user agent header",
				zap.String("header_name", hi.config.Dynamic.UserAgent.HeaderName),
				zap.String("user_agent", userAgent),
			)
		}
	}
	
	// Session ID header
	if hi.config.Dynamic.SessionID.Enabled && hi.config.Dynamic.SessionID.HeaderName != "" && sess != nil {
		sessionID := sess.ID
		if sessionID != "" {
			r.Header.Set(hi.config.Dynamic.SessionID.HeaderName, sessionID)
			hi.logger.Debug("Injected session ID header",
				zap.String("header_name", hi.config.Dynamic.SessionID.HeaderName),
				zap.String("session_id", sessionID),
			)
		}
	}
	
	// Correlation ID header (generate or preserve existing)
	if hi.config.Dynamic.CorrelationID.Enabled && hi.config.Dynamic.CorrelationID.HeaderName != "" {
		correlationID := r.Header.Get(hi.config.Dynamic.CorrelationID.HeaderName)
		if correlationID == "" {
			correlationID = hi.generateCorrelationID()
			r.Header.Set(hi.config.Dynamic.CorrelationID.HeaderName, correlationID)
		}
		hi.logger.Debug("Injected correlation ID header",
			zap.String("header_name", hi.config.Dynamic.CorrelationID.HeaderName),
			zap.String("correlation_id", correlationID),
		)
	}
}

// injectUserHeaders injects user-related headers from session
func (hi *HeaderInjector) injectUserHeaders(r *http.Request, sess *oidc.UserSession) {
	// User ID header
	if hi.config.UserID != "" && sess.ID != "" {
		r.Header.Set(hi.config.UserID, sess.ID)
		hi.logger.Debug("Injected user ID header",
			zap.String("header_name", hi.config.UserID),
			zap.String("user_id", sess.ID),
		)
	}
	
	// User Email header
	if hi.config.UserEmail != "" && sess.Email != "" {
		r.Header.Set(hi.config.UserEmail, sess.Email)
		hi.logger.Debug("Injected user email header",
			zap.String("header_name", hi.config.UserEmail),
			zap.String("user_email", sess.Email),
		)
	}
	
	// User Name header
	if hi.config.UserName != "" && sess.Name != "" {
		r.Header.Set(hi.config.UserName, sess.Name)
		hi.logger.Debug("Injected user name header",
			zap.String("header_name", hi.config.UserName),
			zap.String("user_name", sess.Name),
		)
	}
	
	// User Groups header - extract from claims
	if hi.config.UserGroups != "" && sess.Claims != nil {
		if groupsValue, exists := sess.Claims["groups"]; exists {
			var groups []string
			// Handle different types of groups claim
			switch v := groupsValue.(type) {
			case []string:
				groups = v
			case []interface{}:
				for _, g := range v {
					if gStr, ok := g.(string); ok {
						groups = append(groups, gStr)
					}
				}
			case string:
				// Single group as string
				groups = []string{v}
			}
			
			if len(groups) > 0 {
				groupsStr := strings.Join(groups, ",")
				r.Header.Set(hi.config.UserGroups, groupsStr)
				hi.logger.Debug("Injected user groups header",
					zap.String("header_name", hi.config.UserGroups),
					zap.String("user_groups", groupsStr),
				)
			}
		}
	}
}

// formatTimestamp formats timestamp according to the specified format
func (hi *HeaderInjector) formatTimestamp(format string) string {
	now := time.Now()
	
	switch format {
	case "unix":
		return fmt.Sprintf("%d", now.Unix())
	case "unix_nano":
		return fmt.Sprintf("%d", now.UnixNano())
	case "rfc3339":
		return now.Format(time.RFC3339)
	case "rfc3339_nano":
		return now.Format(time.RFC3339Nano)
	case "iso8601":
		return now.Format("2006-01-02T15:04:05Z07:00")
	default:
		// Use format as Go time format if not predefined
		if format != "" {
			return now.Format(format)
		}
		// Default to RFC3339
		return now.Format(time.RFC3339)
	}
}

// generateRequestID generates a unique request ID
func (hi *HeaderInjector) generateRequestID() string {
	// Generate 16 bytes of random data
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		hi.logger.Warn("Failed to generate random request ID, using timestamp", zap.Error(err))
		return fmt.Sprintf("req_%d", time.Now().UnixNano())
	}
	
	// Convert to hex string
	return fmt.Sprintf("req_%x", bytes)
}

// generateCorrelationID generates a unique correlation ID
func (hi *HeaderInjector) generateCorrelationID() string {
	// Generate 12 bytes of random data for shorter correlation ID
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		hi.logger.Warn("Failed to generate random correlation ID, using timestamp", zap.Error(err))
		return fmt.Sprintf("corr_%d", time.Now().UnixNano())
	}
	
	// Convert to hex string
	return fmt.Sprintf("corr_%x", bytes)
}

// getClientIP extracts the client IP from request headers
func (hi *HeaderInjector) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP from the comma-separated list
		if ips := strings.Split(xff, ","); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Check CF-Connecting-IP (Cloudflare)
	if cfip := r.Header.Get("CF-Connecting-IP"); cfip != "" {
		return cfip
	}
	
	// Fall back to RemoteAddr
	if remoteAddr := r.RemoteAddr; remoteAddr != "" {
		// Use net.SplitHostPort to handle both IPv4 and IPv6 addresses
		if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
			return host
		}
		// If SplitHostPort fails, assume remoteAddr is the IP itself (no port)
		return remoteAddr
	}
	
	return "unknown"
}

// Middleware returns a middleware function that injects headers
func (hi *HeaderInjector) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get session from context if available
		sess := oidc.GetSessionFromContext(r.Context())
		
		// Inject headers
		hi.InjectHeaders(r, sess)
		
		// Continue to next handler
		next.ServeHTTP(w, r)
	})
}
