# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview
MCP OIDC Proxy is a production-ready OAuth 2.1/OIDC authentication proxy for Model Context Protocol (MCP) servers. It's implemented as a single Go binary that provides authentication without requiring Docker or complex dependencies.

## Project Structure
```
mcp-oidc-proxy/
├── go/                    # Main Go implementation
│   ├── cmd/              # Application entry point
│   ├── internal/         # Core application code
│   │   ├── app/         # Application setup and routing
│   │   ├── auth/        # Authentication (OIDC, bypass)
│   │   ├── config/      # Configuration management
│   │   ├── middleware/  # HTTP middleware (security, metrics, logging)
│   │   ├── proxy/       # Reverse proxy with circuit breaker
│   │   ├── server/      # HTTP server
│   │   ├── session/     # Session management (memory/Redis)
│   │   └── tracing/     # OpenTelemetry tracing
│   └── pkg/             # Public packages
├── legacy/              # Old Nginx/Lua implementation (archived)
└── docs/               # Architecture documentation
```

## Key Commands
```bash
# Development
cd go && make build      # Build binary
cd go && make test       # Run tests
cd go && make run        # Run locally

# Production
./mcp-oidc-proxy         # Run with environment variables
```

## Configuration
The proxy is configured via environment variables:
- `OIDC_DISCOVERY_URL`: OIDC provider discovery endpoint
- `OIDC_CLIENT_ID`: OAuth client ID
- `OIDC_CLIENT_SECRET`: OAuth client secret
- `PROXY_TARGET_HOST`: Target MCP server (default: localhost)
- `PROXY_TARGET_PORT`: Target port (default: 3000)
- `AUTH_MODE`: Authentication mode (oidc/bypass)

## Typical Use Case
```bash
# Local MCP server protected by OIDC, exposed via Cloudflare Tunnel
./mcp-oidc-proxy &
cloudflared tunnel --url http://localhost:8080
```

## Code Style Guidelines
- Go standard formatting (gofmt)
- Meaningful variable names
- Comprehensive error handling with zap logger
- Test coverage target: 80%+
- Use interfaces for testability
- Security headers on all responses

## Security Considerations
- Always use PKCE for OAuth flows
- Session cookies are httpOnly
- CSP headers prevent XSS
- Circuit breaker protects backend
- Structured logging (no secrets in logs)

## Testing
- Unit tests for each package
- Integration tests for session stores
- Use testify for assertions
- Mock external dependencies

## Common Tasks
1. **Add new OIDC provider**: Update documentation, test discovery URL
2. **Change default headers**: Modify `middleware.DefaultSecurityHeaders`
3. **Add metrics**: Update `internal/metrics/metrics.go`
4. **Debug auth flow**: Set `LOG_LEVEL=debug`

## Important Notes
- The Nginx/Lua implementation in `legacy/` is deprecated
- Always test with real OIDC providers before release
- Cloudflare Tunnels is the recommended deployment method
- Binary releases are automated via GitHub Actions on tag push

## Pull Request Guidelines

### Creating Pull Requests
1. **Branch Naming**: Use descriptive names like `feature/sse-streaming`, `fix/session-leak`, `docs/api-update`
2. **PR Title**: Clear, concise, imperative mood (e.g., "Add SSE streaming support", not "Added SSE streaming")
3. **PR Description**: Include:
   - Summary of changes
   - Link to related issues with `Closes #123`
   - Test plan or validation steps
   - Breaking changes if any

### AI Code Review Setup

#### GitHub Copilot Code Review
1. **Enable Review**:
   - Select "Copilot" from the Reviewers dropdown on your PR
   - Or configure automatic reviews: Repository Settings → Rules → "Request pull request review from Copilot"
2. **Best Practices**:
   - Keep PRs small and focused (under 500 lines ideal)
   - Ensure CI passes before requesting review
   - Use `copilot:summary` in PR comments for summaries
   - Use `copilot:walkthrough` for detailed explanations

#### Gemini Code Assist
1. **Setup**:
   - Install from GitHub Marketplace
   - Reviews automatically within 5 minutes of PR creation
2. **Commands**:
   - `@gemini-code-assist review` - Request review
   - `/gemini explain` - Ask for explanations
   - `/gemini suggest` - Get improvement suggestions
3. **Configuration** (optional):
   ```yaml
   # .gemini/config.yaml
   disable: false
   comment_severity_threshold: MEDIUM
   max_review_comments: -1
   ```

### PR Review Workflow
1. Create feature branch and push changes
2. Open PR with clear description
3. Wait for CI to pass
4. Request AI reviews (Copilot and/or Gemini)
5. Address feedback with new commits (don't force-push during review)
6. Re-request reviews after significant changes
7. Squash merge when approved

### Example PR Description
```markdown
## Summary
Add dedicated SSE/WebSocket streaming support to fix panic with ResponseRecorder

## Changes
- Added streaming detection for SSE and WebSocket requests
- Implemented direct proxy handlers bypassing ResponseRecorder
- Added streaming-specific metrics
- Comprehensive test coverage

## Testing
- Unit tests pass
- Integration test with mcp-proxy confirms no panic
- Manual testing with SSE client successful

Closes #15
```
