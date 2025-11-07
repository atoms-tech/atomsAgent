"""
Test MCP Composition

Tests the composition of multiple MCP servers.
"""

import asyncio
import os
from atomsAgent.mcp.server import mcp as base_mcp
from atomsAgent.mcp.composition import compose_user_servers, get_user_mcp_servers


async def test_get_user_servers():
    """Test fetching user's MCP servers"""
    print("Testing get_user_mcp_servers...")
    
    # Test with a dummy user ID
    user_id = "test-user-123"
    
    try:
        servers = await get_user_mcp_servers(user_id)
        print(f"✅ Found {len(servers)} servers for user {user_id}")
        
        for server in servers:
            server_data = server.get("server", {})
            print(f"  - {server_data.get('name')} ({server_data.get('scope')})")
        
        return True
    except Exception as e:
        print(f"❌ Error: {e}")
        return False


async def test_compose_servers():
    """Test composing MCP servers"""
    print("\nTesting compose_user_servers...")
    
    user_id = "test-user-123"
    
    try:
        # Compose servers
        composed_mcp = await compose_user_servers(base_mcp, user_id)
        
        # List all tools
        from fastmcp import Client
        async with Client(composed_mcp) as client:
            tools = await client.list_tools()
            print(f"✅ Composed MCP has {len(tools.tools)} tools:")
            
            for tool in tools.tools:
                print(f"  - {tool.name}: {tool.description}")
        
        return True
    except Exception as e:
        print(f"❌ Error: {e}")
        import traceback
        traceback.print_exc()
        return False


async def main():
    """Run all tests"""
    print("=" * 60)
    print("MCP Composition Tests")
    print("=" * 60)
    
    # Check environment variables
    if not os.getenv("SUPABASE_URL") or not os.getenv("SUPABASE_SERVICE_KEY"):
        print("❌ Missing Supabase credentials")
        print("Set SUPABASE_URL and SUPABASE_SERVICE_KEY environment variables")
        return
    
    # Run tests
    test1 = await test_get_user_servers()
    test2 = await test_compose_servers()
    
    print("\n" + "=" * 60)
    print("Test Results:")
    print(f"  get_user_mcp_servers: {'✅ PASS' if test1 else '❌ FAIL'}")
    print(f"  compose_user_servers: {'✅ PASS' if test2 else '❌ FAIL'}")
    print("=" * 60)


if __name__ == "__main__":
    asyncio.run(main())
