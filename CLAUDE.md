# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview
This is a generic OAuth 2.1/OIDC authentication proxy for MCP (Model Context Protocol) servers. It provides authentication layer without running MCP servers itself, supporting multiple OIDC providers including Auth0, Google, Microsoft, and GitHub.

## Build Commands
- Start proxy: `./start-mcp.sh`
- Stop proxy: `./stop-mcp.sh`
- View logs: `docker logs mcp-proxy`
- Health check: `curl http://localhost:8080/health`

## Configuration Examples

### Auth0 (Recommended)
```bash
# Modern OIDC configuration
OIDC_DISCOVERY_URL="https://your-domain.auth0.com/.well-known/openid-configuration" \
OIDC_CLIENT_ID="your-client-id" \
OIDC_CLIENT_SECRET="your-secret" \
AUTH_MODE=oidc ./start-mcp.sh

# Legacy Auth0 configuration (still supported)
AUTH0_DOMAIN=your-domain.auth0.com \
AUTH0_CLIENT_ID=your-client-id \
AUTH0_CLIENT_SECRET=your-secret \
AUTH_MODE=oidc ./start-mcp.sh
```

### Other Providers
```bash
# Google
OIDC_DISCOVERY_URL="https://accounts.google.com/.well-known/openid-configuration"

# Microsoft Azure AD
OIDC_DISCOVERY_URL="https://login.microsoftonline.com/{tenant}/v2.0/.well-known/openid-configuration"
```

### Target Configuration
```bash
# Set MCP target server
MCP_TARGET_HOST=your-mcp-server.com MCP_TARGET_PORT=3000 docker compose up -d
```

### External Access
```bash
# Cloudflare Tunnel
cloudflared tunnel --url http://localhost:8080

# Ngrok
ngrok http 8080
```

## Code Style Guidelines
- Lua indentation: 4 spaces
- Config file formatting: Follow existing style in nginx/conf.d/default.conf
- Error handling: Log errors with appropriate log level (ngx.ERR, ngx.DEBUG)
- Naming conventions: Use snake_case for variables and functions
- Docker/Compose: Use version '3' format, with named containers and networks
- Shell scripts: Include descriptive comments and echo status messages
- Security: Never commit OAuth client credentials, always use environment variables
- Character encoding: Use UTF-8 for all files

## Project Structure
- nginx/conf.d/: NGINX configuration files
- nginx/lua/oidc.lua: Generic OIDC authentication module
- data/: Persistent storage (Redis data)
- docker-compose.yml: Main proxy service configuration
- start-mcp.sh/stop-mcp.sh: Service management scripts

## Key Features
- **Generic OIDC**: Supports Auth0, Google, Microsoft, GitHub, and any OIDC-compliant provider
- **OAuth 2.1 + PKCE**: Modern security standards with Proof Key for Code Exchange
- **Lightweight**: Only authentication proxy, no MCP server execution
- **Flexible**: Works with any external MCP server
- **Scalable**: Redis-based session management
- **Simple**: Easy external publishing via tunnels
- **Backward Compatible**: Legacy Auth0 environment variables still supported

## Authentication Flow
1. Client accesses proxy (http://localhost:8080)
2. If AUTH_MODE=oidc, redirect to OIDC provider
3. User authenticates with provider (Auth0/Google/etc)
4. Provider redirects back to /callback with authorization code
5. Proxy exchanges code for tokens using PKCE
6. Session stored in Redis
7. Subsequent requests include user headers (X-USER, X-EMAIL, etc)
8. Requests proxied to target MCP server

## Environment Variables Priority
1. Generic OIDC variables (OIDC_*) take precedence
2. Legacy Auth0 variables (AUTH0_*) used as fallback
3. If AUTH0_DOMAIN is set, it's automatically converted to OIDC_DISCOVERY_URL

## Documentation
- Keep README.md updated when making significant changes
- Document configuration parameters in comments
- Include usage examples for common OIDC providers
- Maintain backward compatibility notes