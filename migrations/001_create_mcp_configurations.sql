-- Create MCP configurations table
-- This table stores MCP server configurations with tenant isolation

CREATE TABLE IF NOT EXISTS mcp_configurations (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('http', 'sse', 'stdio')),
    endpoint TEXT,
    command TEXT,
    args TEXT, -- JSON array of command arguments
    auth_type VARCHAR(50) NOT NULL CHECK (auth_type IN ('none', 'bearer', 'oauth', 'api_key')),
    auth_token TEXT, -- Encrypted
    auth_header VARCHAR(255),
    config TEXT, -- JSON object for additional configuration
    scope VARCHAR(50) NOT NULL CHECK (scope IN ('org', 'user')),
    org_id VARCHAR(255),
    user_id VARCHAR(255),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255) NOT NULL,
    updated_by VARCHAR(255) NOT NULL,

    -- Indexes for efficient querying
    INDEX idx_org_id (org_id),
    INDEX idx_user_id (user_id),
    INDEX idx_scope (scope),
    INDEX idx_enabled (enabled),
    INDEX idx_type (type),

    -- Constraints for tenant isolation
    CONSTRAINT chk_org_scope CHECK (
        (scope = 'org' AND org_id IS NOT NULL AND user_id IS NULL) OR
        (scope = 'user' AND user_id IS NOT NULL AND org_id IS NULL)
    )
);

-- Create audit log table for MCP operations
CREATE TABLE IF NOT EXISTS mcp_audit_log (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    org_id VARCHAR(255) NOT NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(255),
    details TEXT, -- JSON object
    ip_address VARCHAR(45),
    user_agent TEXT,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Indexes
    INDEX idx_user_id (user_id),
    INDEX idx_org_id (org_id),
    INDEX idx_action (action),
    INDEX idx_resource (resource_type, resource_id),
    INDEX idx_timestamp (timestamp)
);
