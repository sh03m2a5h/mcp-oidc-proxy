# MCP OIDC Proxy

Production-ready OAuth 2.1/OIDC authentication proxy for Model Context Protocol (MCP) servers. A single Go binary that secures your MCP endpoints with modern authentication.

## 🚀 Quick Start

```bash
# Install (Linux/macOS)
curl -sSL https://raw.githubusercontent.com/sh03m2a5h/mcp-oidc-proxy/main/install.sh | bash

# Configure OIDC (example with Auth0)
export OIDC_DISCOVERY_URL="https://your-domain.auth0.com/.well-known/openid-configuration"
export OIDC_CLIENT_ID="your-client-id"
export OIDC_CLIENT_SECRET="your-client-secret"

# Run
mcp-oidc-proxy
```

Your MCP server at `localhost:3000` is now protected with OIDC authentication at `localhost:8080`!

## 🎯 What This Does

Adds enterprise-grade authentication to any MCP server:

```
[Internet] → [Cloudflare] → [MCP OIDC Proxy :8080] → [Your MCP Server :3000]
                                    ↓
                            [OIDC Provider]
                         (Auth0/Google/Azure)
```

## ✨ Features

- 🔐 **Universal OIDC Support**: Works with Auth0, Google, Microsoft, GitHub, or any OIDC provider
- 🚀 **Single Binary**: No Docker, no dependencies - just download and run
- 🛡️ **Modern Security**: OAuth 2.1 with PKCE, secure sessions, CSP headers
- 📊 **Production Ready**: Prometheus metrics, health checks, OpenTelemetry tracing
- 🔄 **Full Protocol Support**: HTTP, SSE/WebSocket streaming, and MCP protocols
- ⚡ **High Performance**: <10ms overhead, 1000+ concurrent connections

## 📦 Installation

### Binary Release (Recommended)
```bash
# One-line install
curl -sSL https://raw.githubusercontent.com/sh03m2a5h/mcp-oidc-proxy/main/install.sh | bash

# Or download directly
wget https://github.com/sh03m2a5h/mcp-oidc-proxy/releases/latest/download/mcp-oidc-proxy-$(uname -s)-$(uname -m)
chmod +x mcp-oidc-proxy-*
```

### From Source
```bash
git clone https://github.com/sh03m2a5h/mcp-oidc-proxy.git
cd mcp-oidc-proxy/go
make build
./bin/mcp-oidc-proxy
```

## 🔧 Configuration

### Auth0 (Recommended)
```bash
export OIDC_DISCOVERY_URL="https://YOUR-DOMAIN.auth0.com/.well-known/openid-configuration"
export OIDC_CLIENT_ID="your-client-id"
export OIDC_CLIENT_SECRET="your-client-secret"
export OIDC_REDIRECT_URL="http://localhost:8080/callback"
```

### Google
```bash
export OIDC_DISCOVERY_URL="https://accounts.google.com/.well-known/openid-configuration"
export OIDC_CLIENT_ID="your-client-id.apps.googleusercontent.com"
export OIDC_CLIENT_SECRET="your-client-secret"
```

### Microsoft Azure AD
```bash
export OIDC_DISCOVERY_URL="https://login.microsoftonline.com/YOUR-TENANT-ID/v2.0/.well-known/openid-configuration"
export OIDC_CLIENT_ID="your-client-id"
export OIDC_CLIENT_SECRET="your-client-secret"
```

## 🌐 Production Deployment

### With Cloudflare Tunnels
```bash
# Start proxy
./mcp-oidc-proxy &

# Create tunnel
cloudflared tunnel --url http://localhost:8080
```

### Systemd Service
```bash
# Download binary
sudo curl -L https://github.com/sh03m2a5h/mcp-oidc-proxy/releases/latest/download/mcp-oidc-proxy-linux-amd64 \
  -o /usr/local/bin/mcp-oidc-proxy
sudo chmod +x /usr/local/bin/mcp-oidc-proxy

# Create service
sudo tee /etc/systemd/system/mcp-oidc-proxy.service > /dev/null <<EOF
[Unit]
Description=MCP OIDC Proxy
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/mcp-oidc-proxy
Restart=always
Environment="OIDC_DISCOVERY_URL=https://your-domain.auth0.com/.well-known/openid-configuration"
Environment="OIDC_CLIENT_ID=your-client-id"
Environment="OIDC_CLIENT_SECRET=your-client-secret"

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl enable --now mcp-oidc-proxy
```

## 📊 Monitoring

```bash
# Health check
curl http://localhost:8080/health

# Prometheus metrics
curl http://localhost:8080/metrics
```

## 🔍 Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_PORT` | Listen port | `8080` |
| `PROXY_TARGET_HOST` | MCP server host | `localhost` |
| `PROXY_TARGET_PORT` | MCP server port | `3000` |
| `AUTH_MODE` | Auth mode (`oidc`, `bypass`) | `oidc` |
| `OIDC_DISCOVERY_URL` | OIDC discovery endpoint | Required |
| `OIDC_CLIENT_ID` | OAuth client ID | Required |
| `OIDC_CLIENT_SECRET` | OAuth client secret | Required |
| `SESSION_STORE` | Session store (`memory`, `redis`) | `memory` |
| `METRICS_ENABLED` | Enable Prometheus metrics | `true` |
| `LOG_LEVEL` | Log level | `info` |

## 📁 Project Structure

```
mcp-oidc-proxy/
├── go/                    # Go implementation (primary)
│   ├── cmd/              # Application entry point
│   ├── internal/         # Core application code
│   └── README.md         # Detailed Go documentation
├── legacy/               # Previous implementations
│   └── nginx/           # Nginx/Lua implementation (archived)
└── docs/                # Architecture documentation
```

## 🏗️ Architecture

The proxy is built with:
- **Language**: Go 1.23+
- **HTTP Framework**: Gin
- **OIDC Library**: coreos/go-oidc
- **Session Store**: In-memory or Redis
- **Metrics**: Prometheus
- **Tracing**: OpenTelemetry

See [docs/](docs/) for detailed architecture documentation.

## 🔄 Recent Updates

### v0.5.0 (Latest)
- **SSE/WebSocket Streaming Support**: Fixed panic issues with streaming protocols
- **Bypass Mode**: Added development/testing mode to bypass authentication
- **Improved Stability**: Better error handling for long-lived connections
- **AI-Assisted Development**: Code quality enhanced through Copilot and Gemini reviews

### v0.4.0
- **Monitoring & Observability**: Prometheus metrics and OpenTelemetry tracing
- **Health Checks**: Built-in health endpoint with subsystem status
- **Circuit Breaker**: Automatic backend failure protection

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📜 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built for the [Model Context Protocol](https://modelcontextprotocol.io) ecosystem
- Inspired by the need for simple, secure MCP server deployment
- SSE/WebSocket streaming support developed for [mcp-proxy](https://github.com/sparfenyuk/mcp-proxy) compatibility

---

### Legacy Implementation

The original Nginx/Lua implementation is available in the [legacy/nginx-implementation](legacy/nginx-implementation/) directory. The Go implementation is now the primary and recommended version.