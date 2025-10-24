# JWT Authentication - Quick Start Guide

Get up and running with JWT authentication in 5 minutes.

## Prerequisites

1. Go 1.23.2 or later
2. A Supabase project
3. Users table with `org_id` and `role` columns

## Step 1: Install Dependencies

The JWT dependency is already included in go.mod:
```bash
go mod download
```

## Step 2: Set Environment Variable

```bash
export SUPABASE_URL="https://your-project.supabase.co"
```

## Step 3: Configure Supabase Custom Claims

Run this SQL in your Supabase SQL Editor:

```sql
-- Ensure users table has required columns
ALTER TABLE public.users
  ADD COLUMN IF NOT EXISTS org_id UUID REFERENCES organizations(id),
  ADD COLUMN IF NOT EXISTS role TEXT DEFAULT 'user';

-- Create custom claims hook
CREATE OR REPLACE FUNCTION custom_access_token_hook(event jsonb)
RETURNS jsonb
LANGUAGE plpgsql
AS $$
DECLARE
  claims jsonb;
  user_role text;
  user_org_id uuid;
BEGIN
  -- Get user data from your users table
  SELECT role, org_id INTO user_role, user_org_id
  FROM public.users
  WHERE id = (event->>'user_id')::uuid;

  -- Extract existing claims
  claims := event->'claims';

  -- Add custom claims
  IF user_role IS NOT NULL THEN
    claims := jsonb_set(claims, '{role}', to_jsonb(user_role));
  ELSE
    claims := jsonb_set(claims, '{role}', to_jsonb('user'));
  END IF;

  IF user_org_id IS NOT NULL THEN
    claims := jsonb_set(claims, '{org_id}', to_jsonb(user_org_id::text));
  END IF;

  -- Update event with new claims
  event := jsonb_set(event, '{claims}', claims);

  RETURN event;
END;
$$;

-- Grant necessary permissions
GRANT EXECUTE ON FUNCTION custom_access_token_hook TO supabase_auth_admin;
REVOKE EXECUTE ON FUNCTION custom_access_token_hook FROM PUBLIC;
```

Then enable the hook in your Supabase Dashboard:
1. Go to Authentication → Hooks
2. Enable "Custom access token hook"
3. Select `custom_access_token_hook` function

## Step 4: Add Middleware to Your Server

```go
package main

import (
    "log"
    "log/slog"
    "net/http"

    "github.com/coder/agentapi/lib/auth"
    "github.com/go-chi/chi/v5"
)

func main() {
    // Create logger
    logger := slog.Default()

    // Create authentication middleware
    authMiddleware, err := auth.NewAuthMiddleware(auth.AuthConfig{
        Logger: logger,
        SkipPaths: []string{
            "/health",      // Health check endpoint
            "/public",      // Public endpoints
            "/docs",        // API documentation
        },
    })
    if err != nil {
        log.Fatal("Failed to create auth middleware:", err)
    }

    // Create router
    router := chi.NewRouter()

    // Apply authentication to all routes
    router.Use(authMiddleware.Middleware)

    // Register your routes
    router.Get("/api/sessions", listSessions)
    router.Post("/api/sessions", createSession)

    // Start server
    logger.Info("Server starting on :3284")
    http.ListenAndServe(":3284", router)
}

func listSessions(w http.ResponseWriter, r *http.Request) {
    // Extract user info from token
    userID, orgID, err := auth.GetUserFromContext(r.Context())
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Your business logic here
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"user_id":"` + userID + `","org_id":"` + orgID + `"}`))
}

func createSession(w http.ResponseWriter, r *http.Request) {
    // Get full claims if needed
    claims, err := auth.GetClaimsFromContext(r.Context())
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Check if user is admin
    if auth.IsAdmin(r.Context()) {
        // Admin-specific logic
    }

    // Your business logic here
    log.Printf("User %s (%s) creating session", claims.Email, claims.Sub)
    w.WriteHeader(http.StatusCreated)
}
```

## Step 5: Test Your API

### Get a JWT Token from Supabase

```bash
# Using Supabase CLI
supabase auth login --email user@example.com --password yourpassword

# Or using curl
curl -X POST "https://your-project.supabase.co/auth/v1/token?grant_type=password" \
  -H "Content-Type: application/json" \
  -H "apikey: YOUR_SUPABASE_ANON_KEY" \
  -d '{
    "email": "user@example.com",
    "password": "yourpassword"
  }'
```

### Call Your Protected API

```bash
# Store the token
export JWT_TOKEN="eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."

# Call your API
curl -X GET "http://localhost:3284/api/sessions" \
  -H "Authorization: Bearer $JWT_TOKEN"
```

### Expected Response

```json
{
  "user_id": "123e4567-e89b-12d3-a456-426614174000",
  "org_id": "789e4567-e89b-12d3-a456-426614174000"
}
```

## Step 6: Add Role-Based Access Control

### Protect Admin Endpoints

```go
// Admin-only routes
router.Group(func(r chi.Router) {
    r.Use(auth.RequireAdminRole(logger))

    r.Get("/api/admin/users", listAllUsers)
    r.Delete("/api/admin/sessions/{id}", deleteAnySession)
})

func listAllUsers(w http.ResponseWriter, r *http.Request) {
    // Only admins can access this
    // Your admin logic here
}
```

### Protect User Endpoints

```go
// User routes (both user and admin can access)
router.Group(func(r chi.Router) {
    r.Use(auth.RequireUserRole(logger))

    r.Get("/api/profile", getProfile)
    r.Put("/api/profile", updateProfile)
})
```

## Common Patterns

### Pattern 1: Check Ownership

```go
func getSession(w http.ResponseWriter, r *http.Request) {
    sessionID := chi.URLParam(r, "id")
    userID, orgID, _ := auth.GetUserFromContext(r.Context())

    session := fetchSessionFromDB(sessionID)

    // Ensure user owns this session
    if session.UserID != userID {
        // Allow admins to access any session in their org
        if !auth.IsAdmin(r.Context()) || session.OrgID != orgID {
            http.Error(w, "Forbidden", http.StatusForbidden)
            return
        }
    }

    // Return session
    respondJSON(w, session)
}
```

### Pattern 2: Different Logic for Roles

```go
func getStats(w http.ResponseWriter, r *http.Request) {
    userID, orgID, _ := auth.GetUserFromContext(r.Context())

    var stats interface{}

    if auth.IsAdmin(r.Context()) {
        // Admins see org-wide stats
        stats = getOrgStats(orgID)
    } else {
        // Users see only their stats
        stats = getUserStats(userID)
    }

    respondJSON(w, stats)
}
```

### Pattern 3: Audit Logging

```go
func performAction(w http.ResponseWriter, r *http.Request) {
    claims, _ := auth.GetClaimsFromContext(r.Context())

    // Your business logic
    result := doSomething()

    // Log the action
    auditLog.Info("Action performed",
        "user_id", claims.Sub,
        "email", claims.Email,
        "org_id", claims.OrgID,
        "role", claims.Role,
        "action", "create_session",
        "result", result,
    )

    respondJSON(w, result)
}
```

## Troubleshooting

### Error: "SUPABASE_URL environment variable not set"

**Solution**: Set the environment variable:
```bash
export SUPABASE_URL="https://your-project.supabase.co"
```

### Error: "Missing org_id claim"

**Solution**: Ensure custom claims hook is configured and enabled in Supabase.

### Error: "Invalid token: token has expired"

**Solution**: Get a fresh token from Supabase. Tokens expire after 1 hour by default.

### Error: "Missing Authorization header"

**Solution**: Include the Bearer token in the request:
```bash
curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:3284/api/sessions
```

### Error: "Invalid Authorization header format"

**Solution**: Ensure you're using the format `Bearer <token>`, not just the token.

## Next Steps

1. **Read the full documentation**: `/lib/auth/README.md`
2. **See examples**: `/lib/auth/example_integration.go`
3. **Review implementation details**: `/lib/auth/IMPLEMENTATION_SUMMARY.md`
4. **Add tests**: See `/lib/auth/middleware_test.go` for examples

## Security Checklist

Before deploying to production:

- [ ] HTTPS enabled for all endpoints
- [ ] Supabase custom claims hook configured
- [ ] Environment variables set correctly
- [ ] Skip paths configured for public endpoints
- [ ] Rate limiting enabled
- [ ] Audit logging implemented
- [ ] Error messages don't leak sensitive info
- [ ] Token expiration properly handled
- [ ] Role-based access control tested
- [ ] Multi-tenant isolation verified

## Support

For questions or issues:
- Check the README: `/lib/auth/README.md`
- Review examples: `/lib/auth/example_integration.go`
- Run tests: `go test ./lib/auth -v`
- Open an issue on GitHub

## License

MIT License
