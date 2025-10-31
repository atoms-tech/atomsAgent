#!/usr/bin/env python3
"""
Test script to verify system_prompts table access after RLS fix.
This script uses the same endpoint that was failing in the log.
"""

import os
import sys
import asyncio
import httpx
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

# Get Supabase credentials
SUPABASE_URL = os.getenv('SUPABASE_URL')
SUPABASE_SERVICE_KEY = os.getenv('SUPABASE_SERVICE_ROLE_KEY')

if not SUPABASE_URL or not SUPABASE_SERVICE_KEY:
    print("ERROR: Missing SUPABASE_URL or SUPABASE_SERVICE_ROLE_KEY in environment")
    sys.exit(1)

async def test_system_prompts_access():
    """Test access to system_prompts table using the same query that was failing."""
    
    url = f"{SUPABASE_URL}/rest/v1/system_prompts"
    headers = {
        "apikey": SUPABASE_SERVICE_KEY,
        "Authorization": f"Bearer {SUPABASE_SERVICE_KEY}",
        "Accept": "application/json",
        "Prefer": "return=representation"
    }
    
    # This is the exact same query that was failing with 403
    params = {
        "select": "id,content,priority,scope,organization_id,user_id,template,enabled",
        "enabled": "eq.true",
        "order": "priority.desc"
    }
    
    print(f"Testing access to: {url}")
    print(f"Headers: {headers}")
    print(f"Params: {params}")
    print("-" * 50)
    
    async with httpx.AsyncClient() as client:
        try:
            response = await client.get(url, headers=headers, params=params)
            
            print(f"Status Code: {response.status_code}")
            print(f"Response Headers: {dict(response.headers)}")
            
            if response.status_code == 200:
                data = response.json()
                print(f"✅ SUCCESS: Retrieved {len(data)} system prompts")
                if data:
                    print("\nFirst prompt preview:")
                    print(f"  ID: {data[0].get('id')}")
                    print(f"  Scope: {data[0].get('scope')}")
                    print(f"  Enabled: {data[0].get('enabled')}")
                    print(f"  Priority: {data[0].get('priority')}")
                    content = data[0].get('content', '')[:100]
                    print(f"  Content: {content}...")
            else:
                print(f"❌ ERROR: {response.status_code}")
                print(f"Response: {response.text}")
                
        except Exception as e:
            print(f"❌ EXCEPTION: {e}")

if __name__ == "__main__":
    asyncio.run(test_system_prompts_access())
