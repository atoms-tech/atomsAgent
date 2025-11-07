"""
Test script for FastMCP server

Tests all 4 tools:
- search_requirements
- create_requirement
- analyze_document
- search_codebase
"""

import asyncio
import os
from fastmcp import Client

# Set environment variables for testing
os.environ["SUPABASE_URL"] = os.getenv("SUPABASE_URL", "https://ydogoylwenufckscqijp.supabase.co")
os.environ["SUPABASE_SERVICE_KEY"] = os.getenv("SUPABASE_SERVICE_KEY", "")


async def test_mcp_server():
    """Test the FastMCP server"""
    print("\n🧪 Testing FastMCP Server\n")
    print("=" * 60)
    
    # Import the MCP server
    from atomsAgent.mcp.server import mcp
    
    # Connect via in-memory transport
    async with Client(mcp) as client:
        print("\n✅ Connected to FastMCP server\n")
        
        # Test 1: List tools
        print("1️⃣  Listing available tools...")
        tools = await client.list_tools()
        print(f"   ✅ Found {len(tools)} tools:")
        for tool in tools:
            print(f"      - {tool.name}: {tool.description}")
        
        # Test 2: Search requirements (will fail without Supabase, but tests the tool)
        print("\n2️⃣  Testing search_requirements...")
        try:
            result = await client.call_tool("search_requirements", {
                "query": "test",
                "limit": 5
            })
            content = result.content[0]
            if hasattr(content, 'text'):
                print(f"   ✅ Tool executed: {content.text[:100]}...")
            else:
                print(f"   ✅ Tool executed: {str(content)[:100]}...")
        except Exception as e:
            print(f"   ⚠️  Tool executed with error: {str(e)[:100]}")
        
        # Test 3: Create requirement (will fail without Supabase, but tests the tool)
        print("\n3️⃣  Testing create_requirement...")
        try:
            result = await client.call_tool("create_requirement", {
                "project_id": "test-project",
                "title": "Test Requirement",
                "description": "This is a test requirement",
                "priority": "high"
            })
            content = result.content[0]
            if hasattr(content, 'text'):
                print(f"   ✅ Tool executed: {content.text[:100]}...")
            else:
                print(f"   ✅ Tool executed: {str(content)[:100]}...")
        except Exception as e:
            print(f"   ⚠️  Tool executed with error: {str(e)[:100]}")
        
        # Test 4: Analyze document (will fail without Supabase, but tests the tool)
        print("\n4️⃣  Testing analyze_document...")
        try:
            result = await client.call_tool("analyze_document", {
                "document_id": "test-doc",
                "analysis_type": "summary"
            })
            content = result.content[0]
            if hasattr(content, 'text'):
                print(f"   ✅ Tool executed: {content.text[:100]}...")
            else:
                print(f"   ✅ Tool executed: {str(content)[:100]}...")
        except Exception as e:
            print(f"   ⚠️  Tool executed with error: {str(e)[:100]}")
        
        # Test 5: Search codebase (should work if ripgrep is installed)
        print("\n5️⃣  Testing search_codebase...")
        try:
            result = await client.call_tool("search_codebase", {
                "query": "FastMCP",
                "file_pattern": "*.py",
                "max_results": 5
            })
            content = result.content[0]
            if hasattr(content, 'text'):
                print(f"   ✅ Tool executed: {content.text[:200]}...")
            else:
                print(f"   ✅ Tool executed: {str(content)[:200]}...")
        except Exception as e:
            print(f"   ⚠️  Tool executed with error: {str(e)[:100]}")
    
    print("\n" + "=" * 60)
    print("\n✅ FastMCP Server Test Complete\!\n")


if __name__ == "__main__":
    asyncio.run(test_mcp_server())
