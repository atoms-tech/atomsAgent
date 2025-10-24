#!/usr/bin/env python3
"""
FastMCP Client Wrapper for AgentAPI
Provides a Python-based MCP client that can be called from Go
"""

import asyncio
import json
import sys
from typing import Dict, List, Any, Optional
from dataclasses import dataclass
from fastmcp import FastMCPClient
from fastmcp.client.auth import BearerAuth, OAuthAuth
from fastmcp.client.transports import HTTPTransport, SSETransport, StdioTransport


@dataclass
class MCPConfig:
    """Configuration for an MCP server"""
    id: str
    name: str
    type: str  # http, sse, stdio
    endpoint: str
    auth_type: str  # bearer, oauth, none
    config: Dict[str, Any]
    auth: Dict[str, str]


class FastMCPWrapper:
    """Wrapper around FastMCP client for AgentAPI integration"""
    
    def __init__(self):
        self.clients: Dict[str, FastMCPClient] = {}
    
    async def connect_mcp(self, config: MCPConfig) -> bool:
        """Connect to an MCP server"""
        try:
            # Create transport based on type
            if config.type == "http":
                transport = HTTPTransport(config.endpoint)
            elif config.type == "sse":
                transport = SSETransport(config.endpoint)
            elif config.type == "stdio":
                command = config.config.get("command", "").split()
                transport = StdioTransport(command)
            else:
                raise ValueError(f"Unsupported MCP type: {config.type}")
            
            # Create authentication
            auth = None
            if config.auth_type == "bearer":
                token = config.auth.get("token", "")
                auth = BearerAuth(token)
            elif config.auth_type == "oauth":
                # OAuth configuration
                client_id = config.auth.get("client_id", "")
                client_secret = config.auth.get("client_secret", "")
                auth_url = config.auth.get("auth_url", "")
                token_url = config.auth.get("token_url", "")
                auth = OAuthAuth(client_id, client_secret, auth_url, token_url)
            
            # Create client
            client = FastMCPClient(
                name=config.name,
                version="1.0.0",
                transport=transport,
                auth=auth
            )
            
            # Connect
            await client.connect()
            self.clients[config.id] = client
            return True
            
        except Exception as e:
            print(f"Failed to connect to MCP {config.name}: {e}", file=sys.stderr)
            return False
    
    async def disconnect_mcp(self, mcp_id: str) -> bool:
        """Disconnect from an MCP server"""
        try:
            if mcp_id in self.clients:
                await self.clients[mcp_id].disconnect()
                del self.clients[mcp_id]
            return True
        except Exception as e:
            print(f"Failed to disconnect MCP {mcp_id}: {e}", file=sys.stderr)
            return False
    
    async def list_tools(self, mcp_id: str) -> List[Dict[str, Any]]:
        """List available tools from an MCP server"""
        if mcp_id not in self.clients:
            return []
        
        try:
            tools = await self.clients[mcp_id].list_tools()
            return [tool.dict() for tool in tools]
        except Exception as e:
            print(f"Failed to list tools for MCP {mcp_id}: {e}", file=sys.stderr)
            return []
    
    async def call_tool(self, mcp_id: str, tool_name: str, arguments: Dict[str, Any]) -> Dict[str, Any]:
        """Call a tool on an MCP server"""
        if mcp_id not in self.clients:
            return {"error": "MCP client not connected"}
        
        try:
            result = await self.clients[mcp_id].call_tool(tool_name, arguments)
            return result.dict()
        except Exception as e:
            return {"error": str(e)}
    
    async def list_resources(self, mcp_id: str) -> List[Dict[str, Any]]:
        """List available resources from an MCP server"""
        if mcp_id not in self.clients:
            return []
        
        try:
            resources = await self.clients[mcp_id].list_resources()
            return [resource.dict() for resource in resources]
        except Exception as e:
            print(f"Failed to list resources for MCP {mcp_id}: {e}", file=sys.stderr)
            return []
    
    async def read_resource(self, mcp_id: str, uri: str) -> Dict[str, Any]:
        """Read a resource from an MCP server"""
        if mcp_id not in self.clients:
            return {"error": "MCP client not connected"}
        
        try:
            result = await self.clients[mcp_id].read_resource(uri)
            return result.dict()
        except Exception as e:
            return {"error": str(e)}
    
    async def list_prompts(self, mcp_id: str) -> List[Dict[str, Any]]:
        """List available prompts from an MCP server"""
        if mcp_id not in self.clients:
            return []
        
        try:
            prompts = await self.clients[mcp_id].list_prompts()
            return [prompt.dict() for prompt in prompts]
        except Exception as e:
            print(f"Failed to list prompts for MCP {mcp_id}: {e}", file=sys.stderr)
            return []
    
    async def get_prompt(self, mcp_id: str, prompt_name: str, arguments: Dict[str, Any]) -> Dict[str, Any]:
        """Get a prompt from an MCP server"""
        if mcp_id not in self.clients:
            return {"error": "MCP client not connected"}
        
        try:
            result = await self.clients[mcp_id].get_prompt(prompt_name, arguments)
            return result.dict()
        except Exception as e:
            return {"error": str(e)}


async def main():
    """Main entry point for the FastMCP wrapper"""
    wrapper = FastMCPWrapper()
    
    # Read command from stdin
    try:
        line = sys.stdin.readline()
        if not line:
            return
        
        command = json.loads(line.strip())
        action = command.get("action")
        
        if action == "connect":
            config_data = command.get("config", {})
            config = MCPConfig(**config_data)
            success = await wrapper.connect_mcp(config)
            print(json.dumps({"success": success}))
        
        elif action == "disconnect":
            mcp_id = command.get("mcp_id")
            success = await wrapper.disconnect_mcp(mcp_id)
            print(json.dumps({"success": success}))
        
        elif action == "list_tools":
            mcp_id = command.get("mcp_id")
            tools = await wrapper.list_tools(mcp_id)
            print(json.dumps({"tools": tools}))
        
        elif action == "call_tool":
            mcp_id = command.get("mcp_id")
            tool_name = command.get("tool_name")
            arguments = command.get("arguments", {})
            result = await wrapper.call_tool(mcp_id, tool_name, arguments)
            print(json.dumps(result))
        
        elif action == "list_resources":
            mcp_id = command.get("mcp_id")
            resources = await wrapper.list_resources(mcp_id)
            print(json.dumps({"resources": resources}))
        
        elif action == "read_resource":
            mcp_id = command.get("mcp_id")
            uri = command.get("uri")
            result = await wrapper.read_resource(mcp_id, uri)
            print(json.dumps(result))
        
        elif action == "list_prompts":
            mcp_id = command.get("mcp_id")
            prompts = await wrapper.list_prompts(mcp_id)
            print(json.dumps({"prompts": prompts}))
        
        elif action == "get_prompt":
            mcp_id = command.get("mcp_id")
            prompt_name = command.get("prompt_name")
            arguments = command.get("arguments", {})
            result = await wrapper.get_prompt(mcp_id, prompt_name, arguments)
            print(json.dumps(result))
        
        else:
            print(json.dumps({"error": f"Unknown action: {action}"}))
    
    except Exception as e:
        print(json.dumps({"error": str(e)}), file=sys.stderr)


if __name__ == "__main__":
    asyncio.run(main())