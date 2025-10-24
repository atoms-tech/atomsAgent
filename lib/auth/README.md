# JWT Authentication Middleware

Production-ready JWT authentication middleware for AgentAPI with Supabase integration.

## Features

- **Supabase JWT Validation**: Full integration with Supabase authentication
- **JWKS Support**: Automatic public key fetching and caching
- **Role-Based Access Control**: Admin and user role enforcement
- **Context Integration**: Claims stored in request context for easy access
- **Comprehensive Error Handling**: Detailed error messages for debugging
- **Production-Ready**: Background key refresh, rate limiting, and caching

## Installation

Add the JWT dependency to your project:

```bash
go get github.com/golang-jwt/jwt/v5
```

## Quick Start

### 1. Basic Setup

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
    // Create authentication middleware
    authMiddleware, err := auth.NewAuthMiddleware(auth.AuthConfig{
        JWKSUrl:  "", // Optional - will use SUPABASE_URL env var
        Issuer:   "https://your-project.supabase.co/auth/v1",
        Audience: "authenticated",
        Logger:   slog.Default(),
        SkipPaths: []string{
            "/health",
            "/public",
            "/docs",
        },
        RefreshPeriod: 5 * time.Minute,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Create router
    router := chi.NewRouter()

    // Apply authentication to all routes
    router.Use(authMiddleware.Middleware)

    // Register routes
    router.Get("/api/sessions", listSessions)
    router.Post("/api/sessions", createSession)

    http.ListenAndServe(":3284", router)
}
```

### 2. Environment Variables

Set up your environment:

```bash
# Required
export SUPABASE_URL="https://your-project.supabase.co"

# Optional - if not using env var, configure directly
export SUPABASE_ANON_KEY="your-anon-key"
export SUPABASE_SERVICE_ROLE_KEY="your-service-role-key"
```

### 3. Using Claims in Handlers

Extract user information from the request context:

```go
func createSession(w http.ResponseWriter, r *http.Request) {
    // Get user ID and org ID
    userID, orgID, err := auth.GetUserFromContext(r.Context())
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Get full claims
    claims, err := auth.GetClaimsFromContext(r.Context())
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Use claims
    log.Printf("User %s (%s) from org %s is creating a session",
        claims.Email, userID, orgID)

    // Your business logic here
}
```

## Role-Based Access Control

### Admin-Only Endpoints

Require admin role for specific endpoints:

```go
router := chi.NewRouter()

// Apply auth middleware first
router.Use(authMiddleware.Middleware)

// Admin-only routes
router.Group(func(r chi.Router) {
    r.Use(auth.RequireAdminRole(slog.Default()))

    r.Get("/api/admin/users", listAllUsers)
    r.Delete("/api/admin/sessions/{id}", deleteAnySession)
})
```

### User-Level Endpoints

Require authenticated user (any role):

```go
router.Group(func(r chi.Router) {
    r.Use(auth.RequireUserRole(slog.Default()))

    r.Get("/api/sessions", listMySessions)
    r.Post("/api/sessions", createSession)
})
```

### Custom Role Requirements

Create custom role checks:

```go
// Require specific role
router.Group(func(r chi.Router) {
    r.Use(auth.RequireRole(auth.RoleAdmin, slog.Default()))

    r.Get("/api/admin/dashboard", adminDashboard)
})

// Check role in handler
func someHandler(w http.ResponseWriter, r *http.Request) {
    if auth.IsAdmin(r.Context()) {
        // Admin-specific logic
    } else {
        // Regular user logic
    }
}
```

## Integration with Existing Code

### Updating multitenant.go

Replace the placeholder functions with proper implementations:

```go
// lib/api/multitenant.go

import "github.com/coder/agentapi/lib/auth"

func getUserIDFromContext(ctx context.Context) string {
    userID, _, err := auth.GetUserFromContext(ctx)
    if err != nil {
        return ""
    }
    return userID
}

func getOrgIDFromContext(ctx context.Context) string {
    _, orgID, err := auth.GetUserFromContext(ctx)
    if err != nil {
        return ""
    }
    return orgID
}
```

### Applying to HTTP Server

Update your HTTP server setup:

```go
// lib/httpapi/server.go

import "github.com/coder/agentapi/lib/auth"

func NewServer(ctx context.Context, config ServerConfig) (*Server, error) {
    router := chi.NewMux()

    // ... existing middleware ...

    // Add authentication middleware
    authMiddleware, err := auth.NewAuthMiddleware(auth.AuthConfig{
        Logger: logger,
        SkipPaths: []string{
            "/health",
            "/chat",    // Public chat interface
            "/docs",    // API documentation
        },
    })
    if err != nil {
        return nil, xerrors.Errorf("failed to create auth middleware: %w", err)
    }

    router.Use(authMiddleware.Middleware)

    // ... rest of server setup ...
}
```

## JWT Token Format

### Expected Claims

The middleware expects Supabase JWTs with the following structure:

```json
{
  "sub": "user-uuid",              // User ID (required)
  "email": "user@example.com",     // User email (required)
  "org_id": "org-uuid",            // Organization ID (required)
  "role": "admin",                 // User role: "admin" or "user" (required)
  "aud": "authenticated",          // Audience
  "iss": "https://project.supabase.co/auth/v1",  // Issuer
  "exp": 1234567890,               // Expiration timestamp (required)
  "iat": 1234567800                // Issued at timestamp
}
```

### Custom Claims in Supabase

To add custom claims like `org_id` and `role` to Supabase JWTs, use a PostgreSQL function:

```sql
-- Create function to add custom claims
CREATE OR REPLACE FUNCTION custom_access_token_hook(event jsonb)
RETURNS jsonb
LANGUAGE plpgsql
AS $$
DECLARE
  claims jsonb;
  user_role text;
  user_org_id uuid;
BEGIN
  -- Get user data
  SELECT role, org_id INTO user_role, user_org_id
  FROM public.users
  WHERE id = (event->>'user_id')::uuid;

  -- Build claims
  claims := event->'claims';

  IF user_role IS NOT NULL THEN
    claims := jsonb_set(claims, '{role}', to_jsonb(user_role));
  END IF;

  IF user_org_id IS NOT NULL THEN
    claims := jsonb_set(claims, '{org_id}', to_jsonb(user_org_id::text));
  END IF;

  -- Update the event claims
  event := jsonb_set(event, '{claims}', claims);

  RETURN event;
END;
$$;

-- Enable the hook in your Supabase project settings
```

## Error Handling

The middleware provides detailed error messages:

### 401 Unauthorized

```json
{
  "error": "Unauthorized",
  "message": "Missing Authorization header"
}
```

```json
{
  "error": "Unauthorized",
  "message": "Invalid Authorization header format. Expected: Bearer <token>"
}
```

```json
{
  "error": "Unauthorized",
  "message": "Invalid token: token has expired"
}
```

### 403 Forbidden

```json
{
  "error": "Forbidden",
  "message": "Role 'admin' required"
}
```

## Testing

### Unit Tests

Run the test suite:

```bash
go test ./lib/auth -v
```

### Integration Tests

Test with a real Supabase instance:

```bash
# Set up test environment
export SUPABASE_URL="https://test-project.supabase.co"
export TEST_JWT_TOKEN="eyJhbGc..."

# Run integration tests
go test ./lib/auth -tags=integration -v
```

### Manual Testing

Test the authentication flow manually:

```bash
# Get a JWT token from Supabase
curl -X POST https://your-project.supabase.co/auth/v1/token \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password"}'

# Use the token to call your API
curl -X GET http://localhost:3284/api/sessions \
  -H "Authorization: Bearer eyJhbGc..."
```

## Security Best Practices

### 1. HTTPS Only

Always use HTTPS in production:

```go
// Redirect HTTP to HTTPS
http.ListenAndServe(":80", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    http.Redirect(w, r, "https://"+r.Host+r.URL.String(), http.StatusMovedPermanently)
}))

// Serve on HTTPS
http.ListenAndServeTLS(":443", "cert.pem", "key.pem", router)
```

### 2. Token Validation

The middleware automatically validates:
- Token signature using Supabase public keys
- Token expiration
- Required claims (sub, email, org_id, role)
- Issuer and audience (if configured)

### 3. Key Rotation

The middleware automatically:
- Fetches new keys every 5 minutes (configurable)
- Caches keys for performance
- Handles key rotation gracefully

### 4. Rate Limiting

Consider adding rate limiting for authentication endpoints:

```go
import "github.com/go-chi/httprate"

router.Use(httprate.LimitByIP(100, 1*time.Minute))
```

### 5. Audit Logging

Log authentication events for security monitoring:

```go
func (am *AuthMiddleware) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // ... authentication logic ...

        if err != nil {
            am.logger.Warn("Authentication failed",
                "path", r.URL.Path,
                "ip", r.RemoteAddr,
                "error", err,
            )
        } else {
            am.logger.Info("Authentication successful",
                "user_id", claims.Sub,
                "org_id", claims.OrgID,
                "path", r.URL.Path,
            )
        }

        // ... rest of handler ...
    })
}
```

## Troubleshooting

### "Missing kid in token header"

The JWT token must include a `kid` (key ID) in the header. Ensure your Supabase project is properly configured.

### "Failed to fetch JWKS"

Check that:
1. `SUPABASE_URL` is set correctly
2. The JWKS endpoint is accessible: `https://your-project.supabase.co/auth/v1/jwks`
3. Network connectivity is working

### "Invalid token: token has expired"

The token has expired. Request a new token from Supabase.

### "Missing org_id claim"

Ensure you've set up the custom claims hook in Supabase (see "Custom Claims in Supabase" section).

### "Invalid signature"

The token signature doesn't match the public key. This could mean:
1. Token was tampered with
2. Token is from a different Supabase project
3. Keys haven't been refreshed (wait 1 minute and retry)

## Advanced Usage

### Custom Token Validation

Use the standalone validation function:

```go
import "github.com/coder/agentapi/lib/auth"

func validateToken(tokenString string) (*auth.Claims, error) {
    claims, err := auth.ValidateSupabaseJWT(tokenString)
    if err != nil {
        return nil, err
    }

    // Additional custom validation
    if claims.Role != auth.RoleAdmin {
        return nil, errors.New("admin required")
    }

    return claims, nil
}
```

### Multiple Authentication Methods

Combine with API key authentication:

```go
func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Try JWT first
        if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
            // JWT authentication
            jwtMiddleware.ServeHTTP(w, r)
            return
        }

        // Fall back to API key
        if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
            // API key authentication
            apiKeyMiddleware.ServeHTTP(w, r)
            return
        }

        http.Error(w, "Unauthorized", http.StatusUnauthorized)
    })
}
```

## Performance Considerations

### Key Caching

The middleware caches public keys to avoid unnecessary JWKS requests:
- Keys are cached in memory
- Automatic refresh every 5 minutes
- Thread-safe with read/write locks

### Token Parsing

JWT parsing is efficient:
- Uses the industry-standard `golang-jwt` library
- Only parses tokens once per request
- Results stored in context for reuse

### Database Queries

Minimize database queries in your handlers:
```go
// Bad: Query for each request
func handler(w http.ResponseWriter, r *http.Request) {
    userID, orgID, _ := auth.GetUserFromContext(r.Context())
    // Query database for user details every time
}

// Good: Cache user details in claims
func handler(w http.ResponseWriter, r *http.Request) {
    claims, _ := auth.GetClaimsFromContext(r.Context())
    // Use cached email, role, etc. from claims
}
```

## License

MIT License - see LICENSE file for details.
