# MCP OIDC Proxy - Nginx/Lua Implementation (Legacy)

⚠️ **This implementation has been superseded by the Go implementation.** Please use the [Go version](../../) for new deployments.

This directory contains the original Nginx/Lua-based implementation of the MCP OIDC Proxy. While functional, it requires Docker and has more complex dependencies compared to the single-binary Go implementation.

## Components

- **OpenResty 1.27.1.2**: NGINX with Lua support
- **Redis 7**: Session storage
- **lua-resty-openidc 1.8.0**: OIDC authentication library
- **lua-resty-session 4.0.5**: Session management

## Quick Start (Docker)

```bash
# Start the proxy
./start-mcp.sh

# Configure target MCP server
MCP_TARGET_HOST=your-mcp-server.com MCP_TARGET_PORT=3000 docker compose up -d
```

## Configuration

### Auth0
```bash
AUTH0_DOMAIN=your-domain.auth0.com \
AUTH0_CLIENT_ID=your-client-id \
AUTH0_CLIENT_SECRET=your-client-secret \
AUTH_MODE=oidc ./start-mcp.sh
```

### Google
```bash
OIDC_DISCOVERY_URL="https://accounts.google.com/.well-known/openid-configuration" \
OIDC_CLIENT_ID="your-google-client-id" \
OIDC_CLIENT_SECRET="your-google-client-secret" \
AUTH_MODE=oidc ./start-mcp.sh
```

## Why This Was Replaced

1. **Deployment Complexity**: Requires Docker, Redis, and multiple containers
2. **Performance**: Go implementation has lower latency and memory usage
3. **Maintenance**: Single binary is easier to update and deploy
4. **Dependencies**: Lua libraries have slower update cycles
5. **Debugging**: Go provides better tooling and observability

## Migration Guide

To migrate from Nginx to Go implementation:

1. **Install Go version**:
   ```bash
   curl -sSL https://raw.githubusercontent.com/sh03m2a5h/mcp-oidc-proxy/main/install.sh | bash
   ```

2. **Use same environment variables**:
   - `OIDC_*` variables work identically
   - `AUTH0_*` legacy variables are still supported
   - `MCP_TARGET_*` → `PROXY_TARGET_*`

3. **Stop Nginx version**:
   ```bash
   ./stop-mcp.sh
   ```

4. **Start Go version**:
   ```bash
   mcp-oidc-proxy
   ```

## Files

- `nginx/`: Nginx configuration files
- `nginx/lua/oidc.lua`: OIDC authentication logic
- `docker-compose.yml`: Container orchestration
- `start-mcp.sh` / `stop-mcp.sh`: Helper scripts

## Support

This implementation is no longer actively maintained. Please use the Go implementation for:
- Bug fixes
- New features
- Security updates
- Production deployments