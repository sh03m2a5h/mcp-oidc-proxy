#!/usr/bin/env python3
"""
Test sending MCP messages through OIDC proxy
"""
import json
import requests

def test_mcp_message():
    """Test sending an MCP message"""
    print("Testing MCP message through OIDC proxy...")
    
    # Use the session ID from previous request
    session_id = "b9bed4179ccc486ba92b669df7c9905c"
    
    # Initialize message
    message = {
        "jsonrpc": "2.0",
        "method": "initialize",
        "params": {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {
                "name": "test-client",
                "version": "1.0.0"
            }
        },
        "id": 1
    }
    
    # Send message to the session
    headers = {
        'Content-Type': 'application/json',
        'Mcp-Session-Id': session_id
    }
    
    response = requests.post(
        'http://localhost:8090/mcp/',
        headers=headers,
        json=message
    )
    
    print(f"Status: {response.status_code}")
    print(f"Response: {response.text}")

if __name__ == "__main__":
    test_mcp_message()