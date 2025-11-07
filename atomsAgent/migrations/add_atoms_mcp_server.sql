-- Migration: Add Atoms MCP Server to database
-- Purpose: Register the production Atoms MCP server as a system-level internal server
-- Date: 2024-11-07

-- Note: This assumes you're using the newer mcp_servers table structure
-- If you're using mcp_configurations table, adjust accordingly

-- First, ensure the is_internal column exists
ALTER TABLE mcp_servers 
ADD COLUMN IF NOT EXISTS is_internal BOOLEAN DEFAULT FALSE;

-- Insert or update the production Atoms MCP server
INSERT INTO mcp_servers (
    id,
    name,
    namespace,
    description,
    server_url,
    transport_type,
    auth_type,
    scope,
    tier,
    is_internal,
    enabled,
    created_at,
    updated_at
) VALUES (
    gen_random_uuid(),
    'Atoms MCP',
    'tech.atoms.mcp',
    'Core Atoms platform tools - workspace, entity, relationship, workflow, and data operations',
    'https://mcp.atoms.tech/api/mcp',
    'http',
    'bearer',  -- Uses bearer token (AuthKit JWT)
    'system',  -- System-level server available to all users
    'free',
    TRUE,      -- Internal server - receives user AuthKit JWT
    TRUE,      -- Enabled
    NOW(),
    NOW()
)
ON CONFLICT (namespace) 
DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    server_url = EXCLUDED.server_url,
    transport_type = EXCLUDED.transport_type,
    auth_type = EXCLUDED.auth_type,
    scope = EXCLUDED.scope,
    is_internal = EXCLUDED.is_internal,
    enabled = EXCLUDED.enabled,
    updated_at = NOW();

-- Also add development version for local testing
INSERT INTO mcp_servers (
    id,
    name,
    namespace,
    description,
    server_url,
    transport_type,
    auth_type,
    scope,
    tier,
    is_internal,
    enabled,
    created_at,
    updated_at
) VALUES (
    gen_random_uuid(),
    'Atoms MCP (Dev)',
    'tech.atoms.mcp.dev',
    'Development instance of Atoms MCP for local testing',
    'http://localhost:8000/api/mcp',
    'http',
    'bearer',
    'system',
    'free',
    TRUE,
    FALSE,  -- Disabled by default - enable for local dev
    NOW(),
    NOW()
)
ON CONFLICT (namespace) 
DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    server_url = EXCLUDED.server_url,
    is_internal = EXCLUDED.is_internal,
    updated_at = NOW();

-- Verify the servers were added
SELECT 
    id,
    name,
    namespace,
    server_url,
    is_internal,
    enabled,
    scope
FROM mcp_servers 
WHERE namespace LIKE 'tech.atoms.mcp%'
ORDER BY name;

