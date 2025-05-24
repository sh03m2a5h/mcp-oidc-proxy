# MCP OIDC Proxy - Go Implementation

A lightweight, high-performance OAuth 2.1/OIDC authentication proxy for Model Context Protocol (MCP) servers, implemented in Go.

## Features

- 🚀 **Single Binary**: Easy deployment with no dependencies
- 🔐 **Multiple OIDC Providers**: Auth0, Google, Microsoft, GitHub support
- 🛡️ **OAuth 2.1 + PKCE**: Modern security standards
- 📊 **Built-in Monitoring**: Prometheus metrics and health checks
- 🔄 **WebSocket Support**: Full MCP protocol compatibility
- ⚡ **High Performance**: <10ms latency, 1000+ concurrent connections

## Quick Start

### Download and Run

```bash
# Download the latest release
wget https://github.com/sh03m2a5h/mcp-oidc-proxy-go/releases/download/v1.0.0/mcp-oidc-proxy-linux-amd64
chmod +x mcp-oidc-proxy-linux-amd64

# Run with environment variables
MCP_TARGET_HOST=localhost \
MCP_TARGET_PORT=3000 \
OIDC_DISCOVERY_URL=https://your-domain.auth0.com/.well-known/openid-configuration \
OIDC_CLIENT_ID=your-client-id \
OIDC_CLIENT_SECRET=your-client-secret \
./mcp-oidc-proxy-linux-amd64
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

## Configuration

The proxy can be configured via:
1. Command-line flags
2. Environment variables
3. Configuration file (YAML)

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `MCP_HOST` | Listen address | `0.0.0.0` |
| `MCP_PORT` | Listen port | `8080` |
| `MCP_TARGET_HOST` | Target MCP server host | `localhost` |
| `MCP_TARGET_PORT` | Target MCP server port | `3000` |
| `AUTH_MODE` | Authentication mode (`oidc`, `bypass`) | `oidc` |
| `OIDC_DISCOVERY_URL` | OIDC discovery endpoint | - |
| `OIDC_CLIENT_ID` | OAuth client ID | - |
| `OIDC_CLIENT_SECRET` | OAuth client secret | - |
| `SESSION_STORE` | Session store (`memory`, `redis`) | `memory` |
| `REDIS_URL` | Redis connection URL | `redis://localhost:6379` |

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

## Deployment

### Docker

```bash
# Build Docker image
make docker

# Run with Docker Compose
docker-compose up -d
```

### Kubernetes

```bash
# Apply Kubernetes manifests
kubectl apply -f deployments/kubernetes/
```

### Systemd

```bash
# Copy binary
sudo cp bin/mcp-oidc-proxy /usr/local/bin/

# Install service
sudo cp deployments/systemd/mcp-oidc-proxy.service /etc/systemd/system/
sudo systemctl enable mcp-oidc-proxy
sudo systemctl start mcp-oidc-proxy
```

## Monitoring

### Health Check

```bash
curl http://localhost:8080/health
```

### Metrics

Prometheus metrics are available at `/metrics`:

```bash
curl http://localhost:8080/metrics
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.