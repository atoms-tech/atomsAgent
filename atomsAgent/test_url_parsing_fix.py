#!/usr/bin/env python3
"""
Test script to verify the URL parsing fix for GitHub-based MCP servers.

This script tests the fix for the issue where MCP server URLs stored as JSON
strings (e.g., {"url":"https://github.com/mcpcap/mcpcap","source":"github"})
were not being properly parsed, causing URL construction errors.
"""

import sys
import asyncio
sys.path.insert(0, 'src')

from atomsAgent.mcp.database import convert_db_server_to_mcp_config
from atomsAgent.mcp.composition import create_mcp_client
from unittest.mock import patch


def test_database_url_parsing():
    """Test URL parsing in database.py module"""
    print("Testing database URL parsing...")
    
    # Test case 1: Normal URL (should work as before)
    server_config_normal = {
        'id': 'test-normal',
        'name': 'normal-server',
        'transport_type': 'http',
        'server_url': 'https://example.com/mcp'
    }
    
    config = convert_db_server_to_mcp_config(server_config_normal)
    assert config.get('url') == 'https://example.com/mcp'
    print("✓ Normal URL parsing works")
    
    # Test case 2: JSON string URL (the problematic case)
    server_config_json = {
        'id': 'test-json',
        'name': 'github-server',
        'transport_type': 'http',
        'server_url': '{"url":"https://github.com/mcpcap/mcpcap","source":"github"}'
    }
    
    config = convert_db_server_to_mcp_config(server_config_json)
    assert config.get('url') == 'https://github.com/mcpcap/mcpcap'
    print("✓ JSON string URL parsing works")
    
    # Test case 3: Invalid JSON (should not crash)
    server_config_invalid = {
        'id': 'test-invalid',
        'name': 'invalid-server',
        'transport_type': 'http',
        'server_url': '{"url":"https://github.com/mcpcap/mcpcap","source":"github"'  # Missing closing brace
    }
    
    config = convert_db_server_to_mcp_config(server_config_invalid)
    # Should not have extracted a URL from invalid JSON
    assert config.get('url') != 'https://github.com/mcpcap/mcpcap'
    print("✓ Invalid JSON handled gracefully")
    

async def test_composition_url_parsing():
    """Test URL parsing in composition.py module"""
    print("\nTesting composition URL parsing...")
    
    # Test case with JSON URL
    server_config = {
        'server': {
            'name': 'github-server',
            'transport_type': 'http',
            'server_url': '{"url":"https://github.com/mcpcap/mcpcap","source":"github"}',
            'auth_type': 'none'
        }
    }
    
    # Mock the Client to avoid actual network calls
    with patch('atomsAgent.mcp.composition.Client') as mock_client:
        await create_mcp_client(server_config)
        
        # Verify the URL was extracted correctly
        mock_client.assert_called_once()
        call_args = mock_client.call_args
        url = call_args.kwargs.get('url') or call_args[1].get('url')
        assert url == 'https://github.com/mcpcap/mcpcap'
        print("✓ JSON URL extracted correctly in composition module")


async def main():
    """Run all tests"""
    print("=" * 60)
    print("Testing URL Parsing Fix for GitHub MCP Servers")
    print("=" * 60)
    
    try:
        test_database_url_parsing()
        await test_composition_url_parsing()
        
        print("\n" + "=" * 60)
        print("✅ All tests passed! The URL parsing fix is working correctly.")
        print("This should resolve the error:")
        print('  "Failed to parse URL from {"url":"https://github.com/mcpcap/mcpcap","source":"github"}/tools/list"')
        print("=" * 60)
        
    except Exception as e:
        print(f"\n❌ Test failed with error: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())
