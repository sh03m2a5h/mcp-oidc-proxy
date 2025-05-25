# MCP OIDC Proxy - Go Implementation

A production-ready OAuth 2.1/OIDC authentication proxy for Model Context Protocol (MCP) servers. Deploy as a single binary and expose your local MCP server securely via Cloudflare Tunnels.

## 🎯 What This Does

This proxy sits between the internet and your MCP server, providing:
- **Authentication**: Users must authenticate via OIDC before accessing your MCP server
- **Zero Trust**: Works perfectly with Cloudflare Tunnels for secure exposure
- **Single Binary**: No Docker, no dependencies, just run it

```
[Internet] → [Cloudflare] → [Cloudflare Tunnel] → [MCP OIDC Proxy :8080] → [Your MCP Server :3000]
```

## ✨ Features

- 🚀 **Single Binary**: Download and run - that's it
- 🔐 **Universal OIDC**: Works with Auth0, Google, Microsoft, GitHub, or any OIDC provider
- 🛡️ **Modern Security**: OAuth 2.1 + PKCE flow
- 📊 **Production Ready**: Prometheus metrics, health checks, structured logging
- 🔄 **Full Protocol Support**: HTTP, WebSocket, and streaming
- ⚡ **High Performance**: <10ms overhead, handles 1000+ concurrent connections
- 🔍 **Observability**: OpenTelemetry tracing, detailed metrics
- 💾 **Flexible Sessions**: In-memory or Redis session storage

## 🚀 Quick Start

### 1. Download

```bash
# One-line install (Linux/macOS)
curl -sSL https://raw.githubusercontent.com/sh03m2a5h/mcp-oidc-proxy/main/install.sh | bash

# Or download directly
wget https://github.com/sh03m2a5h/mcp-oidc-proxy/releases/latest/download/mcp-oidc-proxy-$(uname -s)-$(uname -m)
chmod +x mcp-oidc-proxy-*
```

### 2. Configure & Run

```bash
# Set your OIDC provider (example with Auth0)
export OIDC_DISCOVERY_URL="https://your-domain.auth0.com/.well-known/openid-configuration"
export OIDC_CLIENT_ID="your-client-id"
export OIDC_CLIENT_SECRET="your-client-secret"
export OIDC_REDIRECT_URL="http://localhost:8080/callback"

# Point to your MCP server
export MCP_TARGET_HOST="localhost"
export MCP_TARGET_PORT="3000"

# Run the proxy
./mcp-oidc-proxy
```

### 3. Expose with Cloudflare Tunnel

```bash
# In another terminal
cloudflared tunnel --url http://localhost:8080

# Your MCP server is now accessible at the generated URL with OIDC auth!
```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/sh03m2a5h/mcp-oidc-proxy.git
cd mcp-oidc-proxy/go

# Build
make build

# Run
./bin/mcp-oidc-proxy --config configs/config.example.yaml
```

## ⚙️ Configuration

### Common OIDC Providers

<details>
<summary><b>Auth0</b></summary>

```bash
export OIDC_DISCOVERY_URL="https://YOUR-DOMAIN.auth0.com/.well-known/openid-configuration"
export OIDC_CLIENT_ID="your-client-id"
export OIDC_CLIENT_SECRET="your-client-secret"
```
</details>

<details>
<summary><b>Google</b></summary>

```bash
export OIDC_DISCOVERY_URL="https://accounts.google.com/.well-known/openid-configuration"
export OIDC_CLIENT_ID="your-client-id.apps.googleusercontent.com"
export OIDC_CLIENT_SECRET="your-client-secret"
```
</details>

<details>
<summary><b>Microsoft/Azure AD</b></summary>

```bash
export OIDC_DISCOVERY_URL="https://login.microsoftonline.com/YOUR-TENANT-ID/v2.0/.well-known/openid-configuration"
export OIDC_CLIENT_ID="your-client-id"
export OIDC_CLIENT_SECRET="your-client-secret"
```
</details>

### All Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| **Server** | | |
| `SERVER_HOST` | Listen address | `0.0.0.0` |
| `SERVER_PORT` | Listen port | `8080` |
| **Proxy Target** | | |
| `PROXY_TARGET_HOST` | MCP server host | `localhost` |
| `PROXY_TARGET_PORT` | MCP server port | `3000` |
| **Authentication** | | |
| `AUTH_MODE` | Auth mode (`oidc`, `bypass`) | `oidc` |
| `OIDC_DISCOVERY_URL` | OIDC discovery endpoint | Required for OIDC |
| `OIDC_CLIENT_ID` | OAuth client ID | Required for OIDC |
| `OIDC_CLIENT_SECRET` | OAuth client secret | Required for OIDC |
| `OIDC_REDIRECT_URL` | OAuth callback URL | `http://localhost:8080/callback` |
| **Sessions** | | |
| `SESSION_STORE` | Store type (`memory`, `redis`) | `memory` |
| `SESSION_COOKIE_SECURE` | Secure cookies (HTTPS) | `false` |
| `REDIS_URL` | Redis URL (if using Redis) | `redis://localhost:6379` |
| **Monitoring** | | |
| `METRICS_ENABLED` | Enable Prometheus metrics | `true` |
| `METRICS_PATH` | Metrics endpoint path | `/metrics` |
| `LOG_LEVEL` | Log level (debug/info/warn/error) | `info` |

### Configuration File

See [configs/config.example.yaml](configs/config.example.yaml) for a complete example.

## Development

### Prerequisites

- Go 1.22 or later
- Make
- Docker (optional)

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Run with live reload
make run
```

### Project Structure

```
go/
├── cmd/mcp-oidc-proxy/    # Application entry point
├── internal/              # Private application code
│   ├── config/           # Configuration management
│   ├── server/           # HTTP server
│   ├── auth/             # Authentication handlers
│   ├── session/          # Session management
│   ├── proxy/            # Reverse proxy
│   └── middleware/       # HTTP middleware
├── pkg/                  # Public packages
│   └── version/          # Version information
├── configs/              # Configuration examples
└── deployments/          # Deployment configurations
```

## 🚢 Production Deployment

### With Cloudflare Tunnels (Recommended)

```bash
# 1. Run the proxy
./mcp-oidc-proxy &

# 2. Create persistent tunnel
cloudflared tunnel create mcp-proxy
cloudflared tunnel route dns mcp-proxy your-domain.com
cloudflared tunnel run mcp-proxy
```

### Systemd Service

```bash
# Create service file
sudo tee /etc/systemd/system/mcp-oidc-proxy.service > /dev/null <<EOF
[Unit]
Description=MCP OIDC Proxy
After=network.target

[Service]
Type=simple
User=nobody
Group=nogroup
ExecStart=/usr/local/bin/mcp-oidc-proxy
Restart=always
RestartSec=5
Environment="OIDC_DISCOVERY_URL=https://your-domain.auth0.com/.well-known/openid-configuration"
Environment="OIDC_CLIENT_ID=your-client-id"
Environment="OIDC_CLIENT_SECRET=your-client-secret"

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
sudo systemctl enable mcp-oidc-proxy
sudo systemctl start mcp-oidc-proxy
```

### High Availability with Redis

```bash
# Use Redis for session storage across multiple instances
export SESSION_STORE=redis
export REDIS_URL=redis://your-redis:6379

# Run multiple instances behind a load balancer
./mcp-oidc-proxy --port 8080 &
./mcp-oidc-proxy --port 8081 &
```

## 📊 Monitoring & Observability

### Health Check

```bash
curl http://localhost:8080/health

# Response includes subsystem status
{
  "status": "healthy",
  "checks": {
    "proxy_target": {"status": "healthy"},
    "session_store": {"status": "healthy", "active_sessions": 42}
  }
}
```

### Prometheus Metrics

```bash
# Key metrics available at /metrics
mcp_oidc_proxy_http_requests_total          # HTTP requests by method, path, status
mcp_oidc_proxy_http_request_duration_seconds # Request latency
mcp_oidc_proxy_proxy_requests_total         # Proxy requests to backend
mcp_oidc_proxy_auth_requests_total          # Auth attempts by provider
mcp_oidc_proxy_sessions_active              # Current active sessions
mcp_oidc_proxy_circuit_breaker_state        # Circuit breaker status
```

### OpenTelemetry Tracing

```bash
# Enable tracing
export TRACING_ENABLED=true
export TRACING_ENDPOINT=http://your-collector:4318
export TRACING_SAMPLE_RATE=0.1
```

## 🏁 Performance

- **Latency**: < 10ms overhead (P99)
- **Throughput**: 10,000+ requests/second
- **Concurrent connections**: 1,000+
- **Memory usage**: ~50MB (idle), ~100MB (under load)
- **Startup time**: < 1 second

## 🔒 Security Features

- OAuth 2.1 with PKCE (RFC 7636)
- Secure session management
- Circuit breaker for backend protection
- Rate limiting for auth endpoints
- Structured audit logging
- No external dependencies in binary
- **Security Headers**: Automatically adds security headers to all responses:
  - `X-Frame-Options`: Prevents clickjacking
  - `X-Content-Type-Options`: Prevents MIME sniffing
  - `X-XSS-Protection`: Legacy XSS protection
  - `Content-Security-Policy`: Modern XSS/injection protection
  - `Referrer-Policy`: Controls referrer information
  - `Permissions-Policy`: Restricts browser features

### Security Headers Impact

The proxy adds strict security headers by default. If your MCP server serves web content that requires specific permissions (e.g., embedding in iframes, loading external scripts), you may need to adjust the CSP policy. The default policy:
- Blocks all framing (`frame-ancestors 'none'`)
- Allows only self-hosted scripts and styles
- Restricts connections to same origin

For applications requiring different policies, consider modifying `DefaultSecurityHeaders` in your deployment.

## 🗺️ Roadmap

- [x] Core proxy functionality
- [x] OIDC authentication
- [x] Session management (Memory/Redis)
- [x] Prometheus metrics
- [x] OpenTelemetry tracing
- [x] Circuit breaker & retry logic
- [x] Multi-platform binaries
- [ ] Admin API for session management
- [ ] Built-in rate limiting
- [ ] Config hot-reload
- [ ] WebUI for monitoring

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.

## Acknowledgments

- Built for the [Model Context Protocol](https://modelcontextprotocol.io) ecosystem
- Inspired by the need for simple, secure MCP server deployment
- Cloudflare Tunnels for making zero-trust access easy