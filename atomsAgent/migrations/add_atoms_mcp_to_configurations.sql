-- Migration: Add Atoms MCP to mcp_configurations table
-- Purpose: Register Atoms MCP server for users (alternative to mcp_servers table)
-- Date: 2024-11-07
-- Note: Use this if you're using the mcp_configurations table instead of mcp_servers

-- First, check which table structure you're using:
-- SELECT table_name FROM information_schema.tables 
-- WHERE table_name IN ('mcp_servers', 'mcp_configurations');

-- Option 1: If using mcp_configurations table
-- Add is_internal column
ALTER TABLE mcp_configurations 
ADD COLUMN IF NOT EXISTS is_internal BOOLEAN DEFAULT FALSE;

-- Add comment
COMMENT ON COLUMN mcp_configurations.is_internal IS 
'Indicates if this is an internal/first-party MCP server that should receive the user AuthKit JWT';

-- Insert Atoms MCP as a platform-level configuration
-- Note: You'll need to replace 'SYSTEM_USER_ID' with your actual system user ID
INSERT INTO mcp_configurations (
    id,
    user_id,
    org_id,
    name,
    type,
    endpoint,
    auth_type,
    scope,
    enabled,
    is_internal,
    description,
    created_at,
    updated_at,
    created_by,
    updated_by
) VALUES (
    gen_random_uuid(),
    NULL,  -- NULL user_id means platform-level
    NULL,  -- NULL org_id means platform-level
    'Atoms MCP',
    'http',
    'https://mcp.atoms.tech/api/mcp',
    'bearer',
    'platform',
    TRUE,
    TRUE,  -- Internal server
    'Core Atoms platform tools',
    NOW(),
    NOW(),
    'system',
    'system'
)
ON CONFLICT (user_id, name) 
DO UPDATE SET
    endpoint = EXCLUDED.endpoint,
    is_internal = EXCLUDED.is_internal,
    enabled = EXCLUDED.enabled,
    updated_at = NOW();

-- Update existing Atoms MCP entries to be marked as internal
UPDATE mcp_configurations 
SET is_internal = TRUE 
WHERE name ILIKE '%atoms%mcp%' 
   OR name ILIKE '%atoms-mcp%'
   OR endpoint ILIKE '%mcp.atoms.tech%';

-- Verify
SELECT 
    id,
    name,
    endpoint,
    is_internal,
    enabled,
    scope
FROM mcp_configurations 
WHERE is_internal = TRUE;

