-- Seed Migration: Insert default system prompts
-- Description: Creates default global prompts for the system
-- Author: AgentAPI Team
-- Date: 2025-10-23

-- Insert default global prompt
INSERT INTO system_prompts (
    id,
    scope,
    content,
    template,
    org_id,
    user_id,
    priority,
    enabled,
    created_at,
    updated_at
) VALUES (
    'global-base-assistant',
    'global',
    'You are Claude, an AI assistant created by Anthropic. You are helpful, harmless, and honest.',
    NULL,
    NULL,
    NULL,
    100,
    true,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
) ON CONFLICT (id) DO NOTHING;

-- Insert default safety prompt
INSERT INTO system_prompts (
    id,
    scope,
    content,
    template,
    org_id,
    user_id,
    priority,
    enabled,
    created_at,
    updated_at
) VALUES (
    'global-safety',
    'global',
    'Important safety guidelines:
- Do not provide instructions for illegal or harmful activities
- Respect user privacy and data confidentiality
- Be truthful and acknowledge uncertainty when appropriate
- Decline requests that could cause harm',
    NULL,
    NULL,
    NULL,
    90,
    true,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
) ON CONFLICT (id) DO NOTHING;

-- Insert default coding assistance prompt
INSERT INTO system_prompts (
    id,
    scope,
    content,
    template,
    org_id,
    user_id,
    priority,
    enabled,
    created_at,
    updated_at
) VALUES (
    'global-coding-assistant',
    'global',
    'When helping with code:
- Provide clear explanations
- Follow best practices and security guidelines
- Write clean, well-commented code
- Consider edge cases and error handling
- Suggest testing approaches',
    NULL,
    NULL,
    NULL,
    80,
    true,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
) ON CONFLICT (id) DO NOTHING;

-- Insert templated session info prompt
INSERT INTO system_prompts (
    id,
    scope,
    content,
    template,
    org_id,
    user_id,
    priority,
    enabled,
    created_at,
    updated_at
) VALUES (
    'global-session-context',
    'global',
    NULL,
    'Current session context:
- User: {{.UserID}}
- Organization: {{.OrgID}}
- Date: {{.Date}}
- Time: {{.Time}}

Please provide responses appropriate to this context.',
    NULL,
    NULL,
    70,
    true,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
) ON CONFLICT (id) DO NOTHING;
