# AgentAPI Schema Analysis - Supabase Reuse Opportunities

**Date**: October 24, 2025
**Analysis**: Can we leverage existing tables instead of creating new ones?

---

## What You Already Have ✅

### 1. **profiles** table (19 columns)
- `id` (uuid) - User identifier
- `email` (text) - User email
- `full_name` (text)
- `workos_id` (text) - Auth integration
- `current_organization_id` (uuid) - Current org context
- `last_login_at` (timestamp)
- `preferences` (jsonb) - User settings
- `status` (enum) - User status

**⚡ Insight**: You already have a `profiles` table! This IS your users table.

### 2. **organization_members** table (13 columns)
- Links `user_id` → `organization_id`
- Has `role` (enum) - admin/member roles
- Has `permissions` (jsonb) - Fine-grained permissions
- Tracks `last_active_at`

**⚡ Insight**: This handles user-org relationships better than a separate `users` table.

### 3. **mcp_sessions** table (7 columns)
- `user_id` (uuid)
- `session_id` (text)
- `mcp_state` (jsonb) - MCP configuration state
- `oauth_data` (jsonb) - OAuth tokens
- `created_at`, `expires_at`, `updated_at`

**⚡ Insight**: You ALREADY have session storage for MCP! This can replace:
- `user_sessions` table
- `oauth_states` table
- `mcp_oauth_tokens` table

### 4. **organizations** table (25 columns)
- `id`, `name`, `slug`
- `settings` (jsonb) - Organization settings
- `metadata` (jsonb) - Custom metadata
- Already has timestamps, soft deletes, audit fields

**⚡ Insight**: This is comprehensive. Skip creating a duplicate.

---

## What You're Missing ❌

For AgentAPI to work, you ONLY need these NEW tables:

### **Agent Management** (3 tables - MINIMAL)
1. `agents` - Agent configurations (ccrouter, droid)
2. `models` - LLM models per agent
3. `agent_health` - Agent status tracking

### **Chat Operations** (2 tables - CORE)
1. `chat_sessions` - Conversation sessions
2. `chat_messages` - Individual messages

### **Metrics & Monitoring** (2 tables - OPTIONAL)
1. `agent_executions` - Execution history
2. `agent_metrics` - Daily performance stats

### **Optional for Advanced Features** (1 table)
1. `circuit_breaker_state` - Fault tolerance (can use Redis instead)

---

## Optimized Migration Strategy

### ❌ **DO NOT CREATE** (Already have better equivalents)
```
users                   → Use: profiles + organization_members
user_sessions          → Use: mcp_sessions (already exists!)
oauth_states           → Use: mcp_sessions.mcp_state (JSONB)
mcp_oauth_tokens       → Use: mcp_sessions.oauth_data (JSONB)
mcp_configurations     → Use: organizations.settings (JSONB)
system_prompts         → Use: organizations.metadata or mcp_sessions.mcp_state (JSONB)
```

### ✅ **DO CREATE** (Core AgentAPI tables only)
```
agents                 → Agent definitions (ccrouter, droid)
models                 → Models per agent
chat_sessions          → Chat conversations
chat_messages          → Individual messages
agent_health           → Agent status (lightweight)
agent_executions       → Execution history (optional, can log to file)
agent_metrics          → Performance stats (optional, can use Redis)
```

---

## Redis Integration Strategy

Instead of database tables, use **Upstash Redis** for:

### **Session State** (TTL-based)
```
Key: session:{session_id}
Value: {
  user_id, org_id, model_id, agent_id, messages[], metadata
}
TTL: 24 hours
```

### **OAuth Tokens** (TTL-based)
```
Key: oauth_token:{user_id}:{provider}
Value: {access_token, refresh_token, expires_at}
TTL: Until refresh needed
```

### **Agent Health** (Real-time)
```
Key: agent:health:{agent_id}
Value: {status, last_check, failures, metadata}
TTL: 5 minutes
```

### **Circuit Breaker State** (Temporary)
```
Key: circuit_breaker:{agent_id}
Value: {state, failure_count, timestamp}
TTL: Auto-resets
```

### **Chat Session Cache**
```
Key: chat:{session_id}:messages
Value: [compressed message history]
TTL: 1 hour (persist to DB on expiry)
```

---

## Minimal Database-Only Approach

**ABSOLUTE MINIMUM for AgentAPI**:

```sql
-- Table 1: Agent Definitions
CREATE TABLE agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) UNIQUE NOT NULL,
    type VARCHAR(50) NOT NULL,  -- 'ccrouter', 'droid'
    enabled BOOLEAN DEFAULT true,
    config JSONB,  -- provider, location, etc
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Table 2: LLM Models
CREATE TABLE models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id),
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    provider VARCHAR(100),
    model_id VARCHAR(255),
    enabled BOOLEAN DEFAULT true,
    config JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(agent_id, name)
);

-- Table 3: Chat Sessions (reference profiles, not users!)
CREATE TABLE chat_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES profiles(id),  -- Use existing profiles!
    org_id UUID NOT NULL REFERENCES organizations(id),
    model_id UUID REFERENCES models(id),
    agent_id UUID REFERENCES agents(id),
    title VARCHAR(255),
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Table 4: Chat Messages
CREATE TABLE chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL,  -- 'user', 'assistant'
    content TEXT NOT NULL,
    tokens_in INTEGER,
    tokens_out INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Optional: Agent Health (or use Redis)
CREATE TABLE agent_health (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL UNIQUE REFERENCES agents(id),
    status VARCHAR(50),  -- 'healthy', 'degraded', 'unhealthy'
    last_check TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

**That's it!** 5 tables instead of 15+

---

## Configuration Storage Strategy

### Use existing **mcp_sessions.mcp_state** for:
```json
{
  "mcp_name": "ccrouter",
  "type": "stdio",
  "command": "/usr/local/bin/ccrouter",
  "env": {"VERTEX_AI_PROJECT_ID": "..."},
  "auth_type": "bearer",
  "enabled": true
}
```

### Use **organizations.settings** for:
```json
{
  "default_model": "gemini-1.5-pro",
  "default_agent": "ccrouter",
  "temperature": 0.7,
  "max_tokens": 2048,
  "system_prompt": "You are a helpful assistant..."
}
```

---

## Summary: Recommended Approach

| Need | Current Approach | ✅ Better Approach | Storage |
|------|-----------------|-------------------|---------|
| User identification | Create `users` table | Use `profiles` table | DB |
| User-Org relationship | Create `users` table | Use `organization_members` | DB |
| User sessions | Create `user_sessions` table | Use `mcp_sessions` | DB/Redis |
| OAuth tokens | Create `mcp_oauth_tokens` table | Use `mcp_sessions.oauth_data` | DB/Redis |
| MCP configuration | Create `mcp_configurations` table | Use `mcp_sessions.mcp_state` | DB/Redis |
| System prompts | Create `system_prompts` table | Use `organizations.settings` | DB |
| Agent definitions | Create `agents` table | **STILL NEED** | DB |
| Available models | Create `models` table | **STILL NEED** | DB |
| Chat sessions | Create `chat_sessions` table | **STILL NEED** | DB |
| Chat messages | Create `chat_messages` table | **STILL NEED** | DB |
| Agent health | Create `agent_health` table | **Can use Redis** | Redis |
| Execution history | Create `agent_executions` table | **Can use Redis/Logs** | Redis/Logs |
| Performance metrics | Create `agent_metrics` table | **Can use Redis/CloudWatch** | Redis |
| Circuit breaker | Create `circuit_breaker_state` table | **Use Redis** | Redis |

---

## Final Recommendation

**Create only 5 tables** (instead of 15):
1. `agents` (required)
2. `models` (required)
3. `chat_sessions` (required)
4. `chat_messages` (required)
5. `agent_health` (optional, use Redis instead)

**Leverage Upstash Redis** for:
- Session state caching
- OAuth token storage
- Agent health status
- Circuit breaker state
- Execution history logs
- Performance metrics

This approach:
- ✅ 70% less database overhead
- ✅ Reuses existing user/org infrastructure
- ✅ Uses Redis for ephemeral data
- ✅ Maintains clean separation of concerns
- ✅ Scales better for high-throughput chat
- ✅ Easier to maintain

---

**Should I create this minimal migration instead?**
