# Integration Tests

This directory contains integration tests for the MCP OIDC Proxy, including both automated functional tests and manual OAuth2 testing procedures.

## Test Categories

### 1. Automated Integration Tests
Basic functional testing with bypass mode authentication.

### 2. OAuth2 Manual Testing
Comprehensive OAuth2/OIDC provider testing for production environments.

## Test Files

### Functional Tests
- `test-config.yaml` - Configuration for bypass mode testing
- `test-mcp-setup.sh` - Setup script for MCP test environment
- `run-mcp-proxy.sh` - Wrapper script to run mcp-proxy with mcp-server-fetch
- `test-mcp-client.py` - HTTP-based MCP client test (basic)
- `test-mcp-sse.py` - SSE-based MCP client test
- `test-mcp-message.py` - MCP message sending test
- `test-results.md` - Results from integration testing

### OAuth2 Testing
- `oauth2-testing-guide.md` - Comprehensive OAuth2 testing procedures
- `oauth2-manual-test.sh` - Automated test execution helper script
- `config-google.yaml` - Google OAuth2 configuration template
- `config-azure.yaml` - Azure AD configuration template
- `config-auth0.yaml` - Auth0 configuration template

## Prerequisites

### For Functional Tests
1. Install mcp-proxy:
   ```bash
   pip install mcp-proxy
   ```

2. Install Python dependencies:
   ```bash
   pip install requests sseclient-py
   ```

### For OAuth2 Testing
1. Go binary built:
   ```bash
   cd ../../go && make build
   ```

2. OIDC provider credentials configured
3. Browser access for authentication flows

## Running Tests

### Functional Integration Tests

1. Start mcp-proxy in background:
   ```bash
   ./test-mcp-setup.sh
   ```

2. Start OIDC proxy in bypass mode:
   ```bash
   cd ../../go && ./bin/mcp-oidc-proxy --config ../tests/integration/test-config.yaml
   ```

3. Run individual tests:
   ```bash
   python3 test-mcp-client.py
   python3 test-mcp-sse.py
   python3 test-mcp-message.py
   ```

### OAuth2 Manual Testing

1. **Quick Test with Helper Script:**
   ```bash
   # Interactive test selection
   ./oauth2-manual-test.sh

   # Specific provider test
   ./oauth2-manual-test.sh google

   # Automated test suite
   ./oauth2-manual-test.sh all
   ```

2. **Manual Step-by-Step Testing:**
   ```bash
   # Copy and configure provider template
   cp config-google.yaml my-config.yaml
   # Edit with your OAuth2 credentials

   # Start proxy
   cd ../../go && ./bin/mcp-oidc-proxy --config ../tests/integration/my-config.yaml

   # Follow test procedures in oauth2-testing-guide.md
   ```

## OAuth2 Testing Scenarios

The OAuth2 testing guide covers:

### Core Authentication Flows
- ✅ Authorization Code Flow with PKCE
- ✅ Token validation and refresh
- ✅ Logout and session cleanup
- ✅ State parameter validation
- ✅ Nonce validation

### Security Testing
- 🔒 CSRF protection validation
- 🔒 Token leakage prevention
- 🔒 Session hijacking prevention
- 🔒 XSS protection headers

### Provider Compatibility
- 🌐 Google OAuth2
- 🌐 Microsoft Azure AD
- 🌐 Auth0
- 🌐 Keycloak
- 🌐 Generic OIDC providers

### Error Handling
- ❌ Invalid credentials
- ❌ Network failures
- ❌ Token expiration
- ❌ Provider unavailability

## Test Results

- See `test-results.md` for functional test results
- OAuth2 test results are logged during execution
- Security test findings documented in testing guide

## Known Issues

### Functional Tests
- ~~SSE streaming causes panics~~ ✅ Fixed in streaming support update
- WebSocket support tested and working
- The proxy works well for all MCP transport protocols

### OAuth2 Testing
- Manual testing required due to provider-specific setups
- Browser automation not implemented for OAuth flows
- Provider rate limits may affect rapid testing

## Contributing

When adding new tests:
1. Update this README with test descriptions
2. Follow existing naming conventions
3. Add provider-specific configs to templates
4. Document any new prerequisites or setup steps
