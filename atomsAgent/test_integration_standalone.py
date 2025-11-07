"""
Standalone Integration Tests

Tests that don't rely on atomsAgent package imports.
"""

import asyncio
import os
import sys
import uuid

# Direct imports
sys.path.insert(0, '/Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi/atomsAgent/src')

# Import modules directly
from atomsAgent.services.artifacts import ArtifactDetector, ArtifactStorage
from atomsAgent.mcp.oauth_handler import OAuthHandler, OAuthConfig, PKCEGenerator
from atomsAgent.services.tool_approval import (
    ToolApprovalManager,
    ApprovalPolicy,
    ToolRiskLevel,
    ToolRiskAnalyzer
)


async def test_artifact_detection():
    """Test artifact detection and storage"""
    print("=" * 60)
    print("Test 1: Artifact Detection and Storage")
    print("=" * 60)
    
    try:
        response_text = """
Here's a Python function:

```python
def fibonacci(n):
    if n <= 1:
        return n
    return fibonacci(n-1) + fibonacci(n-2)
```

And JavaScript:

```javascript
function fibonacci(n) {
    if (n <= 1) return n;
    return fibonacci(n-1) + fibonacci(n-2);
}
```
"""
        
        artifacts = ArtifactDetector.detect_artifacts(response_text)
        print(f"✅ Detected {len(artifacts)} artifacts")
        
        for i, artifact in enumerate(artifacts, 1):
            print(f"   {i}. {artifact.type} ({artifact.language}): {artifact.title}")
        
        storage = ArtifactStorage()
        session_id = "test-session-123"
        
        for artifact in artifacts:
            artifact_id = await storage.store_artifact(
                artifact,
                session_id=session_id,
                message_id="msg-123"
            )
            print(f"✅ Stored artifact {artifact_id[:20]}...")
        
        session_artifacts = await storage.get_session_artifacts(session_id)
        print(f"✅ Retrieved {len(session_artifacts)} artifacts for session")
        
        return True
    except Exception as e:
        print(f"❌ Error: {e}")
        import traceback
        traceback.print_exc()
        return False


async def test_oauth_pkce():
    """Test OAuth PKCE generation"""
    print("\n" + "=" * 60)
    print("Test 2: OAuth PKCE")
    print("=" * 60)
    
    try:
        # Generate code verifier
        verifier = PKCEGenerator.generate_code_verifier()
        print(f"✅ Generated code verifier: {verifier[:20]}...")
        
        # Generate code challenge
        challenge = PKCEGenerator.generate_code_challenge(verifier)
        print(f"✅ Generated code challenge: {challenge[:20]}...")
        
        # Verify they're different
        assert verifier != challenge
        print(f"✅ Verifier and challenge are different")
        
        return True
    except Exception as e:
        print(f"❌ Error: {e}")
        import traceback
        traceback.print_exc()
        return False


async def test_oauth_url_generation():
    """Test OAuth URL generation"""
    print("\n" + "=" * 60)
    print("Test 3: OAuth URL Generation")
    print("=" * 60)
    
    try:
        handler = OAuthHandler()
        
        config = OAuthConfig(
            server_id="test-server",
            authorization_url="https://example.com/oauth/authorize",
            token_url="https://example.com/oauth/token",
            client_id="test-client-id",
            client_secret="test-client-secret",
            scopes=["read", "write"],
            use_pkce=True
        )
        
        auth_url, state, verifier = handler.generate_authorization_url(config)
        print(f"✅ Generated authorization URL")
        print(f"   URL: {auth_url[:60]}...")
        print(f"   State: {state[:20]}...")
        print(f"   Verifier: {verifier[:20] if verifier else 'None'}...")
        
        # Verify URL contains required params
        assert "client_id=test-client-id" in auth_url
        assert "state=" in auth_url
        assert "code_challenge=" in auth_url
        print(f"✅ URL contains required parameters")
        
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
        
        policy = ApprovalPolicy(
            auto_approve_risk_levels=[ToolRiskLevel.LOW],
            require_approval_tools=["delete_file"],
            deny_tools=["bash", "shell"]
        )
        manager.set_user_policy("test-user", policy)
        
        # Test 1: Auto-approve low-risk
        request1 = await manager.request_approval(
            request_id=str(uuid.uuid4()),
            tool_name="search_requirements",
            tool_input={"query": "test"},
            session_id="session-123",
            user_id="test-user"
        )
        print(f"✅ Low-risk tool: {request1.status} (expected: auto_approved)")
        assert request1.status == "auto_approved"
        
        # Test 2: Require approval
        request2 = await manager.request_approval(
            request_id=str(uuid.uuid4()),
            tool_name="delete_file",
            tool_input={"path": "/tmp/test.txt"},
            session_id="session-123",
            user_id="test-user"
        )
        print(f"✅ Require approval: {request2.status} (expected: pending)")
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
        
        # Test 4: Approve pending
        approved = await manager.approve_request(
            request2.id,
            approved_by="admin",
            reason="Approved for testing"
        )
        print(f"✅ Approved request: {approved.status} (expected: approved)")
        assert approved.status == "approved"
        
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


async def main():
    """Run all tests"""
    print("\n" + "=" * 60)
    print("STANDALONE INTEGRATION TESTS")
    print("=" * 60)
    
    if not os.getenv("SUPABASE_URL") or not os.getenv("SUPABASE_SERVICE_KEY"):
        print("⚠️  Warning: Missing Supabase credentials")
        print("   Tests will use in-memory storage")
    
    results = {}
    results["artifact_detection"] = await test_artifact_detection()
    results["oauth_pkce"] = await test_oauth_pkce()
    results["oauth_url_generation"] = await test_oauth_url_generation()
    results["tool_approval"] = await test_tool_approval()
    results["risk_analyzer"] = await test_risk_analyzer()
    
    print("\n" + "=" * 60)
    print("TEST RESULTS")
    print("=" * 60)
    
    for test_name, passed in results.items():
        status = "✅ PASS" if passed else "❌ FAIL"
        print(f"  {test_name}: {status}")
    
    print("=" * 60)
    
    all_passed = all(results.values())
    if all_passed:
        print("\n🎉 All integration tests passed!")
    else:
        print("\n❌ Some integration tests failed")
    
    return all_passed


if __name__ == "__main__":
    asyncio.run(main())
