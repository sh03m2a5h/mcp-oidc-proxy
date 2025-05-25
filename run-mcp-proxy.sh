#!/bin/bash
# Run mcp-proxy with mcp-server-fetch
exec mcp-proxy --port 3000 --transport stdio uvx mcp-server-fetch
