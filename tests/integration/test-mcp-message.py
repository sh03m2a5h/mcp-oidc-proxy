#!/usr/bin/env python3
"""
Test sending MCP messages through OIDC proxy
"""
import json
import requests

def get_session_id():
    """Get a new session ID by making an initial request"""
    headers = {'Accept': 'text/event-stream'}
    response = requests.get('http://localhost:8090/mcp/', headers=headers)
    
    # Extract session ID from response headers
    session_id = response.headers.get('Mcp-Session-Id')
    if not session_id:
        raise ValueError("No session ID received from server")
    
    print(f"Got session ID: {session_id}")
    return session_id

def test_mcp_message():
    """Test sending an MCP message"""
    print("Testing MCP message through OIDC proxy...")
    
    # Get a fresh session ID
    try:
        session_id = get_session_id()
    except Exception as e:
        print(f"Failed to get session ID: {e}")
        return
    
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
