"""
Test Claude Integration with MCP Composition

Tests that Claude client properly composes MCP servers for users.
"""

import asyncio
import os
from atomsAgent.mcp.claude_integration import get_mcp_servers_dict, get_composed_mcp_for_user


async def test_get_composed_mcp():
    """Test getting composed MCP for a user"""
    print("=" * 60)
    print("Test 1: Get Composed MCP for User")
    print("=" * 60)
    
    user_id = "test-user-123"
    
    try:
        composed_mcp = await get_composed_mcp_for_user(user_id)
        print(f"✅ Got composed MCP for user {user_id}")
        print(f"   Type: {type(composed_mcp)}")
        
        # Try to list tools
        from fastmcp import Client
        async with Client(composed_mcp) as client:
            tools = await client.list_tools()
            print(f"✅ Composed MCP has {len(tools)} tools:")
            
            for tool in tools[:10]:  # Show first 10
                print(f"   - {tool.name}: {tool.description[:60]}...")
        
        return True
    except Exception as e:
        print(f"❌ Error: {e}")
        import traceback
        traceback.print_exc()
        return False


async def test_get_mcp_servers_dict():
    """Test getting MCP servers dict for Claude SDK"""
    print("\n" + "=" * 60)
    print("Test 2: Get MCP Servers Dict for Claude SDK")
    print("=" * 60)
    
    user_id = "test-user-123"
    
    try:
        servers_dict = await get_mcp_servers_dict(user_id)
        print(f"✅ Got MCP servers dict for user {user_id}")
        print(f"   Servers: {list(servers_dict.keys())}")
        
        # Check the composed server
        if "atoms-composed" in servers_dict:
            composed = servers_dict["atoms-composed"]
            print(f"✅ Found 'atoms-composed' server")
            print(f"   Type: {type(composed)}")
            
            # List tools
            from fastmcp import Client
            async with Client(composed) as client:
                tools = await client.list_tools()
                print(f"✅ atoms-composed has {len(tools)} tools")
        
        return True
    except Exception as e:
        print(f"❌ Error: {e}")
        import traceback
        traceback.print_exc()
        return False


async def test_with_custom_servers():
    """Test with additional custom servers"""
    print("\n" + "=" * 60)
    print("Test 3: Get MCP Servers Dict with Custom Servers")
    print("=" * 60)
    
    user_id = "test-user-123"
    
    # Mock custom server (just for testing structure)
    from atomsAgent.mcp.server import mcp as base_mcp
    custom_servers = {
        "custom-test": base_mcp
    }
    
    try:
        servers_dict = await get_mcp_servers_dict(
            user_id=user_id,
            custom_servers=custom_servers
        )
        print(f"✅ Got MCP servers dict with custom servers")
        print(f"   Servers: {list(servers_dict.keys())}")
        
        if "custom-test" in servers_dict:
            print(f"✅ Custom server 'custom-test' included")
        
        return True
    except Exception as e:
        print(f"❌ Error: {e}")
        import traceback
        traceback.print_exc()
        return False


async def main():
    """Run all tests"""
    print("\n" + "=" * 60)
    print("Claude Integration with MCP Composition Tests")
    print("=" * 60)
    
    # Check environment variables
    if not os.getenv("SUPABASE_URL") or not os.getenv("SUPABASE_SERVICE_KEY"):
        print("⚠️  Warning: Missing Supabase credentials")
        print("   Tests will use empty server list")
        print("   Set SUPABASE_URL and SUPABASE_SERVICE_KEY for full testing")
    
    # Run tests
    test1 = await test_get_composed_mcp()
    test2 = await test_get_mcp_servers_dict()
    test3 = await test_with_custom_servers()
    
    print("\n" + "=" * 60)
    print("Test Results:")
    print(f"  get_composed_mcp_for_user: {'✅ PASS' if test1 else '❌ FAIL'}")
    print(f"  get_mcp_servers_dict: {'✅ PASS' if test2 else '❌ FAIL'}")
    print(f"  with_custom_servers: {'✅ PASS' if test3 else '❌ FAIL'}")
    print("=" * 60)
    
    if test1 and test2 and test3:
        print("\n🎉 All tests passed\!")
        print("✅ Claude integration with MCP composition is working\!")
    else:
        print("\n❌ Some tests failed")


if __name__ == "__main__":
    asyncio.run(main())
