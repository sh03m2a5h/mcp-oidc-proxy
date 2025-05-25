#\!/usr/bin/env python3
"""
Simple test client for MCP OIDC Proxy with mcp-server-fetch
"""
import json
import requests
from typing import Dict, Any

class MCPTestClient:
    def __init__(self, base_url: str = "http://localhost:8090"):
        self.base_url = base_url
        self.session_id = None
        
    def create_session(self) -> str:
        """Create a new MCP session"""
        response = requests.post(f"{self.base_url}/sse/sessions")
        response.raise_for_status()
        data = response.json()
        self.session_id = data['sessionId']
        print(f"Created session: {self.session_id}")
        return self.session_id
    
    def send_message(self, message: Dict[str, Any]) -> Dict[str, Any]:
        """Send a message to the MCP server"""
        if not self.session_id:
            raise ValueError("No session created")
            
        response = requests.post(
            f"{self.base_url}/sse/sessions/{self.session_id}/messages",
            json=message
        )
        response.raise_for_status()
        return response.json()
    
    def initialize(self) -> Dict[str, Any]:
        """Initialize MCP connection"""
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
        return self.send_message(message)
    
    def list_tools(self) -> Dict[str, Any]:
        """List available tools"""
        message = {
            "jsonrpc": "2.0",
            "method": "tools/list",
            "params": {},
            "id": 2
        }
        return self.send_message(message)
    
    def fetch_url(self, url: str) -> Dict[str, Any]:
        """Use the fetch tool to get a URL"""
        message = {
            "jsonrpc": "2.0",
            "method": "tools/call",
            "params": {
                "name": "fetch",
                "arguments": {
                    "url": url
                }
            },
            "id": 3
        }
        return self.send_message(message)

def main():
    print("Testing MCP OIDC Proxy with mcp-server-fetch")
    print("=" * 50)
    
    client = MCPTestClient()
    
    try:
        # Create session
        client.create_session()
        
        # Initialize
        print("\nInitializing MCP...")
        init_response = client.initialize()
        print(f"Initialize response: {json.dumps(init_response, indent=2)}")
        
        # List tools
        print("\nListing available tools...")
        tools_response = client.list_tools()
        print(f"Tools: {json.dumps(tools_response, indent=2)}")
        
        # Test fetch
        print("\nFetching example.com...")
        fetch_response = client.fetch_url("https://example.com")
        print(f"Fetch response: {json.dumps(fetch_response, indent=2)[:500]}...")
        
    except Exception as e:
        print(f"Error: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    main()
