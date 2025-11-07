-- Migration: Add is_internal column to mcp_servers table
-- Purpose: Mark internal MCP servers (like Atoms MCP) that should use AuthKit JWT
-- Date: 2024-11-07

-- Add is_internal column to mcp_servers table
-- This column identifies first-party MCP servers that should receive the user's AuthKit JWT
ALTER TABLE mcp_servers 
ADD COLUMN IF NOT EXISTS is_internal BOOLEAN DEFAULT FALSE;

-- Add comment to explain the column
COMMENT ON COLUMN mcp_servers.is_internal IS 
'Indicates if this is an internal/first-party MCP server that should receive the user AuthKit JWT for authentication';

-- Update existing Atoms MCP servers to be marked as internal
UPDATE mcp_servers 
SET is_internal = TRUE 
WHERE name ILIKE '%atoms%mcp%' 
   OR name ILIKE '%atoms-mcp%'
   OR namespace ILIKE '%atoms%';

-- Verify the migration
SELECT 
    name, 
    namespace, 
    is_internal,
    auth_type
FROM mcp_servers 
WHERE is_internal = TRUE;

