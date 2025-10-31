#!/usr/bin/env python3
"""Test MCP functionality directly without CLI import issues."""

import asyncio
import sys
import os
from pathlib import Path

# Add atomsAgent to path
sys.path.insert(0, str(Path(__file__).parent / "atomsAgent" / "src"))

from atomsAgent.db.supabase import SupabaseClient
from atomsAgent.services.mcp_registry import MCPRegistryService

async def test_mcp_list():
    # Database configuration
    supabase_url = "https://ydogoylwenufckscqijp.supabase.co"
    service_role_key = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6Inlkb2dveWx3ZW51ZmNrc2NxaWpwIiwicm9sZSI6InNlcnZpY2Vfcm9sZSIsImlhdCI6MTczNjczNTE2NiwiZXhwIjoyMDUyMzExMTY2fQ.fSWnBNuIE3QXU93naKCmbUiWkGg5LVnOQg5uSyLYaNo"
    
    # Create client
    client = SupabaseClient(
        url=supabase_url,
        service_role_key=service_role_key
    )
    
    # Create service
    repository = type('DummyRepo', (), {
        'list_configs': lambda self, **kwargs: client.select(
            'mcp_configurations', 
            columns='id,org_id,user_id,name,type,endpoint,auth_type,auth_token,auth_header,config,scope,enabled,description,created_at,updated_at,created_by,updated_by',
            limit=10
        )
    })()
    
    service = MCPRegistryService(repository)
    
    # Test list
    from uuid import UUID
    org_id = UUID("6a1ae886-4eb0-4bac-b729-5dde65efb78c")
    
    try:
        response = await service.list(
            organization_id=org_id,
            user_id=None,
            include_platform=True
        )
        print(f"✅ Success! Found {len(response.items)} MCP configurations:")
        for item in response.items:
            print(f"  • {item.name} (type: {item.type}, enabled: {item.enabled})")
    except Exception as e:
        print(f"❌ Error: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    asyncio.run(test_mcp_list())
