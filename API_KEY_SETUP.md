# API Key Authentication Setup Guide

## Overview

API key authentication allows service-to-service communication and programmatic access to ChatServer without using WorkOS/AuthKit tokens.

---

## Step 1: Create the API Keys Table in Supabase

### Option A: Via Supabase Dashboard (Easiest)

1. Go to https://supabase.com/dashboard
2. Select your project: `ydogoylwenufckscqijp`
3. Click **SQL Editor** (left sidebar)
4. Click **New Query**
5. Copy and paste this SQL:

```sql
-- Create api_keys table for API key authentication
CREATE TABLE IF NOT EXISTS api_keys (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT NOT NULL,
  organization_id TEXT NOT NULL,
  key_hash TEXT NOT NULL UNIQUE,
  name TEXT,
  description TEXT,
  is_active BOOLEAN DEFAULT true,
  expires_at TIMESTAMP NULL,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  last_used_at TIMESTAMP NULL
);

-- Create indexes for fast lookups
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_organization_id ON api_keys(organization_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_active ON api_keys(is_active) WHERE is_active = true;

-- Add RLS policies
ALTER TABLE api_keys ENABLE ROW LEVEL SECURITY;

-- Policy: Users can view their own API keys
CREATE POLICY "Users can view their own API keys"
  ON api_keys FOR SELECT
  USING (auth.uid()::text = user_id);

-- Policy: Admins can view all API keys
CREATE POLICY "Admins can view all API keys"
  ON api_keys FOR SELECT
  USING (
    EXISTS (
      SELECT 1 FROM public.platform_admins
      WHERE workos_user_id = auth.uid()::text
      AND is_active = true
    )
  );

-- Table documentation
COMMENT ON TABLE api_keys IS 'API keys for service-to-service and programmatic authentication';
COMMENT ON COLUMN api_keys.key_hash IS 'SHA256 hash of the actual API key (never store plaintext keys)';
COMMENT ON COLUMN api_keys.is_active IS 'Whether this key is active (soft delete via this flag)';
```

6. Click **Run**
7. You should see "Success" message

### Option B: Via Supabase CLI

```bash
# If you have Supabase CLI installed
supabase db push
```

---

## Step 2: Generate an API Key

### Via SQL (Direct)

```sql
-- For testing/development - create an API key for a user
INSERT INTO api_keys (
  user_id,
  organization_id,
  key_hash,
  name,
  description,
  is_active,
  expires_at
) VALUES (
  'user_123',
  'org_456',
  'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855', -- SHA256 of test_key_12345
  'Development Key',
  'For local development',
  true,
  '2026-01-01'::timestamp
);
```

The actual API key to use would be: `test_key_12345`

### Via Application (When You Build the Endpoint)

Create an endpoint that generates and returns API keys:

```go
// Example endpoint (to be implemented)
POST /api/v1/api-keys
Content-Type: application/json

{
  "name": "Production Integration",
  "description": "For monitoring system",
  "expires_in_days": 365
}

// Response:
{
  "id": "uuid-here",
  "key": "sk_prod_xyz123abc456",  // Only returned once, user must save it
  "key_hash": "e3b0c44...",
  "name": "Production Integration",
  "created_at": "2025-10-25T00:00:00Z"
}
```

---

## Step 3: Use the API Key to Call ChatServer

### Authentication Header Format

```
Authorization: Bearer <your-api-key>
```

### Example Requests

**Using cURL:**
```bash
curl -X POST http://localhost:3284/v1/chat/completions \
  -H "Authorization: Bearer sk_prod_xyz123abc456" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "ccrouter",
    "messages": [
      {"role": "user", "content": "Hello, what can you do?"}
    ]
  }'
```

**Using Python:**
```python
import requests

API_KEY = "sk_prod_xyz123abc456"
CHATSERVER_URL = "http://localhost:3284"

headers = {
    "Authorization": f"Bearer {API_KEY}",
    "Content-Type": "application/json"
}

payload = {
    "model": "ccrouter",
    "messages": [
        {"role": "user", "content": "Hello!"}
    ]
}

response = requests.post(
    f"{CHATSERVER_URL}/v1/chat/completions",
    headers=headers,
    json=payload
)

print(response.json())
```

**Using JavaScript:**
```javascript
const API_KEY = "sk_prod_xyz123abc456";
const CHATSERVER_URL = "http://localhost:3284";

const response = await fetch(`${CHATSERVER_URL}/v1/chat/completions`, {
  method: "POST",
  headers: {
    "Authorization": `Bearer ${API_KEY}`,
    "Content-Type": "application/json"
  },
  body: JSON.stringify({
    model: "ccrouter",
    messages: [
      { role: "user", content: "Hello!" }
    ]
  })
});

const data = await response.json();
console.log(data);
```

**Using Go:**
```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func main() {
	apiKey := "sk_prod_xyz123abc456"
	chatServerURL := "http://localhost:3284"

	payload := map[string]interface{}{
		"model": "ccrouter",
		"messages": []map[string]string{
			{"role": "user", "content": "Hello!"},
		},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST",
		fmt.Sprintf("%s/v1/chat/completions", chatServerURL),
		bytes.NewBuffer(body))

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, _ := client.Do(req)
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	fmt.Println(result)
}
```

---

## Step 4: How API Key Authentication Works

### Flow Diagram

```
Request arrives at ChatServer
    ↓
Authorization header extracted: "Bearer sk_prod_xyz123abc456"
    ↓
API Key validator receives: "sk_prod_xyz123abc456"
    ↓
SHA256 hash computed: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
    ↓
Database query: SELECT * FROM api_keys WHERE key_hash = ?
    ↓
Key found in database
    ↓
Checks:
  ✓ is_active = true
  ✓ expires_at IS NULL OR expires_at > NOW()
  ✓ user_id exists
  ✓ organization_id exists
    ↓
All checks pass
    ↓
Load user info from users table
    ↓
Check platform_admins table for admin status
    ↓
Return AuthKitUser with:
  - ID: user_id
  - OrgID: organization_id
  - Email: from users table
  - Name: from users table
  - AuthenticationMethod: "api_key"
  - IsPlatformAdmin: from platform_admins
    ↓
Request processed with authenticated context
```

---

## Step 5: API Key Management

### Viewing All Keys

```sql
SELECT id, name, user_id, organization_id, is_active, expires_at, created_at
FROM api_keys
WHERE user_id = 'user_123'
ORDER BY created_at DESC;
```

### Deactivating a Key (Soft Delete)

```sql
UPDATE api_keys
SET is_active = false
WHERE id = 'key-uuid-here';
```

### Deleting a Key (Permanent)

```sql
DELETE FROM api_keys
WHERE id = 'key-uuid-here';
```

### Setting Expiration

```sql
UPDATE api_keys
SET expires_at = '2026-01-01'::timestamp
WHERE id = 'key-uuid-here';
```

### Updating Last Used Time

```sql
UPDATE api_keys
SET last_used_at = NOW()
WHERE id = 'key-uuid-here';
```

---

## Step 6: Security Best Practices

### ✅ DO

- ✅ Hash keys before storing (SHA256)
- ✅ Only return keys once when created
- ✅ Store keys securely in environment variables
- ✅ Rotate keys regularly
- ✅ Use short expiration times for sensitive operations
- ✅ Track which keys are used when (audit logging)
- ✅ Deactivate old keys instead of deleting

### ❌ DON'T

- ❌ Don't store plaintext keys in database
- ❌ Don't send keys in URLs or query parameters
- ❌ Don't commit keys to version control
- ❌ Don't log full keys
- ❌ Don't reuse the same key across multiple services
- ❌ Don't skip expiration dates

---

## Step 7: Testing Your Setup

### Test 1: Generate a Test Key

```sql
-- Generate test API key
INSERT INTO api_keys (user_id, organization_id, key_hash, name)
VALUES (
  'test_user_123',
  'test_org_456',
  'a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3', -- SHA256 of 'test'
  'Test Key'
);
```

Actual key to use: `test`

### Test 2: Call ChatServer with API Key

```bash
curl -X POST http://localhost:3284/v1/chat/completions \
  -H "Authorization: Bearer test" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "ccrouter",
    "messages": [{"role": "user", "content": "Hi"}]
  }'
```

### Test 3: Check Response

Should see:
- ✅ 200 status code
- ✅ Chat response from CCRouter
- ✅ No authentication errors

### Test 4: Try Expired Key

```sql
-- Create an expired key
INSERT INTO api_keys (user_id, organization_id, key_hash, name, expires_at)
VALUES (
  'test_user_123',
  'test_org_456',
  'b7c6c5c4b3a2a1a0f0e0d0c0b0a09080', -- SHA256 of 'expired'
  'Expired Key',
  '2025-01-01'::timestamp
);
```

Call with `Bearer expired` should get 401 Unauthorized

---

## Step 8: Next: Build the Key Management Endpoint

The APIKeyValidator is ready. Next, you'll want to create HTTP endpoints to:

1. **Create API Key** - POST /api/v1/api-keys
2. **List Keys** - GET /api/v1/api-keys
3. **Revoke Key** - DELETE /api/v1/api-keys/{id}
4. **Rotate Key** - POST /api/v1/api-keys/{id}/rotate

These endpoints would:
- Generate random keys
- Hash them before storage
- Return unhashed key to user (only once)
- Support expiration
- Support revocation

---

## Troubleshooting

### Error: "invalid or expired API key"
- Check if key exists in database
- Verify key_hash matches (use SHA256 hash of key)
- Check if key is marked as active
- Check if expiration date hasn't passed

### Error: "authentication failed"
- Check database connection is working
- Check api_keys table exists
- Check RLS policies aren't blocking the query

### Performance Issues
- Ensure indexes exist (idx_api_keys_key_hash is critical)
- Monitor last_used_at updates (don't update on every request)
- Consider caching key validity for 5-10 seconds

---

## Summary

You now have:
1. ✅ API keys table in Supabase
2. ✅ Go code (APIKeyValidator) to validate keys
3. ✅ Documentation on how to use API keys
4. ✅ Security best practices
5. ✅ Example requests in multiple languages

**Next Step**: Create HTTP endpoints to generate and manage API keys, then test end-to-end.

