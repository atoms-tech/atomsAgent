-- Rollback Migration: Drop system_prompts table
-- Description: Rolls back the creation of system_prompts table
-- Author: AgentAPI Team
-- Date: 2025-10-23

-- Drop indexes first
DROP INDEX IF EXISTS idx_system_prompts_composite;
DROP INDEX IF EXISTS idx_system_prompts_priority;
DROP INDEX IF EXISTS idx_system_prompts_enabled;
DROP INDEX IF EXISTS idx_system_prompts_user_id;
DROP INDEX IF EXISTS idx_system_prompts_org_id;
DROP INDEX IF EXISTS idx_system_prompts_scope;

-- Drop the table
DROP TABLE IF EXISTS system_prompts;
