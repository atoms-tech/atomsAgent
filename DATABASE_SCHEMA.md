# AgentAPI Database Schema

**Created**: October 24, 2025  
**Database**: Supabase PostgreSQL  
**Status**: Ready to use

---

## Overview

The AgentAPI database schema supports:
- ✅ Multi-agent management (CCRouter, Droid, custom)
- ✅ Dynamic model discovery and configuration
- ✅ Chat session and message history
- ✅ Agent execution tracking
- ✅ Agent health monitoring
- ✅ Performance metrics
- ✅ Circuit breaker state management

---

## Tables

### 1. `agents`
Stores agent configurations (CCRouter, Droid, etc.)

**Columns**:
- `id` (UUID): Primary key
- `name` (VARCHAR): Unique agent name
- `type` (VARCHAR): Agent type ('ccrouter', 'droid', 'custom')
- `description` (TEXT): Agent description
- `enabled` (BOOLEAN): Active/inactive flag
- `config` (JSONB): Agent-specific configuration
- `created_at`, `updated_at`: Timestamps

**Default Data**:
- CCRouter (VertexAI/Gemini)
- Droid (Multi-model via OpenRouter)

---

### 2. `models`
Available LLM models per agent

**Columns**:
- `id` (UUID): Primary key
- `agent_id` (UUID): Foreign key to agents
- `name` (VARCHAR): Model name (unique per agent)
- `display_name` (VARCHAR): User-friendly name
- `description` (TEXT): What the model does
- `provider` (VARCHAR): 'gemini', 'openrouter', etc
- `model_id` (VARCHAR): Provider-specific model ID
- `enabled` (BOOLEAN): Active/inactive flag
- `config` (JSONB): Temperature, max_tokens, etc
- `created_at`, `updated_at`: Timestamps

**Default Models**:
- CCRouter: gemini-1.5-pro, gemini-1.5-flash
- Droid: claude-3-opus, gpt-4

---

### 3. `chat_sessions`
Conversation sessions between users and agents

**Columns**:
- `id` (UUID): Primary key
- `user_id` (VARCHAR): WorkOS user ID
- `org_id` (VARCHAR): Organization ID
- `model_id` (UUID): Selected model
- `agent_id` (UUID): Selected agent
- `title` (VARCHAR): Session title
- `metadata` (JSONB): Custom data
- `created_at`, `updated_at`, `last_message_at`: Timestamps

---

### 4. `chat_messages`
Individual messages in a session

**Columns**:
- `id` (UUID): Primary key
- `session_id` (UUID): Foreign key to sessions
- `role` (VARCHAR): 'user', 'assistant', 'system'
- `content` (TEXT): Message content
- `tokens_in`, `tokens_out`, `tokens_total`: Token counts
- `metadata` (JSONB): Custom data
- `created_at`: Timestamp

---

### 5. `agent_executions`
Track each agent's responses and performance

**Columns**:
- `id` (UUID): Primary key
- `session_id`, `message_id` (UUID): Foreign keys
- `agent_id`, `model_id` (UUID): Which agent/model executed
- `status` (VARCHAR): 'pending', 'running', 'success', 'failed'
- `input_tokens`, `output_tokens`, `total_tokens`: Token counts
- `latency_ms` (INTEGER): Execution time
- `error_message`, `response_content` (TEXT): Results
- `metadata` (JSONB): Custom data
- `created_at`, `completed_at`: Timestamps

---

### 6. `agent_health`
Agent availability and error tracking

**Columns**:
- `id` (UUID): Primary key
- `agent_id` (UUID): Foreign key to agents
- `status` (VARCHAR): 'healthy', 'degraded', 'unhealthy'
- `last_check` (TIMESTAMP): Last health check time
- `last_error` (TEXT): Most recent error
- `consecutive_failures` (INTEGER): Failure count
- `metadata` (JSONB): Custom data
- `created_at`, `updated_at`: Timestamps

---

### 7. `agent_metrics`
Daily performance metrics per agent

**Columns**:
- `id` (UUID): Primary key
- `agent_id` (UUID): Foreign key to agents
- `date` (DATE): Metric date
- `total_requests`, `successful_requests`, `failed_requests` (INTEGER)
- `avg_latency_ms` (DECIMAL): Average execution time
- `avg_tokens_in`, `avg_tokens_out` (INTEGER): Token averages
- `created_at`: Timestamp

---

### 8. `circuit_breaker_state`
Circuit breaker pattern state for fault tolerance

**Columns**:
- `id` (UUID): Primary key
- `agent_id` (UUID): Foreign key to agents
- `state` (VARCHAR): 'closed', 'open', 'half_open'
- `failure_count`, `success_count` (INTEGER): Counts
- `last_failure_time`, `last_success_time` (TIMESTAMP)
- `opened_at` (TIMESTAMP): When circuit opened
- `metadata` (JSONB): Custom data
- `created_at`, `updated_at`: Timestamps

---

## Views

### `v_recent_sessions`
Quick view of recent sessions with message counts

```sql
SELECT * FROM v_recent_sessions 
WHERE org_id = 'your-org-id'
ORDER BY updated_at DESC
LIMIT 10;
```

### `v_agent_status`
Agent status summary with health and metrics

```sql
SELECT * FROM v_agent_status;
```

---

## Indexes

Performance indexes created for:
- Chat sessions (user_id, org_id, created_at)
- Chat messages (session_id, created_at)
- Agent executions (session_id, agent_id, status, created_at)
- Models (agent_id, enabled)
- Agent metrics (agent_id, date)

---

## Setup

### 1. Apply Schema
```bash
bash setup_supabase_schema.sh
```

### 2. Verify
```bash
# Connect to Supabase
psql $DATABASE_URL

# List tables
\dt

# List views
\dv

# Check agents
SELECT * FROM agents;
SELECT * FROM models;
```

---

## Usage Examples

### Create a Chat Session
```sql
INSERT INTO chat_sessions (user_id, org_id, agent_id, model_id, title)
SELECT 
    'user_123',
    'org_456',
    agents.id,
    models.id,
    'Chat with Gemini'
FROM agents
JOIN models ON agents.id = models.agent_id
WHERE agents.name = 'ccrouter' 
  AND models.name = 'gemini-1.5-pro'
RETURNING id;
```

### Add Message to Session
```sql
INSERT INTO chat_messages (session_id, role, content, tokens_total)
VALUES ('session-uuid', 'user', 'Hello, what can you do?', 10)
RETURNING id;
```

### Track Agent Execution
```sql
INSERT INTO agent_executions (
    session_id, message_id, agent_id, model_id, 
    status, input_tokens, output_tokens, latency_ms
)
VALUES (
    'session-uuid', 'message-uuid', 'agent-uuid', 'model-uuid',
    'success', 10, 150, 2500
)
RETURNING id;
```

### Get Agent Performance
```sql
SELECT 
    a.name as agent,
    COALESCE(am.total_requests, 0) as requests_today,
    COALESCE(am.successful_requests, 0) as successful,
    COALESCE(am.avg_latency_ms, 0) as avg_latency_ms
FROM agents a
LEFT JOIN agent_metrics am ON a.id = am.agent_id 
  AND am.date = CURRENT_DATE
ORDER BY am.total_requests DESC;
```

---

## Maintenance

### Archive Old Sessions
```sql
-- Delete sessions older than 90 days
DELETE FROM chat_sessions 
WHERE created_at < CURRENT_DATE - INTERVAL '90 days'
  AND user_id NOT IN (
    SELECT DISTINCT user_id FROM chat_sessions 
    WHERE created_at >= CURRENT_DATE - INTERVAL '30 days'
  );
```

### Update Agent Health
```sql
UPDATE agent_health
SET status = 'healthy', last_check = NOW()
WHERE agent_id = 'agent-uuid';
```

### Aggregate Daily Metrics
```sql
-- Should be run daily via scheduled job
INSERT INTO agent_metrics (agent_id, date, total_requests, successful_requests, failed_requests, avg_latency_ms)
SELECT 
    ae.agent_id,
    CURRENT_DATE,
    COUNT(*),
    COUNT(CASE WHEN status = 'success' THEN 1 END),
    COUNT(CASE WHEN status = 'failed' THEN 1 END),
    AVG(latency_ms)
FROM agent_executions ae
WHERE DATE(ae.created_at) = CURRENT_DATE
GROUP BY ae.agent_id
ON CONFLICT (agent_id, date) DO UPDATE
SET total_requests = EXCLUDED.total_requests,
    successful_requests = EXCLUDED.successful_requests,
    failed_requests = EXCLUDED.failed_requests,
    avg_latency_ms = EXCLUDED.avg_latency_ms;
```

---

## Backup & Recovery

### Backup
```bash
pg_dump $DATABASE_URL > agentapi_backup.sql
```

### Restore
```bash
psql $DATABASE_URL < agentapi_backup.sql
```

---

## Performance Tips

1. **Regular Vacuuming**: Supabase handles this automatically
2. **Index Usage**: Verify indexes are being used with EXPLAIN
3. **Partition Large Tables**: Consider partitioning chat_messages by date
4. **Archive Old Data**: Delete sessions > 90 days old monthly
5. **Connection Pooling**: Use pgBouncer for many connections

---

## Next Steps

1. ✅ Create schema: `bash setup_supabase_schema.sh`
2. ✅ Verify tables: `psql $DATABASE_URL -c "\dt"`
3. ✅ Start AgentAPI: `bash run.sh docker` or `bash run.sh binary`
4. ✅ Create first session: See "Usage Examples" above

---

**Status**: Ready to use  
**Last Updated**: October 24, 2025
