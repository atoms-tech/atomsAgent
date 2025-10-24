#!/usr/bin/env python3
"""
FastMCP Service for AgentAPI
Production-ready FastAPI service for MCP client management using FastMCP 2.0
"""

import asyncio
import logging
import sys
import traceback
import uuid
from contextlib import asynccontextmanager
from datetime import datetime
from typing import Any, Dict, List, Optional, Union
from enum import Enum

from fastapi import FastAPI, HTTPException, Request, status
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
from pydantic import BaseModel, Field, validator
import uvicorn

try:
    from fastmcp import FastMCPClient
    from fastmcp.client.auth import BearerAuth, OAuthAuth
    from fastmcp.client.transports import HTTPTransport, SSETransport, StdioTransport
except ImportError:
    print("Error: FastMCP 2.0 is not installed. Please install it with: pip install fastmcp>=2.0.0", file=sys.stderr)
    sys.exit(1)


# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(sys.stdout),
        logging.FileHandler('fastmcp_service.log')
    ]
)
logger = logging.getLogger(__name__)


# ===========================
# Enums and Constants
# ===========================

class TransportType(str, Enum):
    """Supported transport types"""
    HTTP = "http"
    SSE = "sse"
    STDIO = "stdio"


class AuthType(str, Enum):
    """Supported authentication types"""
    NONE = "none"
    BEARER = "bearer"
    OAUTH = "oauth"


class ClientStatus(str, Enum):
    """MCP client connection status"""
    CONNECTING = "connecting"
    CONNECTED = "connected"
    DISCONNECTED = "disconnected"
    ERROR = "error"


# ===========================
# Request/Response Models
# ===========================

class ConnectRequest(BaseModel):
    """Request model for connecting to an MCP server"""
    transport: TransportType = Field(..., description="Transport type (http, sse, or stdio)")
    mcp_url: Optional[str] = Field(None, description="MCP server URL (required for http/sse)")
    command: Optional[List[str]] = Field(None, description="Command to execute (required for stdio)")
    auth_type: AuthType = Field(default=AuthType.NONE, description="Authentication type")
    bearer_token: Optional[str] = Field(None, description="Bearer token (required if auth_type is bearer)")
    oauth_provider: Optional[str] = Field(None, description="OAuth provider name")
    oauth_client_id: Optional[str] = Field(None, description="OAuth client ID")
    oauth_client_secret: Optional[str] = Field(None, description="OAuth client secret")
    oauth_auth_url: Optional[str] = Field(None, description="OAuth authorization URL")
    oauth_token_url: Optional[str] = Field(None, description="OAuth token URL")
    oauth_redirect_uri: Optional[str] = Field(None, description="OAuth redirect URI")
    oauth_scopes: Optional[List[str]] = Field(None, description="OAuth scopes")
    name: Optional[str] = Field(default=None, description="Client name for identification")
    version: str = Field(default="1.0.0", description="Client version")
    timeout: Optional[int] = Field(default=30, description="Connection timeout in seconds")

    @validator('mcp_url')
    def validate_mcp_url(cls, v, values):
        """Validate MCP URL is provided for http/sse transports"""
        if values.get('transport') in [TransportType.HTTP, TransportType.SSE] and not v:
            raise ValueError("mcp_url is required for http/sse transports")
        return v

    @validator('command')
    def validate_command(cls, v, values):
        """Validate command is provided for stdio transport"""
        if values.get('transport') == TransportType.STDIO and not v:
            raise ValueError("command is required for stdio transport")
        return v

    @validator('bearer_token')
    def validate_bearer_token(cls, v, values):
        """Validate bearer token is provided when auth_type is bearer"""
        if values.get('auth_type') == AuthType.BEARER and not v:
            raise ValueError("bearer_token is required when auth_type is bearer")
        return v


class ToolCallRequest(BaseModel):
    """Request model for calling an MCP tool"""
    client_id: str = Field(..., description="Client ID returned from connect endpoint")
    tool_name: str = Field(..., description="Name of the tool to call")
    arguments: Dict[str, Any] = Field(default_factory=dict, description="Tool arguments")
    timeout: Optional[int] = Field(default=60, description="Tool execution timeout in seconds")


class ResourceReadRequest(BaseModel):
    """Request model for reading an MCP resource"""
    client_id: str = Field(..., description="Client ID")
    uri: str = Field(..., description="Resource URI to read")


class PromptGetRequest(BaseModel):
    """Request model for getting an MCP prompt"""
    client_id: str = Field(..., description="Client ID")
    prompt_name: str = Field(..., description="Name of the prompt to get")
    arguments: Dict[str, Any] = Field(default_factory=dict, description="Prompt arguments")


class DisconnectRequest(BaseModel):
    """Request model for disconnecting an MCP client"""
    client_id: str = Field(..., description="Client ID to disconnect")


class ToolResult(BaseModel):
    """Response model for tool execution results"""
    status: str = Field(..., description="Execution status (success, error)")
    result: Optional[Any] = Field(None, description="Tool execution result")
    error: Optional[str] = Field(None, description="Error message if status is error")
    execution_time: Optional[float] = Field(None, description="Execution time in seconds")


class ConnectResponse(BaseModel):
    """Response model for connect endpoint"""
    status: ClientStatus = Field(..., description="Connection status")
    client_id: str = Field(..., description="Unique client identifier")
    tools: List[Dict[str, Any]] = Field(default_factory=list, description="Available tools")
    resources: List[Dict[str, Any]] = Field(default_factory=list, description="Available resources")
    prompts: List[Dict[str, Any]] = Field(default_factory=list, description="Available prompts")
    message: Optional[str] = Field(None, description="Status message")


class DisconnectResponse(BaseModel):
    """Response model for disconnect endpoint"""
    status: str = Field(..., description="Disconnection status")
    client_id: str = Field(..., description="Disconnected client ID")
    message: str = Field(..., description="Status message")


class HealthResponse(BaseModel):
    """Response model for health check"""
    status: str = Field(..., description="Service health status")
    timestamp: str = Field(..., description="Current timestamp")
    active_clients: int = Field(..., description="Number of active MCP clients")
    version: str = Field(..., description="Service version")


# ===========================
# MCP Client Manager
# ===========================

class MCPClientManager:
    """Manages FastMCP client instances with thread-safe operations"""

    def __init__(self):
        self.clients: Dict[str, FastMCPClient] = {}
        self.client_metadata: Dict[str, Dict[str, Any]] = {}
        self.lock = asyncio.Lock()
        logger.info("MCPClientManager initialized")

    async def create_client(self, config: ConnectRequest) -> tuple[str, FastMCPClient]:
        """Create and connect a new FastMCP client"""
        async with self.lock:
            client_id = str(uuid.uuid4())

            try:
                # Create transport based on type
                transport = self._create_transport(config)

                # Create authentication
                auth = self._create_auth(config)

                # Create client name
                client_name = config.name or f"agentapi-mcp-{client_id[:8]}"

                # Create FastMCP client
                client = FastMCPClient(
                    name=client_name,
                    version=config.version,
                    transport=transport,
                    auth=auth
                )

                # Connect to MCP server with timeout
                logger.info(f"Connecting client {client_id} to MCP server...")
                try:
                    await asyncio.wait_for(
                        client.connect(),
                        timeout=config.timeout
                    )
                except asyncio.TimeoutError:
                    raise HTTPException(
                        status_code=status.HTTP_408_REQUEST_TIMEOUT,
                        detail=f"Connection timeout after {config.timeout} seconds"
                    )

                # Store client and metadata
                self.clients[client_id] = client
                self.client_metadata[client_id] = {
                    "created_at": datetime.utcnow().isoformat(),
                    "transport": config.transport.value,
                    "auth_type": config.auth_type.value,
                    "name": client_name,
                    "mcp_url": config.mcp_url,
                    "last_activity": datetime.utcnow().isoformat()
                }

                logger.info(f"Client {client_id} connected successfully")
                return client_id, client

            except Exception as e:
                logger.error(f"Failed to create client {client_id}: {str(e)}")
                logger.error(traceback.format_exc())
                raise HTTPException(
                    status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                    detail=f"Failed to connect to MCP server: {str(e)}"
                )

    def _create_transport(self, config: ConnectRequest):
        """Create transport based on configuration"""
        if config.transport == TransportType.HTTP:
            return HTTPTransport(config.mcp_url)
        elif config.transport == TransportType.SSE:
            return SSETransport(config.mcp_url)
        elif config.transport == TransportType.STDIO:
            return StdioTransport(config.command)
        else:
            raise ValueError(f"Unsupported transport type: {config.transport}")

    def _create_auth(self, config: ConnectRequest):
        """Create authentication based on configuration"""
        if config.auth_type == AuthType.BEARER:
            return BearerAuth(config.bearer_token)
        elif config.auth_type == AuthType.OAUTH:
            return OAuthAuth(
                client_id=config.oauth_client_id,
                client_secret=config.oauth_client_secret,
                auth_url=config.oauth_auth_url,
                token_url=config.oauth_token_url,
                redirect_uri=config.oauth_redirect_uri,
                scopes=config.oauth_scopes or []
            )
        else:
            return None

    def get_client(self, client_id: str) -> FastMCPClient:
        """Get a client by ID"""
        if client_id not in self.clients:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Client {client_id} not found"
            )

        # Update last activity
        if client_id in self.client_metadata:
            self.client_metadata[client_id]["last_activity"] = datetime.utcnow().isoformat()

        return self.clients[client_id]

    async def disconnect_client(self, client_id: str) -> bool:
        """Disconnect and remove a client"""
        async with self.lock:
            if client_id not in self.clients:
                raise HTTPException(
                    status_code=status.HTTP_404_NOT_FOUND,
                    detail=f"Client {client_id} not found"
                )

            try:
                client = self.clients[client_id]
                await client.disconnect()
                del self.clients[client_id]
                del self.client_metadata[client_id]
                logger.info(f"Client {client_id} disconnected successfully")
                return True
            except Exception as e:
                logger.error(f"Failed to disconnect client {client_id}: {str(e)}")
                raise HTTPException(
                    status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                    detail=f"Failed to disconnect client: {str(e)}"
                )

    async def disconnect_all(self):
        """Disconnect all clients (called on shutdown)"""
        async with self.lock:
            client_ids = list(self.clients.keys())
            for client_id in client_ids:
                try:
                    await self.clients[client_id].disconnect()
                    logger.info(f"Client {client_id} disconnected during shutdown")
                except Exception as e:
                    logger.error(f"Error disconnecting client {client_id} during shutdown: {e}")

            self.clients.clear()
            self.client_metadata.clear()

    def get_active_count(self) -> int:
        """Get count of active clients"""
        return len(self.clients)

    def get_client_metadata(self, client_id: str) -> Dict[str, Any]:
        """Get metadata for a client"""
        if client_id not in self.client_metadata:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Client {client_id} not found"
            )
        return self.client_metadata[client_id]


# ===========================
# FastAPI Application
# ===========================

# Global client manager instance
client_manager: Optional[MCPClientManager] = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Manage application lifecycle"""
    global client_manager

    # Startup
    logger.info("Starting FastMCP Service...")
    client_manager = MCPClientManager()
    logger.info("FastMCP Service started successfully")

    yield

    # Shutdown
    logger.info("Shutting down FastMCP Service...")
    if client_manager:
        await client_manager.disconnect_all()
    logger.info("FastMCP Service shutdown complete")


# Create FastAPI application
app = FastAPI(
    title="FastMCP Service",
    description="Production-ready MCP client management service using FastMCP 2.0",
    version="2.0.0",
    lifespan=lifespan
)


# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Configure appropriately for production
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


# Request logging middleware
@app.middleware("http")
async def log_requests(request: Request, call_next):
    """Log all incoming requests"""
    start_time = datetime.utcnow()

    # Log request
    logger.info(f"Request: {request.method} {request.url.path}")

    # Process request
    response = await call_next(request)

    # Log response
    duration = (datetime.utcnow() - start_time).total_seconds()
    logger.info(f"Response: {response.status_code} - Duration: {duration:.3f}s")

    return response


# Exception handler
@app.exception_handler(Exception)
async def global_exception_handler(request: Request, exc: Exception):
    """Global exception handler"""
    logger.error(f"Unhandled exception: {str(exc)}")
    logger.error(traceback.format_exc())

    return JSONResponse(
        status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
        content={
            "status": "error",
            "message": "Internal server error",
            "detail": str(exc) if app.debug else "An unexpected error occurred"
        }
    )


# ===========================
# API Endpoints
# ===========================

@app.get("/health", response_model=HealthResponse)
async def health_check():
    """
    Health check endpoint

    Returns the current health status of the service including:
    - Service status
    - Current timestamp
    - Number of active MCP clients
    - Service version
    """
    return HealthResponse(
        status="healthy",
        timestamp=datetime.utcnow().isoformat(),
        active_clients=client_manager.get_active_count(),
        version="2.0.0"
    )


@app.post("/mcp/connect", response_model=ConnectResponse, status_code=status.HTTP_201_CREATED)
async def connect_mcp(request: ConnectRequest):
    """
    Connect to an MCP server

    Creates a new FastMCP client and connects to the specified MCP server.
    Returns a unique client_id that should be used for subsequent operations.

    Supports multiple transport types:
    - HTTP: Standard HTTP transport
    - SSE: Server-Sent Events transport
    - STDIO: Standard input/output transport (for local processes)

    Supports multiple authentication types:
    - None: No authentication
    - Bearer: Bearer token authentication
    - OAuth: OAuth 2.0 authentication
    """
    try:
        # Create and connect client
        client_id, client = await client_manager.create_client(request)

        # Fetch available tools, resources, and prompts
        tools = []
        resources = []
        prompts = []

        try:
            tool_list = await client.list_tools()
            tools = [tool.dict() if hasattr(tool, 'dict') else tool for tool in tool_list]
        except Exception as e:
            logger.warning(f"Failed to list tools: {str(e)}")

        try:
            resource_list = await client.list_resources()
            resources = [res.dict() if hasattr(res, 'dict') else res for res in resource_list]
        except Exception as e:
            logger.warning(f"Failed to list resources: {str(e)}")

        try:
            prompt_list = await client.list_prompts()
            prompts = [prompt.dict() if hasattr(prompt, 'dict') else prompt for prompt in prompt_list]
        except Exception as e:
            logger.warning(f"Failed to list prompts: {str(e)}")

        return ConnectResponse(
            status=ClientStatus.CONNECTED,
            client_id=client_id,
            tools=tools,
            resources=resources,
            prompts=prompts,
            message=f"Successfully connected to MCP server"
        )

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Connection error: {str(e)}")
        logger.error(traceback.format_exc())
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to connect: {str(e)}"
        )


@app.post("/mcp/call_tool", response_model=ToolResult)
async def call_tool(request: ToolCallRequest):
    """
    Call a tool on an MCP server

    Executes a tool on the connected MCP server with the provided arguments.
    Returns the tool execution result or an error if the tool fails.

    The tool execution is subject to a timeout (default 60 seconds) to prevent
    hanging requests.
    """
    start_time = datetime.utcnow()

    try:
        # Get client
        client = client_manager.get_client(request.client_id)

        # Call tool with timeout
        try:
            result = await asyncio.wait_for(
                client.call_tool(request.tool_name, request.arguments),
                timeout=request.timeout
            )

            # Calculate execution time
            execution_time = (datetime.utcnow() - start_time).total_seconds()

            # Convert result to dict if it has a dict method
            if hasattr(result, 'dict'):
                result = result.dict()

            return ToolResult(
                status="success",
                result=result,
                error=None,
                execution_time=execution_time
            )

        except asyncio.TimeoutError:
            raise HTTPException(
                status_code=status.HTTP_408_REQUEST_TIMEOUT,
                detail=f"Tool execution timeout after {request.timeout} seconds"
            )

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Tool execution error: {str(e)}")
        logger.error(traceback.format_exc())

        execution_time = (datetime.utcnow() - start_time).total_seconds()

        return ToolResult(
            status="error",
            result=None,
            error=str(e),
            execution_time=execution_time
        )


@app.get("/mcp/list_tools")
async def list_tools(client_id: str):
    """
    List available tools from an MCP server

    Returns a list of all tools available on the connected MCP server.
    Each tool includes its name, description, and input schema.
    """
    try:
        client = client_manager.get_client(client_id)
        tools = await client.list_tools()

        # Convert to dict list
        tool_list = [tool.dict() if hasattr(tool, 'dict') else tool for tool in tools]

        return {
            "status": "success",
            "client_id": client_id,
            "tools": tool_list,
            "count": len(tool_list)
        }

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"List tools error: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to list tools: {str(e)}"
        )


@app.get("/mcp/list_resources")
async def list_resources(client_id: str):
    """
    List available resources from an MCP server

    Returns a list of all resources available on the connected MCP server.
    Each resource includes its URI, name, description, and MIME type.
    """
    try:
        client = client_manager.get_client(client_id)
        resources = await client.list_resources()

        # Convert to dict list
        resource_list = [res.dict() if hasattr(res, 'dict') else res for res in resources]

        return {
            "status": "success",
            "client_id": client_id,
            "resources": resource_list,
            "count": len(resource_list)
        }

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"List resources error: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to list resources: {str(e)}"
        )


@app.post("/mcp/read_resource")
async def read_resource(request: ResourceReadRequest):
    """
    Read a resource from an MCP server

    Retrieves the contents of a resource identified by its URI.
    Returns the resource content along with metadata.
    """
    try:
        client = client_manager.get_client(request.client_id)
        result = await client.read_resource(request.uri)

        # Convert to dict if needed
        if hasattr(result, 'dict'):
            result = result.dict()

        return {
            "status": "success",
            "client_id": request.client_id,
            "uri": request.uri,
            "content": result
        }

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Read resource error: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to read resource: {str(e)}"
        )


@app.get("/mcp/list_prompts")
async def list_prompts(client_id: str):
    """
    List available prompts from an MCP server

    Returns a list of all prompts available on the connected MCP server.
    Each prompt includes its name, description, and argument schema.
    """
    try:
        client = client_manager.get_client(client_id)
        prompts = await client.list_prompts()

        # Convert to dict list
        prompt_list = [prompt.dict() if hasattr(prompt, 'dict') else prompt for prompt in prompts]

        return {
            "status": "success",
            "client_id": client_id,
            "prompts": prompt_list,
            "count": len(prompt_list)
        }

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"List prompts error: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to list prompts: {str(e)}"
        )


@app.post("/mcp/get_prompt")
async def get_prompt(request: PromptGetRequest):
    """
    Get a prompt from an MCP server

    Retrieves a prompt with the given name and arguments.
    Returns the rendered prompt content.
    """
    try:
        client = client_manager.get_client(request.client_id)
        result = await client.get_prompt(request.prompt_name, request.arguments)

        # Convert to dict if needed
        if hasattr(result, 'dict'):
            result = result.dict()

        return {
            "status": "success",
            "client_id": request.client_id,
            "prompt_name": request.prompt_name,
            "content": result
        }

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Get prompt error: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to get prompt: {str(e)}"
        )


@app.post("/mcp/disconnect", response_model=DisconnectResponse)
async def disconnect_mcp(request: DisconnectRequest):
    """
    Disconnect from an MCP server

    Disconnects the specified client and cleans up resources.
    The client_id will no longer be valid after this operation.
    """
    try:
        await client_manager.disconnect_client(request.client_id)

        return DisconnectResponse(
            status="disconnected",
            client_id=request.client_id,
            message="Successfully disconnected from MCP server"
        )

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Disconnect error: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to disconnect: {str(e)}"
        )


@app.get("/mcp/clients")
async def list_clients():
    """
    List all active MCP clients

    Returns a list of all currently connected MCP clients with their metadata.
    Useful for debugging and monitoring.
    """
    try:
        clients = []
        for client_id in client_manager.clients.keys():
            metadata = client_manager.get_client_metadata(client_id)
            clients.append({
                "client_id": client_id,
                **metadata
            })

        return {
            "status": "success",
            "clients": clients,
            "count": len(clients)
        }

    except Exception as e:
        logger.error(f"List clients error: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to list clients: {str(e)}"
        )


@app.get("/mcp/client/{client_id}")
async def get_client_info(client_id: str):
    """
    Get information about a specific MCP client

    Returns detailed metadata about a connected MCP client.
    """
    try:
        metadata = client_manager.get_client_metadata(client_id)

        return {
            "status": "success",
            "client_id": client_id,
            "metadata": metadata
        }

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Get client info error: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to get client info: {str(e)}"
        )


# ===========================
# Main Entry Point
# ===========================

def main():
    """Main entry point for running the service"""
    import argparse

    parser = argparse.ArgumentParser(description="FastMCP Service")
    parser.add_argument("--host", default="0.0.0.0", help="Host to bind to")
    parser.add_argument("--port", type=int, default=8080, help="Port to bind to")
    parser.add_argument("--reload", action="store_true", help="Enable auto-reload")
    parser.add_argument("--debug", action="store_true", help="Enable debug mode")

    args = parser.parse_args()

    # Set debug mode on app
    app.debug = args.debug

    # Configure logging level
    if args.debug:
        logging.getLogger().setLevel(logging.DEBUG)

    # Run server
    logger.info(f"Starting FastMCP Service on {args.host}:{args.port}")
    uvicorn.run(
        "fastmcp_service:app",
        host=args.host,
        port=args.port,
        reload=args.reload,
        log_level="debug" if args.debug else "info"
    )


if __name__ == "__main__":
    main()
