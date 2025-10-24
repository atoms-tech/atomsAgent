-- Migration: Create system_prompts table
-- Description: Creates the system_prompts table for storing multi-tenant prompts
-- Author: AgentAPI Team
-- Date: 2025-10-23

-- Create the system_prompts table
CREATE TABLE IF NOT EXISTS system_prompts (
    id VARCHAR(255) PRIMARY KEY,
    scope VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    template TEXT,
    org_id VARCHAR(255),
    user_id VARCHAR(255),
    priority INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Ensure valid scope values
    CONSTRAINT valid_scope CHECK (scope IN ('global', 'org', 'user')),

    -- Ensure global prompts have no org_id or user_id
    CONSTRAINT valid_global_scope CHECK (
        scope != 'global' OR (org_id IS NULL AND user_id IS NULL)
    ),

    -- Ensure org prompts have org_id but no user_id
    CONSTRAINT valid_org_scope CHECK (
        scope != 'org' OR (org_id IS NOT NULL AND user_id IS NULL)
    ),

    -- Ensure user prompts have both org_id and user_id
    CONSTRAINT valid_user_scope CHECK (
        scope != 'user' OR (org_id IS NOT NULL AND user_id IS NOT NULL)
    )
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_system_prompts_scope
    ON system_prompts(scope);

CREATE INDEX IF NOT EXISTS idx_system_prompts_org_id
    ON system_prompts(org_id)
    WHERE org_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_system_prompts_user_id
    ON system_prompts(user_id)
    WHERE user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_system_prompts_enabled
    ON system_prompts(enabled)
    WHERE enabled = true;

CREATE INDEX IF NOT EXISTS idx_system_prompts_priority
    ON system_prompts(priority DESC);

-- Create composite index for common query patterns
CREATE INDEX IF NOT EXISTS idx_system_prompts_composite
    ON system_prompts(enabled, scope, priority DESC)
    WHERE enabled = true;

-- Comments for documentation
COMMENT ON TABLE system_prompts IS
    'Stores system prompts with multi-tenant scoping (global, org, user)';

COMMENT ON COLUMN system_prompts.id IS
    'Unique identifier for the prompt';

COMMENT ON COLUMN system_prompts.scope IS
    'Scope of the prompt: global, org, or user';

COMMENT ON COLUMN system_prompts.content IS
    'Static content of the prompt (used if template is empty)';

COMMENT ON COLUMN system_prompts.template IS
    'Go template string (rendered if provided, otherwise content is used)';

COMMENT ON COLUMN system_prompts.org_id IS
    'Organization ID (required for org and user scopes, null for global)';

COMMENT ON COLUMN system_prompts.user_id IS
    'User ID (required for user scope, null for global and org)';

COMMENT ON COLUMN system_prompts.priority IS
    'Priority for ordering (higher values appear first in composition)';

COMMENT ON COLUMN system_prompts.enabled IS
    'Whether the prompt is active (only enabled prompts are composed)';

COMMENT ON COLUMN system_prompts.created_at IS
    'Timestamp when the prompt was created';

COMMENT ON COLUMN system_prompts.updated_at IS
    'Timestamp when the prompt was last updated';
