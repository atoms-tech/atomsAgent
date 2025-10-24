# Python-Go Integration Strategy for FastMCP

## Problem Statement

FastMCP 2.0 is a Python-based framework, but AgentAPI is written in Go. We need an efficient way to integrate FastMCP's MCP client capabilities into our Go codebase.

## Integration Options Analysis

### Option 1: gopy (Recommended) ⭐
**Pros:**
- Direct Python function calls from Go
- No network overhead
- Type-safe bindings
- Single binary deployment
- Good performance

**Cons:**
- Requires Python runtime in production
- More complex build process
- CGO dependency

**Implementation:**
```go
//go:build cgo
// +build cgo

package main

/*
#cgo pkg-config: python3
#include <Python.h>
*/
import "C"
import "github.com/go-python/gopy/bind"

func main() {
    // Initialize Python
    C.Py_Initialize()
    defer C.Py_Finalize()
    
    // Import FastMCP module
    fastmcp := bind.PyImport_ImportModule("fastmcp_client")
    
    // Call Python functions directly
    client := fastmcp.CallMethod("create_client", args...)
    tools := client.CallMethod("list_tools")
}
```

### Option 2: gRPC Microservice
**Pros:**
- Language agnostic
- High performance
- Built-in load balancing
- Easy to scale independently
- Clear service boundaries

**Cons:**
- Network overhead
- Additional infrastructure
- More complex deployment
- Service discovery needed

**Implementation:**
```protobuf
// fastmcp.proto
syntax = "proto3";

service FastMCPService {
    rpc ConnectMCP(ConnectRequest) returns (ConnectResponse);
    rpc ListTools(ListToolsRequest) returns (ListToolsResponse);
    rpc CallTool(CallToolRequest) returns (CallToolResponse);
    rpc ListResources(ListResourcesRequest) returns (ListResourcesResponse);
}

message ConnectRequest {
    string id = 1;
    string name = 2;
    string type = 3;
    string endpoint = 4;
    map<string, string> auth = 5;
}
```

### Option 3: HTTP Microservice
**Pros:**
- Simple to implement
- Easy debugging
- RESTful interface
- Language agnostic

**Cons:**
- JSON serialization overhead
- HTTP connection overhead
- Less type safety
- More error handling

**Implementation:**
```go
type FastMCPClient struct {
    baseURL string
    client  *http.Client
}

func (c *FastMCPClient) ConnectMCP(config MCPConfig) error {
    resp, err := c.client.Post(c.baseURL+"/connect", "application/json", 
        bytes.NewBuffer(config.ToJSON()))
    // Handle response
}
```

### Option 4: CGO with Python C API
**Pros:**
- Maximum performance
- Direct Python C API access
- No intermediate layers

**Cons:**
- Very complex implementation
- Hard to maintain
- Platform-specific
- Memory management issues

### Option 5: Process Communication (Current)
**Pros:**
- Simple implementation
- No CGO dependency
- Easy to debug

**Cons:**
- High overhead
- Process management complexity
- Not suitable for production
- Error handling difficulties

## Recommended Solution: gopy + gRPC Hybrid

### Architecture

```
AgentAPI (Go) → gopy bindings → FastMCP (Python) → MCP Servers
                ↓
            gRPC Service (Python) → FastMCP (Python) → MCP Servers
```

### Implementation Plan

#### Phase 1: gopy Integration (MVP)
1. Create Python wrapper module for FastMCP
2. Generate Go bindings using gopy
3. Integrate bindings into AgentAPI
4. Test basic MCP operations

#### Phase 2: gRPC Service (Production)
1. Create Python gRPC service with FastMCP
2. Implement connection pooling
3. Add monitoring and metrics
4. Deploy as separate service

#### Phase 3: Hybrid Approach
1. Use gopy for simple operations
2. Use gRPC for complex operations
3. Implement fallback mechanisms
4. Add load balancing

## Detailed Implementation

### 1. Python Wrapper Module

```python
# fastmcp_wrapper.py
import asyncio
from typing import Dict, List, Any
from fastmcp import FastMCPClient
from fastmcp.client.auth import BearerAuth, OAuthAuth
from fastmcp.client.transports import HTTPTransport, SSETransport, StdioTransport

class FastMCPWrapper:
    def __init__(self):
        self.clients: Dict[str, FastMCPClient] = {}
        self.loop = asyncio.new_event_loop()
        asyncio.set_event_loop(self.loop)
    
    def connect_mcp(self, config: Dict[str, Any]) -> bool:
        """Synchronous wrapper for async connect"""
        return self.loop.run_until_complete(self._connect_mcp_async(config))
    
    async def _connect_mcp_async(self, config: Dict[str, Any]) -> bool:
        try:
            # Create transport
            if config["type"] == "http":
                transport = HTTPTransport(config["endpoint"])
            elif config["type"] == "sse":
                transport = SSETransport(config["endpoint"])
            elif config["type"] == "stdio":
                command = config["config"]["command"].split()
                transport = StdioTransport(command)
            
            # Create auth
            auth = None
            if config["auth_type"] == "bearer":
                auth = BearerAuth(config["auth"]["token"])
            elif config["auth_type"] == "oauth":
                auth = OAuthAuth(
                    config["auth"]["client_id"],
                    config["auth"]["client_secret"],
                    config["auth"]["auth_url"],
                    config["auth"]["token_url"]
                )
            
            # Create client
            client = FastMCPClient(
                name=config["name"],
                version="1.0.0",
                transport=transport,
                auth=auth
            )
            
            # Connect
            await client.connect()
            self.clients[config["id"]] = client
            return True
            
        except Exception as e:
            print(f"Failed to connect: {e}")
            return False
    
    def list_tools(self, mcp_id: str) -> List[Dict[str, Any]]:
        """Synchronous wrapper for async list_tools"""
        return self.loop.run_until_complete(self._list_tools_async(mcp_id))
    
    async def _list_tools_async(self, mcp_id: str) -> List[Dict[str, Any]]:
        if mcp_id not in self.clients:
            return []
        
        try:
            tools = await self.clients[mcp_id].list_tools()
            return [tool.dict() for tool in tools]
        except Exception as e:
            print(f"Failed to list tools: {e}")
            return []
    
    def call_tool(self, mcp_id: str, tool_name: str, arguments: Dict[str, Any]) -> Dict[str, Any]:
        """Synchronous wrapper for async call_tool"""
        return self.loop.run_until_complete(self._call_tool_async(mcp_id, tool_name, arguments))
    
    async def _call_tool_async(self, mcp_id: str, tool_name: str, arguments: Dict[str, Any]) -> Dict[str, Any]:
        if mcp_id not in self.clients:
            return {"error": "MCP client not connected"}
        
        try:
            result = await self.clients[mcp_id].call_tool(tool_name, arguments)
            return result.dict()
        except Exception as e:
            return {"error": str(e)}
    
    def disconnect_mcp(self, mcp_id: str) -> bool:
        """Synchronous wrapper for async disconnect"""
        return self.loop.run_until_complete(self._disconnect_mcp_async(mcp_id))
    
    async def _disconnect_mcp_async(self, mcp_id: str) -> bool:
        try:
            if mcp_id in self.clients:
                await self.clients[mcp_id].disconnect()
                del self.clients[mcp_id]
            return True
        except Exception as e:
            print(f"Failed to disconnect: {e}")
            return False

# Global instance
_wrapper = FastMCPWrapper()

def connect_mcp(config: Dict[str, Any]) -> bool:
    return _wrapper.connect_mcp(config)

def list_tools(mcp_id: str) -> List[Dict[str, Any]]:
    return _wrapper.list_tools(mcp_id)

def call_tool(mcp_id: str, tool_name: str, arguments: Dict[str, Any]) -> Dict[str, Any]:
    return _wrapper.call_tool(mcp_id, tool_name, arguments)

def disconnect_mcp(mcp_id: str) -> bool:
    return _wrapper.disconnect_mcp(mcp_id)
```

### 2. Go Bindings with gopy

```go
//go:build cgo
// +build cgo

package mcp

/*
#cgo pkg-config: python3
#include <Python.h>
*/
import "C"
import (
    "context"
    "encoding/json"
    "fmt"
    "unsafe"
)

// FastMCPGoClient wraps Python FastMCP functions
type FastMCPGoClient struct {
    module *C.PyObject
}

// NewFastMCPGoClient creates a new Go client with Python bindings
func NewFastMCPGoClient() (*FastMCPGoClient, error) {
    // Initialize Python
    C.Py_Initialize()
    
    // Import our Python module
    moduleName := C.CString("fastmcp_wrapper")
    defer C.free(unsafe.Pointer(moduleName))
    
    module := C.PyImport_Import(C.PyUnicode_FromString(moduleName))
    if module == nil {
        return nil, fmt.Errorf("failed to import fastmcp_wrapper module")
    }
    
    return &FastMCPGoClient{module: module}, nil
}

// ConnectMCP connects to an MCP server
func (c *FastMCPGoClient) ConnectMCP(ctx context.Context, config MCPConfig) error {
    // Convert Go struct to Python dict
    configDict := c.goToPythonDict(config)
    defer C.Py_DecRef(configDict)
    
    // Call Python function
    funcName := C.CString("connect_mcp")
    defer C.free(unsafe.Pointer(funcName))
    
    func := C.PyObject_GetAttrString(c.module, funcName)
    defer C.Py_DecRef(func)
    
    args := C.PyTuple_New(1)
    C.PyTuple_SetItem(args, 0, configDict)
    defer C.Py_DecRef(args)
    
    result := C.PyObject_CallObject(func, args)
    defer C.Py_DecRef(result)
    
    // Convert Python bool to Go bool
    if C.PyObject_IsTrue(result) == 0 {
        return fmt.Errorf("failed to connect to MCP server")
    }
    
    return nil
}

// ListTools lists available tools
func (c *FastMCPGoClient) ListTools(ctx context.Context, mcpID string) ([]Tool, error) {
    // Convert string to Python string
    pyMCPID := C.PyUnicode_FromString(C.CString(mcpID))
    defer C.Py_DecRef(pyMCPID)
    
    // Call Python function
    funcName := C.CString("list_tools")
    defer C.free(unsafe.Pointer(funcName))
    
    func := C.PyObject_GetAttrString(c.module, funcName)
    defer C.Py_DecRef(func)
    
    args := C.PyTuple_New(1)
    C.PyTuple_SetItem(args, 0, pyMCPID)
    defer C.Py_DecRef(args)
    
    result := C.PyObject_CallObject(func, args)
    defer C.Py_DecRef(result)
    
    // Convert Python list to Go slice
    return c.pythonListToGoTools(result)
}

// CallTool calls a tool on the MCP server
func (c *FastMCPGoClient) CallTool(ctx context.Context, mcpID, toolName string, arguments map[string]any) (map[string]any, error) {
    // Convert arguments to Python dict
    pyArgs := c.goMapToPythonDict(arguments)
    defer C.Py_DecRef(pyArgs)
    
    // Create Python tuple for arguments
    args := C.PyTuple_New(3)
    C.PyTuple_SetItem(args, 0, C.PyUnicode_FromString(C.CString(mcpID)))
    C.PyTuple_SetItem(args, 1, C.PyUnicode_FromString(C.CString(toolName)))
    C.PyTuple_SetItem(args, 2, pyArgs)
    defer C.Py_DecRef(args)
    
    // Call Python function
    funcName := C.CString("call_tool")
    defer C.free(unsafe.Pointer(funcName))
    
    func := C.PyObject_GetAttrString(c.module, funcName)
    defer C.Py_DecRef(func)
    
    result := C.PyObject_CallObject(func, args)
    defer C.Py_DecRef(result)
    
    // Convert Python dict to Go map
    return c.pythonDictToGoMap(result)
}

// Helper functions for Python-Go conversion
func (c *FastMCPGoClient) goToPythonDict(config MCPConfig) *C.PyObject {
    dict := C.PyDict_New()
    
    // Add fields to Python dict
    C.PyDict_SetItemString(dict, C.CString("id"), C.PyUnicode_FromString(C.CString(config.ID)))
    C.PyDict_SetItemString(dict, C.CString("name"), C.PyUnicode_FromString(C.CString(config.Name)))
    C.PyDict_SetItemString(dict, C.CString("type"), C.PyUnicode_FromString(C.CString(config.Type)))
    C.PyDict_SetItemString(dict, C.CString("endpoint"), C.PyUnicode_FromString(C.CString(config.Endpoint)))
    C.PyDict_SetItemString(dict, C.CString("auth_type"), C.PyUnicode_FromString(C.CString(config.AuthType)))
    
    // Add config and auth as Python dicts
    configDict := c.goMapToPythonDict(config.Config)
    authDict := c.goMapToPythonDict(config.Auth)
    C.PyDict_SetItemString(dict, C.CString("config"), configDict)
    C.PyDict_SetItemString(dict, C.CString("auth"), authDict)
    
    return dict
}

func (c *FastMCPGoClient) goMapToPythonDict(goMap map[string]any) *C.PyObject {
    dict := C.PyDict_New()
    
    for key, value := range goMap {
        pyKey := C.PyUnicode_FromString(C.CString(key))
        pyValue := c.goValueToPython(value)
        C.PyDict_SetItem(dict, pyKey, pyValue)
        C.Py_DecRef(pyKey)
        C.Py_DecRef(pyValue)
    }
    
    return dict
}

func (c *FastMCPGoClient) goValueToPython(value any) *C.PyObject {
    switch v := value.(type) {
    case string:
        return C.PyUnicode_FromString(C.CString(v))
    case int:
        return C.PyLong_FromLong(C.long(v))
    case bool:
        if v {
            return C.Py_True
        }
        return C.Py_False
    case map[string]any:
        return c.goMapToPythonDict(v)
    case []any:
        return c.goSliceToPythonList(v)
    default:
        // Convert to string as fallback
        return C.PyUnicode_FromString(C.CString(fmt.Sprintf("%v", v)))
    }
}

func (c *FastMCPGoClient) goSliceToPythonList(slice []any) *C.PyObject {
    list := C.PyList_New(C.Py_ssize_t(len(slice)))
    
    for i, value := range slice {
        pyValue := c.goValueToPython(value)
        C.PyList_SetItem(list, C.Py_ssize_t(i), pyValue)
    }
    
    return list
}

func (c *FastMCPGoClient) pythonListToGoTools(pyList *C.PyObject) ([]Tool, error) {
    size := C.PyList_Size(pyList)
    tools := make([]Tool, 0, int(size))
    
    for i := C.Py_ssize_t(0); i < size; i++ {
        item := C.PyList_GetItem(pyList, i)
        defer C.Py_DecRef(item)
        
        tool, err := c.pythonDictToGoTool(item)
        if err != nil {
            return nil, err
        }
        
        tools = append(tools, tool)
    }
    
    return tools, nil
}

func (c *FastMCPGoClient) pythonDictToGoTool(pyDict *C.PyObject) (Tool, error) {
    var tool Tool
    
    // Extract name
    if name := C.PyDict_GetItemString(pyDict, C.CString("name")); name != nil {
        tool.Name = C.GoString(C.PyUnicode_AsUTF8(name))
    }
    
    // Extract description
    if desc := C.PyDict_GetItemString(pyDict, C.CString("description")); desc != nil {
        tool.Description = C.GoString(C.PyUnicode_AsUTF8(desc))
    }
    
    // Extract input schema
    if schema := C.PyDict_GetItemString(pyDict, C.CString("inputSchema")); schema != nil {
        schemaMap, err := c.pythonDictToGoMap(schema)
        if err != nil {
            return tool, err
        }
        tool.InputSchema = schemaMap
    }
    
    return tool, nil
}

func (c *FastMCPGoClient) pythonDictToGoMap(pyDict *C.PyObject) (map[string]any, error) {
    result := make(map[string]any)
    
    // Get all keys
    keys := C.PyDict_Keys(pyDict)
    defer C.Py_DecRef(keys)
    
    size := C.PyList_Size(keys)
    for i := C.Py_ssize_t(0); i < size; i++ {
        key := C.PyList_GetItem(keys, i)
        defer C.Py_DecRef(key)
        
        value := C.PyDict_GetItem(pyDict, key)
        defer C.Py_DecRef(value)
        
        goKey := C.GoString(C.PyUnicode_AsUTF8(key))
        goValue := c.pythonValueToGo(value)
        
        result[goKey] = goValue
    }
    
    return result, nil
}

func (c *FastMCPGoClient) pythonValueToGo(pyValue *C.PyObject) any {
    if C.PyUnicode_Check(pyValue) != 0 {
        return C.GoString(C.PyUnicode_AsUTF8(pyValue))
    } else if C.PyLong_Check(pyValue) != 0 {
        return int(C.PyLong_AsLong(pyValue))
    } else if C.PyBool_Check(pyValue) != 0 {
        return C.PyObject_IsTrue(pyValue) != 0
    } else if C.PyDict_Check(pyValue) != 0 {
        result, _ := c.pythonDictToGoMap(pyValue)
        return result
    } else if C.PyList_Check(pyValue) != 0 {
        size := C.PyList_Size(pyValue)
        result := make([]any, 0, int(size))
        
        for i := C.Py_ssize_t(0); i < size; i++ {
            item := C.PyList_GetItem(pyValue, i)
            defer C.Py_DecRef(item)
            result = append(result, c.pythonValueToGo(item))
        }
        
        return result
    }
    
    // Fallback to string representation
    str := C.PyObject_Str(pyValue)
    defer C.Py_DecRef(str)
    return C.GoString(C.PyUnicode_AsUTF8(str))
}

// Close cleans up Python resources
func (c *FastMCPGoClient) Close() {
    C.Py_DecRef(c.module)
    C.Py_Finalize()
}
```

### 3. Build Configuration

```makefile
# Makefile
.PHONY: build-python build-go test clean

# Build Python module
build-python:
	pip install -r requirements.txt
	python -m py_compile fastmcp_wrapper.py

# Build Go with CGO
build-go:
	CGO_ENABLED=1 go build -o agentapi main.go

# Generate gopy bindings
generate-bindings:
	gopy build -output=bindings fastmcp_wrapper.py

# Test the integration
test:
	go test -v ./lib/mcp/...

# Clean build artifacts
clean:
	rm -rf bindings/
	rm -f agentapi
```

### 4. Docker Configuration

```dockerfile
# Dockerfile with Python and Go
FROM python:3.11-slim AS python-base

# Install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy Python module
COPY fastmcp_wrapper.py /usr/local/lib/python3.11/site-packages/

FROM golang:1.21-alpine AS go-builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev pkgconfig python3-dev

# Copy source code
COPY . .

# Build with CGO
ENV CGO_ENABLED=1
RUN go build -o agentapi main.go

# Final stage
FROM python:3.11-slim

# Copy Python runtime
COPY --from=python-base /usr/local/lib/python3.11 /usr/local/lib/python3.11

# Copy Go binary
COPY --from=go-builder /app/agentapi /usr/local/bin/

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Set Python path
ENV PYTHONPATH=/usr/local/lib/python3.11/site-packages

EXPOSE 3284
CMD ["agentapi"]
```

## Performance Comparison

| Approach | Latency | Memory | Complexity | Deployment |
|----------|---------|--------|------------|------------|
| gopy | ~1ms | Low | High | Single binary |
| gRPC | ~5ms | Medium | Medium | Microservice |
| HTTP | ~10ms | Medium | Low | Microservice |
| Process | ~50ms | High | Low | Process management |
| CGO | ~0.5ms | Low | Very High | Single binary |

## Recommendation

For the MVP, use **gopy** with the following approach:

1. **Phase 1**: Implement gopy bindings for basic MCP operations
2. **Phase 2**: Add gRPC service for production scaling
3. **Phase 3**: Implement hybrid approach with fallback

This provides the best balance of performance, maintainability, and deployment simplicity for your enterprise requirements.