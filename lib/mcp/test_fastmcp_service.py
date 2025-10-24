#!/usr/bin/env python3
"""
Test suite for FastMCP Service
Run with: pytest test_fastmcp_service.py -v
"""

import asyncio
import pytest
from fastapi.testclient import TestClient
from unittest.mock import Mock, AsyncMock, patch

# Import the service
from fastmcp_service import app, MCPClientManager, ConnectRequest, TransportType, AuthType


@pytest.fixture
def client():
    """Create a test client"""
    return TestClient(app)


@pytest.fixture
def mock_fastmcp_client():
    """Create a mock FastMCP client"""
    client = AsyncMock()
    client.connect = AsyncMock()
    client.disconnect = AsyncMock()
    client.list_tools = AsyncMock(return_value=[])
    client.list_resources = AsyncMock(return_value=[])
    client.list_prompts = AsyncMock(return_value=[])
    client.call_tool = AsyncMock(return_value={"result": "success"})
    client.read_resource = AsyncMock(return_value={"content": "test"})
    client.get_prompt = AsyncMock(return_value={"prompt": "test"})
    return client


class TestHealthEndpoint:
    """Test health check endpoint"""

    def test_health_check(self, client):
        """Test basic health check"""
        response = client.get("/health")
        assert response.status_code == 200

        data = response.json()
        assert data["status"] == "healthy"
        assert "timestamp" in data
        assert "active_clients" in data
        assert data["version"] == "2.0.0"


class TestConnectEndpoint:
    """Test MCP connection endpoint"""

    @patch('fastmcp_service.FastMCPClient')
    def test_connect_http_transport(self, mock_client_class, client, mock_fastmcp_client):
        """Test connecting with HTTP transport"""
        mock_client_class.return_value = mock_fastmcp_client

        request_data = {
            "transport": "http",
            "mcp_url": "http://example.com/mcp",
            "auth_type": "none",
            "name": "test-client"
        }

        response = client.post("/mcp/connect", json=request_data)
        assert response.status_code == 201

        data = response.json()
        assert data["status"] == "connected"
        assert "client_id" in data
        assert isinstance(data["tools"], list)
        assert isinstance(data["resources"], list)
        assert isinstance(data["prompts"], list)

    @patch('fastmcp_service.FastMCPClient')
    def test_connect_with_bearer_auth(self, mock_client_class, client, mock_fastmcp_client):
        """Test connecting with bearer authentication"""
        mock_client_class.return_value = mock_fastmcp_client

        request_data = {
            "transport": "http",
            "mcp_url": "http://example.com/mcp",
            "auth_type": "bearer",
            "bearer_token": "test-token-123"
        }

        response = client.post("/mcp/connect", json=request_data)
        assert response.status_code == 201

        data = response.json()
        assert data["status"] == "connected"

    def test_connect_missing_url(self, client):
        """Test connection fails without URL for HTTP transport"""
        request_data = {
            "transport": "http",
            "auth_type": "none"
        }

        response = client.post("/mcp/connect", json=request_data)
        assert response.status_code == 422  # Validation error

    def test_connect_stdio_transport(self, client):
        """Test connection fails without command for stdio transport"""
        request_data = {
            "transport": "stdio",
            "auth_type": "none"
        }

        response = client.post("/mcp/connect", json=request_data)
        assert response.status_code == 422  # Validation error


class TestToolCallEndpoint:
    """Test tool calling endpoint"""

    @patch('fastmcp_service.FastMCPClient')
    def test_call_tool_success(self, mock_client_class, client, mock_fastmcp_client):
        """Test successful tool call"""
        # First connect
        mock_client_class.return_value = mock_fastmcp_client

        connect_data = {
            "transport": "http",
            "mcp_url": "http://example.com/mcp",
            "auth_type": "none"
        }

        connect_response = client.post("/mcp/connect", json=connect_data)
        client_id = connect_response.json()["client_id"]

        # Then call tool
        tool_request = {
            "client_id": client_id,
            "tool_name": "test_tool",
            "arguments": {"arg1": "value1"}
        }

        response = client.post("/mcp/call_tool", json=tool_request)
        assert response.status_code == 200

        data = response.json()
        assert data["status"] == "success"
        assert data["result"] is not None
        assert "execution_time" in data

    def test_call_tool_invalid_client(self, client):
        """Test tool call with invalid client ID"""
        tool_request = {
            "client_id": "invalid-client-id",
            "tool_name": "test_tool",
            "arguments": {}
        }

        response = client.post("/mcp/call_tool", json=tool_request)
        assert response.status_code == 404


class TestListToolsEndpoint:
    """Test list tools endpoint"""

    @patch('fastmcp_service.FastMCPClient')
    def test_list_tools(self, mock_client_class, client, mock_fastmcp_client):
        """Test listing tools"""
        # Setup mock tools
        mock_tool = Mock()
        mock_tool.dict.return_value = {
            "name": "test_tool",
            "description": "A test tool"
        }
        mock_fastmcp_client.list_tools.return_value = [mock_tool]
        mock_client_class.return_value = mock_fastmcp_client

        # Connect first
        connect_data = {
            "transport": "http",
            "mcp_url": "http://example.com/mcp",
            "auth_type": "none"
        }

        connect_response = client.post("/mcp/connect", json=connect_data)
        client_id = connect_response.json()["client_id"]

        # List tools
        response = client.get(f"/mcp/list_tools?client_id={client_id}")
        assert response.status_code == 200

        data = response.json()
        assert data["status"] == "success"
        assert "tools" in data
        assert data["count"] >= 0

    def test_list_tools_invalid_client(self, client):
        """Test listing tools with invalid client ID"""
        response = client.get("/mcp/list_tools?client_id=invalid-id")
        assert response.status_code == 404


class TestDisconnectEndpoint:
    """Test disconnect endpoint"""

    @patch('fastmcp_service.FastMCPClient')
    def test_disconnect_success(self, mock_client_class, client, mock_fastmcp_client):
        """Test successful disconnect"""
        mock_client_class.return_value = mock_fastmcp_client

        # Connect first
        connect_data = {
            "transport": "http",
            "mcp_url": "http://example.com/mcp",
            "auth_type": "none"
        }

        connect_response = client.post("/mcp/connect", json=connect_data)
        client_id = connect_response.json()["client_id"]

        # Disconnect
        disconnect_data = {"client_id": client_id}
        response = client.post("/mcp/disconnect", json=disconnect_data)
        assert response.status_code == 200

        data = response.json()
        assert data["status"] == "disconnected"
        assert data["client_id"] == client_id

    def test_disconnect_invalid_client(self, client):
        """Test disconnect with invalid client ID"""
        disconnect_data = {"client_id": "invalid-id"}
        response = client.post("/mcp/disconnect", json=disconnect_data)
        assert response.status_code == 404


class TestClientManagement:
    """Test client management endpoints"""

    @patch('fastmcp_service.FastMCPClient')
    def test_list_clients(self, mock_client_class, client, mock_fastmcp_client):
        """Test listing all clients"""
        mock_client_class.return_value = mock_fastmcp_client

        # Connect a client
        connect_data = {
            "transport": "http",
            "mcp_url": "http://example.com/mcp",
            "auth_type": "none"
        }

        client.post("/mcp/connect", json=connect_data)

        # List clients
        response = client.get("/mcp/clients")
        assert response.status_code == 200

        data = response.json()
        assert data["status"] == "success"
        assert "clients" in data
        assert data["count"] > 0

    @patch('fastmcp_service.FastMCPClient')
    def test_get_client_info(self, mock_client_class, client, mock_fastmcp_client):
        """Test getting client info"""
        mock_client_class.return_value = mock_fastmcp_client

        # Connect a client
        connect_data = {
            "transport": "http",
            "mcp_url": "http://example.com/mcp",
            "auth_type": "none"
        }

        connect_response = client.post("/mcp/connect", json=connect_data)
        client_id = connect_response.json()["client_id"]

        # Get client info
        response = client.get(f"/mcp/client/{client_id}")
        assert response.status_code == 200

        data = response.json()
        assert data["status"] == "success"
        assert data["client_id"] == client_id
        assert "metadata" in data


class TestMCPClientManager:
    """Test MCPClientManager class"""

    @pytest.mark.asyncio
    async def test_create_client(self):
        """Test creating a client"""
        manager = MCPClientManager()

        # Mock the FastMCPClient
        with patch('fastmcp_service.FastMCPClient') as mock_client_class:
            mock_client = AsyncMock()
            mock_client.connect = AsyncMock()
            mock_client_class.return_value = mock_client

            config = ConnectRequest(
                transport=TransportType.HTTP,
                mcp_url="http://example.com/mcp",
                auth_type=AuthType.NONE
            )

            client_id, client = await manager.create_client(config)

            assert client_id is not None
            assert client is not None
            assert client_id in manager.clients
            assert client_id in manager.client_metadata

    @pytest.mark.asyncio
    async def test_get_client(self):
        """Test getting a client"""
        manager = MCPClientManager()

        with patch('fastmcp_service.FastMCPClient') as mock_client_class:
            mock_client = AsyncMock()
            mock_client.connect = AsyncMock()
            mock_client_class.return_value = mock_client

            config = ConnectRequest(
                transport=TransportType.HTTP,
                mcp_url="http://example.com/mcp",
                auth_type=AuthType.NONE
            )

            client_id, _ = await manager.create_client(config)
            retrieved_client = manager.get_client(client_id)

            assert retrieved_client is not None

    @pytest.mark.asyncio
    async def test_disconnect_client(self):
        """Test disconnecting a client"""
        manager = MCPClientManager()

        with patch('fastmcp_service.FastMCPClient') as mock_client_class:
            mock_client = AsyncMock()
            mock_client.connect = AsyncMock()
            mock_client.disconnect = AsyncMock()
            mock_client_class.return_value = mock_client

            config = ConnectRequest(
                transport=TransportType.HTTP,
                mcp_url="http://example.com/mcp",
                auth_type=AuthType.NONE
            )

            client_id, _ = await manager.create_client(config)
            success = await manager.disconnect_client(client_id)

            assert success is True
            assert client_id not in manager.clients
            assert client_id not in manager.client_metadata

    @pytest.mark.asyncio
    async def test_disconnect_all(self):
        """Test disconnecting all clients"""
        manager = MCPClientManager()

        with patch('fastmcp_service.FastMCPClient') as mock_client_class:
            mock_client = AsyncMock()
            mock_client.connect = AsyncMock()
            mock_client.disconnect = AsyncMock()
            mock_client_class.return_value = mock_client

            # Create multiple clients
            config = ConnectRequest(
                transport=TransportType.HTTP,
                mcp_url="http://example.com/mcp",
                auth_type=AuthType.NONE
            )

            await manager.create_client(config)
            await manager.create_client(config)

            # Disconnect all
            await manager.disconnect_all()

            assert len(manager.clients) == 0
            assert len(manager.client_metadata) == 0


class TestValidation:
    """Test request validation"""

    def test_connect_request_validation(self):
        """Test ConnectRequest validation"""
        # Valid HTTP request
        request = ConnectRequest(
            transport=TransportType.HTTP,
            mcp_url="http://example.com/mcp",
            auth_type=AuthType.NONE
        )
        assert request.transport == TransportType.HTTP

        # Invalid - missing URL for HTTP
        with pytest.raises(ValueError):
            ConnectRequest(
                transport=TransportType.HTTP,
                auth_type=AuthType.NONE
            )

        # Invalid - missing command for STDIO
        with pytest.raises(ValueError):
            ConnectRequest(
                transport=TransportType.STDIO,
                auth_type=AuthType.NONE
            )

        # Invalid - missing bearer token
        with pytest.raises(ValueError):
            ConnectRequest(
                transport=TransportType.HTTP,
                mcp_url="http://example.com/mcp",
                auth_type=AuthType.BEARER
            )


# Run tests
if __name__ == "__main__":
    pytest.main([__file__, "-v"])
