#!/usr/bin/env python3
"""
Example usage of FastMCP Service
Demonstrates how to interact with the FastMCP service from Python
"""

import asyncio
import httpx
import json
from typing import Dict, Any


class FastMCPServiceClient:
    """Client for interacting with FastMCP Service"""

    def __init__(self, base_url: str = "http://localhost:8080"):
        self.base_url = base_url
        self.client_id: str = None

    async def connect(
        self,
        transport: str = "http",
        mcp_url: str = None,
        command: list = None,
        auth_type: str = "none",
        bearer_token: str = None,
        name: str = "example-client"
    ) -> Dict[str, Any]:
        """Connect to an MCP server"""
        async with httpx.AsyncClient() as client:
            request_data = {
                "transport": transport,
                "auth_type": auth_type,
                "name": name
            }

            if transport in ["http", "sse"]:
                if not mcp_url:
                    raise ValueError("mcp_url is required for http/sse transports")
                request_data["mcp_url"] = mcp_url

            if transport == "stdio":
                if not command:
                    raise ValueError("command is required for stdio transport")
                request_data["command"] = command

            if auth_type == "bearer":
                if not bearer_token:
                    raise ValueError("bearer_token is required for bearer auth")
                request_data["bearer_token"] = bearer_token

            response = await client.post(
                f"{self.base_url}/mcp/connect",
                json=request_data
            )
            response.raise_for_status()

            data = response.json()
            self.client_id = data["client_id"]

            print(f"✓ Connected to MCP server")
            print(f"  Client ID: {self.client_id}")
            print(f"  Available tools: {len(data.get('tools', []))}")
            print(f"  Available resources: {len(data.get('resources', []))}")
            print(f"  Available prompts: {len(data.get('prompts', []))}")

            return data

    async def list_tools(self) -> Dict[str, Any]:
        """List available tools"""
        if not self.client_id:
            raise ValueError("Not connected. Call connect() first.")

        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{self.base_url}/mcp/list_tools",
                params={"client_id": self.client_id}
            )
            response.raise_for_status()
            data = response.json()

            print(f"\n✓ Tools available: {data['count']}")
            for tool in data['tools']:
                print(f"  - {tool.get('name', 'unknown')}: {tool.get('description', 'No description')}")

            return data

    async def call_tool(
        self,
        tool_name: str,
        arguments: Dict[str, Any] = None,
        timeout: int = 60
    ) -> Dict[str, Any]:
        """Call a tool"""
        if not self.client_id:
            raise ValueError("Not connected. Call connect() first.")

        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{self.base_url}/mcp/call_tool",
                json={
                    "client_id": self.client_id,
                    "tool_name": tool_name,
                    "arguments": arguments or {},
                    "timeout": timeout
                }
            )
            response.raise_for_status()
            data = response.json()

            print(f"\n✓ Tool '{tool_name}' executed")
            print(f"  Status: {data['status']}")
            print(f"  Execution time: {data.get('execution_time', 0):.3f}s")

            if data['status'] == 'success':
                print(f"  Result: {json.dumps(data['result'], indent=2)}")
            else:
                print(f"  Error: {data.get('error', 'Unknown error')}")

            return data

    async def list_resources(self) -> Dict[str, Any]:
        """List available resources"""
        if not self.client_id:
            raise ValueError("Not connected. Call connect() first.")

        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{self.base_url}/mcp/list_resources",
                params={"client_id": self.client_id}
            )
            response.raise_for_status()
            data = response.json()

            print(f"\n✓ Resources available: {data['count']}")
            for resource in data['resources']:
                print(f"  - {resource.get('uri', 'unknown')}: {resource.get('name', 'No name')}")

            return data

    async def read_resource(self, uri: str) -> Dict[str, Any]:
        """Read a resource"""
        if not self.client_id:
            raise ValueError("Not connected. Call connect() first.")

        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{self.base_url}/mcp/read_resource",
                json={
                    "client_id": self.client_id,
                    "uri": uri
                }
            )
            response.raise_for_status()
            data = response.json()

            print(f"\n✓ Resource read: {uri}")
            print(f"  Content: {json.dumps(data['content'], indent=2)}")

            return data

    async def list_prompts(self) -> Dict[str, Any]:
        """List available prompts"""
        if not self.client_id:
            raise ValueError("Not connected. Call connect() first.")

        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{self.base_url}/mcp/list_prompts",
                params={"client_id": self.client_id}
            )
            response.raise_for_status()
            data = response.json()

            print(f"\n✓ Prompts available: {data['count']}")
            for prompt in data['prompts']:
                print(f"  - {prompt.get('name', 'unknown')}: {prompt.get('description', 'No description')}")

            return data

    async def get_prompt(
        self,
        prompt_name: str,
        arguments: Dict[str, Any] = None
    ) -> Dict[str, Any]:
        """Get a prompt"""
        if not self.client_id:
            raise ValueError("Not connected. Call connect() first.")

        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{self.base_url}/mcp/get_prompt",
                json={
                    "client_id": self.client_id,
                    "prompt_name": prompt_name,
                    "arguments": arguments or {}
                }
            )
            response.raise_for_status()
            data = response.json()

            print(f"\n✓ Prompt '{prompt_name}' retrieved")
            print(f"  Content: {json.dumps(data['content'], indent=2)}")

            return data

    async def disconnect(self) -> Dict[str, Any]:
        """Disconnect from MCP server"""
        if not self.client_id:
            raise ValueError("Not connected. Call connect() first.")

        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{self.base_url}/mcp/disconnect",
                json={"client_id": self.client_id}
            )
            response.raise_for_status()
            data = response.json()

            print(f"\n✓ Disconnected from MCP server")
            self.client_id = None

            return data

    async def health_check(self) -> Dict[str, Any]:
        """Check service health"""
        async with httpx.AsyncClient() as client:
            response = await client.get(f"{self.base_url}/health")
            response.raise_for_status()
            data = response.json()

            print(f"\n✓ Service Health Check")
            print(f"  Status: {data['status']}")
            print(f"  Active clients: {data['active_clients']}")
            print(f"  Version: {data['version']}")

            return data


async def example_http_connection():
    """Example: Connect to an HTTP MCP server"""
    print("\n" + "="*60)
    print("Example 1: HTTP Connection")
    print("="*60)

    client = FastMCPServiceClient()

    try:
        # Connect
        await client.connect(
            transport="http",
            mcp_url="http://localhost:3000/mcp",
            auth_type="none",
            name="http-example"
        )

        # List tools
        await client.list_tools()

        # Call a tool (if available)
        # await client.call_tool("example_tool", {"arg": "value"})

        # List resources
        await client.list_resources()

        # Disconnect
        await client.disconnect()

    except Exception as e:
        print(f"\n✗ Error: {str(e)}")


async def example_bearer_auth():
    """Example: Connect with bearer token authentication"""
    print("\n" + "="*60)
    print("Example 2: Bearer Token Authentication")
    print("="*60)

    client = FastMCPServiceClient()

    try:
        # Connect with bearer token
        await client.connect(
            transport="http",
            mcp_url="http://localhost:3000/mcp",
            auth_type="bearer",
            bearer_token="your-secret-token-here",
            name="bearer-auth-example"
        )

        # List tools
        await client.list_tools()

        # Disconnect
        await client.disconnect()

    except Exception as e:
        print(f"\n✗ Error: {str(e)}")


async def example_stdio_connection():
    """Example: Connect to a local MCP server via stdio"""
    print("\n" + "="*60)
    print("Example 3: STDIO Connection")
    print("="*60)

    client = FastMCPServiceClient()

    try:
        # Connect via stdio
        await client.connect(
            transport="stdio",
            command=["node", "/path/to/mcp-server.js"],
            auth_type="none",
            name="stdio-example"
        )

        # List tools
        await client.list_tools()

        # Disconnect
        await client.disconnect()

    except Exception as e:
        print(f"\n✗ Error: {str(e)}")


async def example_complete_workflow():
    """Example: Complete workflow with all operations"""
    print("\n" + "="*60)
    print("Example 4: Complete Workflow")
    print("="*60)

    client = FastMCPServiceClient()

    try:
        # Health check
        await client.health_check()

        # Connect
        await client.connect(
            transport="http",
            mcp_url="http://localhost:3000/mcp",
            auth_type="none",
            name="complete-workflow-example"
        )

        # List all available capabilities
        tools = await client.list_tools()
        resources = await client.list_resources()
        prompts = await client.list_prompts()

        # Example: Call a tool if available
        if tools['count'] > 0:
            first_tool = tools['tools'][0]
            tool_name = first_tool.get('name')
            print(f"\nCalling tool: {tool_name}")

            # Note: Replace with actual arguments based on tool schema
            # await client.call_tool(tool_name, {"example": "argument"})

        # Example: Read a resource if available
        if resources['count'] > 0:
            first_resource = resources['resources'][0]
            resource_uri = first_resource.get('uri')
            print(f"\nReading resource: {resource_uri}")

            # await client.read_resource(resource_uri)

        # Example: Get a prompt if available
        if prompts['count'] > 0:
            first_prompt = prompts['prompts'][0]
            prompt_name = first_prompt.get('name')
            print(f"\nGetting prompt: {prompt_name}")

            # await client.get_prompt(prompt_name, {})

        # Disconnect
        await client.disconnect()

    except Exception as e:
        print(f"\n✗ Error: {str(e)}")


async def example_error_handling():
    """Example: Error handling"""
    print("\n" + "="*60)
    print("Example 5: Error Handling")
    print("="*60)

    client = FastMCPServiceClient()

    # Test 1: Invalid client ID
    print("\nTest 1: List tools without connecting")
    try:
        await client.list_tools()
    except ValueError as e:
        print(f"✓ Caught expected error: {str(e)}")

    # Test 2: Invalid MCP URL
    print("\nTest 2: Connect to invalid MCP URL")
    try:
        await client.connect(
            transport="http",
            mcp_url="http://invalid-url-that-does-not-exist:9999/mcp",
            auth_type="none"
        )
    except Exception as e:
        print(f"✓ Caught connection error: {type(e).__name__}")

    # Test 3: Call non-existent tool
    print("\nTest 3: Call non-existent tool")
    try:
        # First connect to a valid server
        await client.connect(
            transport="http",
            mcp_url="http://localhost:3000/mcp",
            auth_type="none"
        )

        # Try to call a tool that doesn't exist
        await client.call_tool("non_existent_tool", {})

    except Exception as e:
        print(f"✓ Caught tool error: {type(e).__name__}")
    finally:
        if client.client_id:
            await client.disconnect()


async def main():
    """Run all examples"""
    print("\n" + "="*60)
    print("FastMCP Service - Usage Examples")
    print("="*60)

    # Note: These examples assume the FastMCP service is running
    # Start it with: python lib/mcp/fastmcp_service.py

    print("\nMake sure the FastMCP service is running:")
    print("  python lib/mcp/fastmcp_service.py")
    print("\nAnd an MCP server is available at http://localhost:3000/mcp")
    print("")

    # Run examples
    # Uncomment the examples you want to run:

    # await example_http_connection()
    # await example_bearer_auth()
    # await example_stdio_connection()
    # await example_complete_workflow()
    # await example_error_handling()

    # Simple health check that should always work
    client = FastMCPServiceClient()
    try:
        await client.health_check()
        print("\n✓ FastMCP service is running and healthy!")
    except Exception as e:
        print(f"\n✗ Cannot reach FastMCP service: {str(e)}")
        print("Make sure the service is running with:")
        print("  python lib/mcp/fastmcp_service.py")


if __name__ == "__main__":
    asyncio.run(main())
