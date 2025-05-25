# Integration Tests

This directory contains integration tests for the MCP OIDC Proxy.

## Test Files

- `test-config.yaml` - Configuration for bypass mode testing
- `test-mcp-setup.sh` - Setup script for MCP test environment
- `run-mcp-proxy.sh` - Wrapper script to run mcp-proxy with mcp-server-fetch
- `test-mcp-client.py` - HTTP-based MCP client test (basic)
- `test-mcp-sse.py` - SSE-based MCP client test
- `test-mcp-message.py` - MCP message sending test
- `test-results.md` - Results from integration testing

## Prerequisites

1. Install mcp-proxy:
   ```bash
   pip install mcp-proxy
   ```

2. Install Python dependencies:
   ```bash
   pip install requests sseclient-py
   ```

## Running Tests

1. Start mcp-proxy in background:
   ```bash
   ./test-mcp-setup.sh
   ```

2. Start OIDC proxy in bypass mode:
   ```bash
   cd ../go && ./bin/mcp-oidc-proxy --config ../tests/integration/test-config.yaml
   ```

3. Run individual tests:
   ```bash
   python3 test-mcp-client.py
   python3 test-mcp-sse.py
   python3 test-mcp-message.py
   ```

## Test Results

See `test-results.md` for detailed test results and findings.

## Known Issues

- SSE streaming causes panics in the current proxy implementation
- WebSocket support is untested
- The proxy works well for standard HTTP/JSON-RPC APIs