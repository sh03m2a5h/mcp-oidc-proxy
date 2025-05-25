#!/usr/bin/env python3
"""
Test MCP SSE connection through OIDC proxy
"""
import json
import requests
import sseclient

def test_mcp_sse():
    """Test MCP SSE connection"""
    print("Testing MCP SSE through OIDC proxy...")
    
    # Connect to SSE endpoint
    headers = {'Accept': 'text/event-stream'}
    response = requests.get('http://localhost:8090/mcp/', headers=headers, stream=True)
    
    print(f"Status: {response.status_code}")
    print(f"Headers: {dict(response.headers)}")
    
    if response.status_code == 200:
        # Create SSE client
        client = sseclient.SSEClient(response)
        
        print("\nListening for SSE events...")
        for event in client.events():
            print(f"Event: {event.event}")
            print(f"Data: {event.data}")
            
            # Parse and pretty print JSON data
            try:
                data = json.loads(event.data)
                print(f"Parsed: {json.dumps(data, indent=2)}")
            except:
                pass
            
            # Break after first few events for testing
            if event.event == 'message':
                break

if __name__ == "__main__":
    test_mcp_sse()
