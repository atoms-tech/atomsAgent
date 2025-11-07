"""
Integration Tests

End-to-end testing of the complete MCP + Claude + Artifacts + OAuth flow.
"""

import asyncio
import os
import sys
import uuid

# Add src to path
sys.path.insert(0, '/Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi/atomsAgent/src')

# Import directly to avoid circular imports
from fastmcp import FastMCP

# Create base MCP server inline
base_mcp = FastMCP("atoms-tools-test")

from atomsAgent.mcp.composition import compose_user_servers
from atomsAgent.mcp.claude_integration import get_mcp_servers_dict
from atomsAgent.services.artifacts import ArtifactDetector, ArtifactStorage, Artifact
from atomsAgent.mcp.oauth_handler import OAuthHandler, OAuthConfig
from atomsAgent.services.tool_approval import (
    ToolApprovalManager,
    ApprovalPolicy,
    ToolRiskLevel,
    ToolRiskAnalyzer
)


async def test_mcp_composition():
    """Test MCP composition"""
    print("=" * 60)
    print("Test 1: MCP Composition")
    print("=" * 60)
    
    user_id = "test-user-integration"
    
    try:
        # Compose MCP servers
        composed_mcp = await compose_user_servers(base_mcp, user_id)
        print(f"✅ Composed MCP for user {user_id}")
        
        # List tools
        from fastmcp import Client
        async with Client(composed_mcp) as client:
            tools = await client.list_tools()
            print(f"✅ Found {len(tools)} tools")
            
            # Test calling a tool
            result = await client.call_tool("search_requirements", {
                "query": "test",
                "limit": 5
            })
            print(f"✅ Called search_requirements tool")
            print(f"   Result: {result[:100] if isinstance(result, str) else str(result)[:100]}...")
        
        return True
    except Exception as e:
        print(f"❌ Error: {e}")
        import traceback
        traceback.print_exc()
        return False


async def test_artifact_detection():
    """Test artifact detection and storage"""
    print("\n" + "=" * 60)
    print("Test 2: Artifact Detection and Storage")
    print("=" * 60)
    
    try:
        # Test response with code block
        response_text = """
Here's a Python function to calculate fibonacci:

```python
def fibonacci(n):
    if n <= 1:
        return n
    return fibonacci(n-1) + fibonacci(n-2)
```

And here's a JavaScript version:

```javascript
function fibonacci(n) {
    if (n <= 1) return n;
    return fibonacci(n-1) + fibonacci(n-2);
}
```
"""
        
        # Detect artifacts
        artifacts = ArtifactDetector.detect_artifacts(response_text)
        print(f"✅ Detected {len(artifacts)} artifacts")
        
        for i, artifact in enumerate(artifacts, 1):
            print(f"   {i}. {artifact.type} ({artifact.language}): {artifact.title}")
        
        # Store artifacts
        storage = ArtifactStorage()
        session_id = "test-session-123"
        
        for artifact in artifacts:
            artifact_id = await storage.store_artifact(
                artifact,
                session_id=session_id,
                message_id="msg-123"
            )
            print(f"✅ Stored artifact {artifact_id}")
        
        # Retrieve artifacts
        session_artifacts = await storage.get_session_artifacts(session_id)
        print(f"✅ Retrieved {len(session_artifacts)} artifacts for session")
        
        return True
    except Exception as e:
        print(f"❌ Error: {e}")
        import traceback
        traceback.print_exc()
        return False


async def test_oauth_flow():
    """Test OAuth flow (without actual HTTP calls)"""
    print("\n" + "=" * 60)
    print("Test 3: OAuth Flow")
    print("=" * 60)
    
    try:
        handler = OAuthHandler()
        
        # Create OAuth config
        config = OAuthConfig(
            server_id="test-server",
            authorization_url="https://example.com/oauth/authorize",
            token_url="https://example.com/oauth/token",
            client_id="test-client-id",
            client_secret="test-client-secret",
            scopes=["read", "write"],
            use_pkce=True
        )
        
        # Generate authorization URL
        auth_url, state, verifier = handler.generate_authorization_url(config)
        print(f"✅ Generated authorization URL")
        print(f"   State: {state[:20]}...")
        print(f"   Verifier: {verifier[:20] if verifier else 'None'}...")
        
        # Verify PKCE
        if verifier:
            from atomsAgent.mcp.oauth_handler import PKCEGenerator
            challenge = PKCEGenerator.generate_code_challenge(verifier)
            print(f"✅ Generated PKCE challenge: {challenge[:20]}...")
        
        return True
    except Exception as e:
        print(f"❌ Error: {e}")
        import traceback
        traceback.print_exc()
        return False


async def test_tool_approval():
    """Test tool approval system"""
    print("\n" + "=" * 60)
    print("Test 4: Tool Approval System")
    print("=" * 60)
    
    try:
        manager = ToolApprovalManager()
        
        # Set up policy
        policy = ApprovalPolicy(
            auto_approve_risk_levels=[ToolRiskLevel.LOW],
            require_approval_tools=["delete_file"],
            deny_tools=["bash", "shell"]
        )
        manager.set_user_policy("test-user", policy)
        
        # Test 1: Auto-approve low-risk tool
        request1 = await manager.request_approval(
            request_id=str(uuid.uuid4()),
            tool_name="search_requirements",
            tool_input={"query": "test"},
            session_id="session-123",
            user_id="test-user"
        )
        print(f"✅ Low-risk tool: {request1.status} (expected: auto_approved)")
        assert request1.status == "auto_approved"
        
        # Test 2: Require approval for specific tool
        request2 = await manager.request_approval(
            request_id=str(uuid.uuid4()),
            tool_name="delete_file",
            tool_input={"path": "/tmp/test.txt"},
            session_id="session-123",
            user_id="test-user"
        )
        print(f"✅ Require approval tool: {request2.status} (expected: pending)")
        assert request2.status == "pending"
        
        # Test 3: Deny blocked tool
        request3 = await manager.request_approval(
            request_id=str(uuid.uuid4()),
            tool_name="bash",
            tool_input={"command": "ls"},
            session_id="session-123",
            user_id="test-user"
        )
        print(f"✅ Denied tool: {request3.status} (expected: denied)")
        assert request3.status == "denied"
        
        # Test 4: Approve pending request
        approved = await manager.approve_request(
            request2.id,
            approved_by="admin",
            reason="Approved for testing"
        )
        print(f"✅ Approved request: {approved.status} (expected: approved)")
        assert approved.status == "approved"
        
        # Test 5: Get pending requests
        pending = await manager.get_pending_requests("test-user")
        print(f"✅ Pending requests: {len(pending)}")
        
        return True
    except Exception as e:
        print(f"❌ Error: {e}")
        import traceback
        traceback.print_exc()
        return False


async def test_risk_analyzer():
    """Test tool risk analyzer"""
    print("\n" + "=" * 60)
    print("Test 5: Tool Risk Analyzer")
    print("=" * 60)
    
    try:
        # Test known tools
        tests = [
            ("search_requirements", {}, ToolRiskLevel.LOW),
            ("create_requirement", {}, ToolRiskLevel.MEDIUM),
            ("delete_file", {}, ToolRiskLevel.HIGH),
            ("bash", {"command": "ls"}, ToolRiskLevel.CRITICAL),
        ]
        
        for tool_name, tool_input, expected_risk in tests:
            risk = ToolRiskAnalyzer.analyze_risk(tool_name, tool_input)
            status = "✅" if risk == expected_risk else "❌"
            print(f"{status} {tool_name}: {risk.value} (expected: {expected_risk.value})")
            assert risk == expected_risk
        
        return True
    except Exception as e:
        print(f"❌ Error: {e}")
        import traceback
        traceback.print_exc()
        return False


async def test_end_to_end():
    """Test complete end-to-end flow"""
    print("\n" + "=" * 60)
    print("Test 6: End-to-End Flow")
    print("=" * 60)
    
    try:
        user_id = "test-user-e2e"
        session_id = "session-e2e"
        
        # 1. Get composed MCP servers
        servers_dict = await get_mcp_servers_dict(user_id)
        print(f"✅ Step 1: Got MCP servers dict with {len(servers_dict)} servers")
        
        # 2. Simulate Claude response with artifacts
        response_text = """
I'll help you create a function:

```python
def greet(name: str) -> str:
    return f"Hello, {name}\!"
```
"""
        
        # 3. Detect artifacts
        artifacts = ArtifactDetector.detect_artifacts(response_text)
        print(f"✅ Step 2: Detected {len(artifacts)} artifacts")
        
        # 4. Store artifacts
        storage = ArtifactStorage()
        for artifact in artifacts:
            await storage.store_artifact(artifact, session_id)
        print(f"✅ Step 3: Stored artifacts")
        
        # 5. Request tool approval
        manager = ToolApprovalManager()
        request = await manager.request_approval(
            request_id=str(uuid.uuid4()),
            tool_name="search_requirements",
            tool_input={"query": "test"},
            session_id=session_id,
            user_id=user_id
        )
        print(f"✅ Step 4: Tool approval: {request.status}")
        
        # 6. Retrieve session artifacts
        session_artifacts = await storage.get_session_artifacts(session_id)
        print(f"✅ Step 5: Retrieved {len(session_artifacts)} artifacts")
        
        print(f"\n✅ End-to-end flow completed successfully\!")
        return True
    except Exception as e:
        print(f"❌ Error: {e}")
        import traceback
        traceback.print_exc()
        return False


async def main():
    """Run all integration tests"""
    print("\n" + "=" * 60)
    print("INTEGRATION TESTS")
    print("=" * 60)
    
    # Check environment
    if not os.getenv("SUPABASE_URL") or not os.getenv("SUPABASE_SERVICE_KEY"):
        print("⚠️  Warning: Missing Supabase credentials")
        print("   Tests will use in-memory storage")
    
    # Run tests
    results = {}
    results["mcp_composition"] = await test_mcp_composition()
    results["artifact_detection"] = await test_artifact_detection()
    results["oauth_flow"] = await test_oauth_flow()
    results["tool_approval"] = await test_tool_approval()
    results["risk_analyzer"] = await test_risk_analyzer()
    results["end_to_end"] = await test_end_to_end()
    
    # Summary
    print("\n" + "=" * 60)
    print("TEST RESULTS")
    print("=" * 60)
    
    for test_name, passed in results.items():
        status = "✅ PASS" if passed else "❌ FAIL"
        print(f"  {test_name}: {status}")
    
    print("=" * 60)
    
    all_passed = all(results.values())
    if all_passed:
        print("\n🎉 All integration tests passed\!")
    else:
        print("\n❌ Some integration tests failed")
    
    return all_passed


if __name__ == "__main__":
    asyncio.run(main())
