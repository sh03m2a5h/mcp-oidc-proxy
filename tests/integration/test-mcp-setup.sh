#!/bin/bash

# Test setup for MCP OIDC Proxy with mcp-proxy and mcp-server-fetch

echo "=== MCP OIDC Proxy Test Setup ==="
echo

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Step 1: Start mcp-proxy with mcp-server-fetch
echo -e "${YELLOW}1. Starting mcp-proxy with mcp-server-fetch...${NC}"
echo "Command: mcp-proxy --transport stdio uvx mcp-server-fetch"
echo

# Create a simple supervisor script
cat > run-mcp-proxy.sh << 'EOF'
#!/bin/bash
# Run mcp-proxy with mcp-server-fetch
exec mcp-proxy --port 3000 --transport stdio uvx mcp-server-fetch
EOF
chmod +x run-mcp-proxy.sh

echo -e "${GREEN}✓ Created run-mcp-proxy.sh${NC}"
echo

# Step 2: Build and run OIDC proxy
echo -e "${YELLOW}2. Building MCP OIDC Proxy...${NC}"
cd go
make build
cd ..
echo -e "${GREEN}✓ Build complete${NC}"
echo

# Step 3: Show test commands
echo -e "${YELLOW}3. Test Commands:${NC}"
echo
echo "Terminal 1 - Start MCP server:"
echo -e "${GREEN}./run-mcp-proxy.sh${NC}"
echo
echo "Terminal 2 - Start OIDC Proxy (bypass mode for testing):"
echo -e "${GREEN}AUTH_MODE=bypass ./go/bin/mcp-oidc-proxy${NC}"
echo
echo "Terminal 3 - Test the setup:"
echo -e "${GREEN}# Test health check${NC}"
echo "curl http://localhost:8080/health"
echo
echo -e "${GREEN}# Test MCP server through proxy${NC}"
echo "curl -X POST http://localhost:8080/ -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"2025-03-26\",\"capabilities\":{}},\"id\":1}'"
echo
echo -e "${GREEN}# Test with OIDC authentication:${NC}"
echo "export AUTH_MODE=oidc"
echo "export OIDC_DISCOVERY_URL=https://your-domain.auth0.com/.well-known/openid-configuration"
echo "export OIDC_CLIENT_ID=your-client-id"
echo "export OIDC_CLIENT_SECRET=your-client-secret"
echo "./go/bin/mcp-oidc-proxy"
echo

# Step 4: Create test client
cat > test-mcp-client.py << 'EOF'
#!/usr/bin/env python3
"""Simple MCP client to test the proxy"""

import requests
import json

PROXY_URL = "http://localhost:8080"

def test_initialize():
    """Test MCP initialize method"""
    payload = {
        "jsonrpc": "2.0",
        "method": "initialize",
        "params": {
            "protocolVersion": "2025-03-26",
            "capabilities": {}
        },
        "id": 1
    }
    
    response = requests.post(PROXY_URL, json=payload)
    print(f"Status: {response.status_code}")
    print(f"Response: {json.dumps(response.json(), indent=2)}")

def test_list_tools():
    """Test listing available tools"""
    payload = {
        "jsonrpc": "2.0",
        "method": "list/tools",
        "params": {},
        "id": 2
    }
    
    response = requests.post(PROXY_URL, json=payload)
    print(f"Status: {response.status_code}")
    print(f"Response: {json.dumps(response.json(), indent=2)}")

if __name__ == "__main__":
    print("Testing MCP OIDC Proxy...")
    print("=" * 50)
    
    print("\n1. Testing initialize:")
    test_initialize()
    
    print("\n2. Testing list/tools:")
    test_list_tools()
EOF
chmod +x test-mcp-client.py

echo -e "${GREEN}✓ Created test-mcp-client.py${NC}"
echo
echo -e "${YELLOW}Setup complete! Follow the test commands above to verify the integration.${NC}"