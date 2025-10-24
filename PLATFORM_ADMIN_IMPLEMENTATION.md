# Platform-Wide Admin Management Implementation

**Date**: October 24, 2025
**Purpose**: Handle platform admins who manage the entire AgentAPI system across all organizations

---

## Overview

You have **three options** for managing platform-wide admins:

1. **WorkOS-Native Solution** (Recommended) - Use WorkOS Organizations & Roles
2. **Database-Backed Solution** - Store platform admins in PostgreSQL
3. **Hybrid Solution** - WorkOS for org roles + Database for platform admins

---

## Option 1: WorkOS-Native Solution (RECOMMENDED)

### Concept
Create a special "Platform Admin" organization in WorkOS that holds all platform administrators. Users who belong to this org get elevated permissions across the entire platform.

### Architecture

```
WorkOS Organizations:
├── Platform Admin Org (special)
│   ├── User1 → role: "admin" → Has all platform permissions
│   ├── User2 → role: "admin" → Has all platform permissions
│   └── ...
├── Company A Org
│   ├── User3 → role: "admin" → Admin of Company A only
│   ├── User4 → role: "member" → Member of Company A
│   └── ...
└── Company B Org
    ├── User5 → role: "admin" → Admin of Company B only
    └── ...
```

### Implementation

#### 1. Create Platform Admin Organization in WorkOS

```bash
# Via WorkOS Dashboard:
# 1. Go to WorkOS Dashboard → Organizations
# 2. Create new org: "Platform Admins"
# 3. Note the org ID (you'll need this)
# 4. Add users to this org with "admin" role
```

#### 2. Update `lib/auth/authkit.go`

Add platform admin detection:

```go
// AuthKitUser represents authenticated user information
type AuthKitUser struct {
	ID          string
	OrgID       string
	Email       string
	Name        string
	Role        string
	Permissions []string
	IsPlatformAdmin bool  // NEW: Flag for platform admins
	Token       string
}

// Add to NewAuthKitValidator or pass as config
type AuthKitValidatorConfig struct {
	Logger              *slog.Logger
	JWKSURL             string
	PlatformAdminOrgID  string  // NEW: Your special "Platform Admins" org ID
}

// Update ValidateToken to check for platform admin
func (av *AuthKitValidator) ValidateToken(ctx context.Context, tokenString string) (*AuthKitUser, error) {
	// ... existing validation code ...

	user := &AuthKitUser{
		ID:          claims.Sub,
		OrgID:       claims.Org,
		Email:       claims.Email,
		Name:        claims.Name,
		Role:        claims.Role,
		Permissions: claims.Permissions,
		Token:       tokenString,
	}

	// NEW: Check if user is in platform admin org
	user.IsPlatformAdmin = claims.Org == av.platformAdminOrgID && claims.Role == "admin"

	return user, nil
}

// Add helper method
func (user *AuthKitUser) IsPlatformAdmin() bool {
	return user.IsPlatformAdmin
}

// Update IsAdmin to consider platform admins
func (user *AuthKitUser) IsAdmin() bool {
	return user.Role == "admin" || user.IsPlatformAdmin
}
```

#### 3. Update `lib/middleware/authkit.go`

Add platform admin access level:

```go
// AccessLevel defines access control levels
type AccessLevel string

const (
	Public            AccessLevel = "public"            // No authentication required
	Authenticated     AccessLevel = "authenticated"     // AuthKit token required
	OrgAdmin          AccessLevel = "org_admin"         // Organization admin required
	PlatformAdmin     AccessLevel = "platform_admin"    // Platform admin required
)

// Update getAccessLevel to check for platform admins
func (tam *TieredAccessMiddleware) checkAccess(user *auth.AuthKitUser, accessLevel AccessLevel) bool {
	switch accessLevel {
	case Public:
		return true
	case Authenticated:
		return user != nil
	case OrgAdmin:
		return user != nil && (user.IsAdmin() || user.IsPlatformAdmin())
	case PlatformAdmin:
		return user != nil && user.IsPlatformAdmin()
	default:
		return false
	}
}
```

#### 4. Register Platform Admin Routes in `api/v1/chat.go`

```go
// Platform admin endpoints - manage all organizations
func ChatRouter(router *http.ServeMux, logger *slog.Logger, chatHandler *chat.ChatHandler, authKitMiddleware *middleware.TieredAccessMiddleware) {
	// ... existing routes ...

	// NEW: Platform admin routes
	router.HandleFunc("GET /api/v1/platform/stats", func(w http.ResponseWriter, r *http.Request) {
		authKitMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			chatHandler.HandlePlatformStats(w, r)
		})).ServeHTTP(w, r)
	})

	router.HandleFunc("GET /api/v1/platform/orgs", func(w http.ResponseWriter, r *http.Request) {
		authKitMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			chatHandler.HandleListOrganizations(w, r)
		})).ServeHTTP(w, r)
	})

	router.HandleFunc("POST /api/v1/platform/admins/add", func(w http.ResponseWriter, r *http.Request) {
		authKitMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			chatHandler.HandleAddPlatformAdmin(w, r)
		})).ServeHTTP(w, r)
	})

	router.HandleFunc("DELETE /api/v1/platform/admins/{email}", func(w http.ResponseWriter, r *http.Request) {
		authKitMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			chatHandler.HandleRemovePlatformAdmin(w, r)
		})).ServeHTTP(w, r)
	})
}

// Set access level for these routes
func setupAccessLevels(tam *middleware.TieredAccessMiddleware) {
	tam.RegisterRoute("/api/v1/platform/stats", middleware.PlatformAdmin)
	tam.RegisterRoute("/api/v1/platform/orgs", middleware.PlatformAdmin)
	tam.RegisterRoute("/api/v1/platform/admins/add", middleware.PlatformAdmin)
	tam.RegisterRoute("/api/v1/platform/admins", middleware.PlatformAdmin)
}
```

#### 5. Example Handlers in `lib/chat/handler.go`

```go
// HandlePlatformStats returns platform-wide statistics
func (ch *ChatHandler) HandlePlatformStats(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("authkit_user").(*auth.AuthKitUser)

	if !user.IsPlatformAdmin() {
		http.Error(w, "forbidden: platform admin required", http.StatusForbidden)
		return
	}

	// Query database for platform stats
	stats := map[string]interface{}{
		"total_organizations": 0,
		"total_users": 0,
		"total_api_calls": 0,
		"total_tokens_used": 0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// HandleAddPlatformAdmin adds a user as platform admin via WorkOS API
func (ch *ChatHandler) HandleAddPlatformAdmin(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("authkit_user").(*auth.AuthKitUser)

	if !user.IsPlatformAdmin() {
		http.Error(w, "forbidden: platform admin required", http.StatusForbidden)
		return
	}

	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Call WorkOS API to add user to platform admin org
	// This requires WorkOS API credentials (not just JWKS)
	// See WorkOS API docs for organization membership management

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "user_added_as_platform_admin",
		"email": req.Email,
	})
}
```

### Pros & Cons

**Pros:**
- ✅ Leverages existing WorkOS infrastructure
- ✅ No additional database tables needed
- ✅ Scales cleanly (organization membership is built-in)
- ✅ Works with existing atoms.tech setup
- ✅ Clean separation of platform vs org admins

**Cons:**
- ❌ Requires WorkOS API access (not just JWKS)
- ❌ Need to manage platform admin org manually
- ❌ Can't use this method if limited to JWKS only

---

## Option 2: Database-Backed Solution

### Concept
Store platform admins in PostgreSQL with a simple table mapping.

### Implementation

#### 1. Create Database Tables

```sql
-- Table for platform-wide admins
CREATE TABLE platform_admins (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	workos_user_id TEXT NOT NULL UNIQUE,
	email TEXT NOT NULL UNIQUE,
	name TEXT,
	added_at TIMESTAMPTZ DEFAULT NOW(),
	added_by UUID REFERENCES platform_admins(id),
	is_active BOOLEAN DEFAULT true,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Table for audit logging of admin actions
CREATE TABLE admin_audit_log (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	admin_id UUID NOT NULL REFERENCES platform_admins(id),
	action TEXT NOT NULL, -- 'added_admin', 'removed_admin', 'accessed_stats', etc.
	target_org_id TEXT,
	target_user_id TEXT,
	details JSONB,
	ip_address TEXT,
	created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Add RLS policies for security
ALTER TABLE platform_admins ENABLE ROW LEVEL SECURITY;

CREATE POLICY platform_admins_select ON platform_admins
	FOR SELECT
	USING (auth.uid()::text IN (SELECT workos_user_id FROM platform_admins WHERE is_active = true));

CREATE POLICY platform_admins_insert ON platform_admins
	FOR INSERT
	WITH CHECK (auth.uid()::text IN (SELECT workos_user_id FROM platform_admins WHERE is_active = true));
```

#### 2. Update `lib/auth/authkit.go`

```go
// AuthKitUser now includes platform admin flag
type AuthKitUser struct {
	ID          string
	OrgID       string
	Email       string
	Name        string
	Role        string
	Permissions []string
	IsPlatformAdmin bool  // From database check
	Token       string
}

// Pass database connection to validator
type AuthKitValidator struct {
	logger        *slog.Logger
	jwksURL       string
	publicKeys    map[string]*rsa.PublicKey
	mu            sync.RWMutex
	keyExpiry     time.Time
	keyRefreshTTL time.Duration
	db            *sql.DB  // NEW: Database connection
}

func (av *AuthKitValidator) ValidateToken(ctx context.Context, tokenString string) (*AuthKitUser, error) {
	// ... existing JWT validation ...

	// NEW: Check if user is platform admin
	var isPlatformAdmin bool
	err := av.db.QueryRowContext(
		ctx,
		"SELECT is_active FROM platform_admins WHERE workos_user_id = $1 AND is_active = true",
		claims.Sub,
	).Scan(&isPlatformAdmin)

	if err == sql.ErrNoRows {
		isPlatformAdmin = false
	} else if err != nil {
		av.logger.Error("failed to check platform admin status", "error", err)
		isPlatformAdmin = false
	}

	return &AuthKitUser{
		ID:              claims.Sub,
		OrgID:           claims.Org,
		Email:           claims.Email,
		Name:            claims.Name,
		Role:            claims.Role,
		Permissions:     claims.Permissions,
		IsPlatformAdmin: isPlatformAdmin,
		Token:           tokenString,
	}, nil
}

// Helper method
func (user *AuthKitUser) IsPlatformAdmin() bool {
	return user.IsPlatformAdmin
}
```

#### 3. Create Admin Management Service

```go
// File: lib/admin/platform.go

package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

type PlatformAdminService struct {
	db     *sql.DB
	logger *slog.Logger
}

type PlatformAdmin struct {
	ID          string    `json:"id"`
	WorkOSID    string    `json:"workos_id"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	IsActive    bool      `json:"is_active"`
	AddedAt     time.Time `json:"added_at"`
	AddedBy     *string   `json:"added_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewPlatformAdminService(db *sql.DB, logger *slog.Logger) *PlatformAdminService {
	return &PlatformAdminService{
		db:     db,
		logger: logger,
	}
}

// AddAdmin adds a user as platform admin
func (pas *PlatformAdminService) AddAdmin(ctx context.Context, workosID, email, name, addedByID string) error {
	query := `
		INSERT INTO platform_admins (workos_user_id, email, name, added_by, is_active)
		VALUES ($1, $2, $3, $4, true)
		ON CONFLICT (email) DO UPDATE SET is_active = true, updated_at = NOW()
		RETURNING id
	`

	var id string
	err := pas.db.QueryRowContext(ctx, query, workosID, email, name, addedByID).Scan(&id)
	if err != nil {
		pas.logger.Error("failed to add platform admin", "error", err, "email", email)
		return fmt.Errorf("failed to add admin: %w", err)
	}

	// Log audit event
	_ = pas.logAuditEvent(ctx, addedByID, "added_admin", "", workosID, map[string]interface{}{
		"email": email,
		"name":  name,
	})

	pas.logger.Info("platform admin added", "email", email, "admin_id", id)
	return nil
}

// RemoveAdmin revokes platform admin access
func (pas *PlatformAdminService) RemoveAdmin(ctx context.Context, email, removedByID string) error {
	query := `UPDATE platform_admins SET is_active = false, updated_at = NOW() WHERE email = $1 RETURNING workos_user_id`

	var workosID string
	err := pas.db.QueryRowContext(ctx, query, email).Scan(&workosID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("admin not found: %s", email)
	} else if err != nil {
		return fmt.Errorf("failed to remove admin: %w", err)
	}

	// Log audit event
	_ = pas.logAuditEvent(ctx, removedByID, "removed_admin", "", workosID, map[string]interface{}{
		"email": email,
	})

	pas.logger.Info("platform admin removed", "email", email)
	return nil
}

// ListAdmins returns all active platform admins
func (pas *PlatformAdminService) ListAdmins(ctx context.Context) ([]PlatformAdmin, error) {
	query := `
		SELECT id, workos_user_id, email, name, is_active, added_at, added_by, created_at, updated_at
		FROM platform_admins
		WHERE is_active = true
		ORDER BY added_at DESC
	`

	rows, err := pas.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list admins: %w", err)
	}
	defer rows.Close()

	var admins []PlatformAdmin
	for rows.Next() {
		var admin PlatformAdmin
		if err := rows.Scan(
			&admin.ID, &admin.WorkOSID, &admin.Email, &admin.Name, &admin.IsActive,
			&admin.AddedAt, &admin.AddedBy, &admin.CreatedAt, &admin.UpdatedAt,
		); err != nil {
			return nil, err
		}
		admins = append(admins, admin)
	}

	return admins, rows.Err()
}

// logAuditEvent logs an admin action
func (pas *PlatformAdminService) logAuditEvent(ctx context.Context, adminID, action, targetOrgID, targetUserID string, details map[string]interface{}) error {
	detailsJSON, _ := json.Marshal(details)

	query := `
		INSERT INTO admin_audit_log (admin_id, action, target_org_id, target_user_id, details)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := pas.db.ExecContext(ctx, query, adminID, action, targetOrgID, targetUserID, detailsJSON)
	return err
}
```

#### 4. Update handlers to use admin service

```go
// In lib/chat/handler.go

func (ch *ChatHandler) HandleAddPlatformAdmin(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("authkit_user").(*auth.AuthKitUser)

	if !user.IsPlatformAdmin() {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var req struct {
		Email string `json:"email"`
		Name  string `json:"name"`
		WorkOSID string `json:"workos_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Use admin service
	err := ch.adminService.AddAdmin(r.Context(), req.WorkOSID, req.Email, req.Name, user.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to add admin: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"email": req.Email,
	})
}
```

### Pros & Cons

**Pros:**
- ✅ Full control - no dependency on WorkOS org management
- ✅ Audit logging built-in
- ✅ Can implement complex rules (e.g., time-limited admins)
- ✅ Doesn't require WorkOS API credentials
- ✅ Works with JWKS-only setup

**Cons:**
- ❌ Requires database schema management
- ❌ Manual sync if users change in WorkOS
- ❌ Need to maintain additional table

---

## Option 3: Hybrid Solution (BEST FOR YOUR SETUP)

### Concept
Use **WorkOS org roles** for org-level admins (already in JWT) + **Database** for platform admins.

This gives you:
- Organization admins from JWT claims (workOS)
- Platform admins from database (PostgreSQL)
- Single source of truth for each level

### Implementation

```go
// File: lib/auth/authkit.go

type AuthKitUser struct {
	ID          string       // User ID
	OrgID       string       // Organization ID
	Email       string
	Name        string
	Role        string       // org, member, viewer (from WorkOS)
	Permissions []string

	// Authorization levels
	IsOrgAdmin  bool         // true if Role == "admin" (from WorkOS)
	IsPlatformAdmin bool      // true if in platform_admins table (from DB)
	Token       string
}

// Helper methods
func (user *AuthKitUser) IsOrgAdmin() bool {
	return user.Role == "admin" // Organization-level admin
}

func (user *AuthKitUser) IsPlatformAdmin() bool {
	return user.IsPlatformAdmin // Platform-level admin
}

func (user *AuthKitUser) IsAnyAdmin() bool {
	return user.IsOrgAdmin() || user.IsPlatformAdmin
}

// For granular control
func (user *AuthKitUser) CanManageOrg(orgID string) bool {
	if user.IsPlatformAdmin {
		return true // Platform admins can manage any org
	}
	return user.Role == "admin" && user.OrgID == orgID // Org admins manage only their org
}
```

### Tiered Access Control

```go
// File: lib/middleware/authkit.go

type AccessLevel string

const (
	Public         AccessLevel = "public"         // No auth
	Authenticated  AccessLevel = "authenticated"  // Any logged-in user
	OrgAdmin       AccessLevel = "org_admin"      // Admin of their org
	PlatformAdmin  AccessLevel = "platform_admin" // Platform-wide admin
)

func (tam *TieredAccessMiddleware) checkAccess(user *auth.AuthKitUser, accessLevel AccessLevel) bool {
	if user == nil {
		return accessLevel == Public
	}

	switch accessLevel {
	case Public:
		return true
	case Authenticated:
		return true
	case OrgAdmin:
		return user.IsOrgAdmin() || user.IsPlatformAdmin
	case PlatformAdmin:
		return user.IsPlatformAdmin
	default:
		return false
	}
}
```

### Route Registration

```go
// File: api/v1/chat.go

func setupAccessLevels(tam *middleware.TieredAccessMiddleware) {
	// Public
	tam.RegisterRoute("/health", middleware.Public)
	tam.RegisterRoute("/ready", middleware.Public)

	// Authenticated (any user)
	tam.RegisterRoute("/v1/chat/completions", middleware.Authenticated)
	tam.RegisterRoute("/v1/models", middleware.Authenticated)

	// Org admins (manage their org)
	tam.RegisterRoute("/api/v1/org/settings", middleware.OrgAdmin)
	tam.RegisterRoute("/api/v1/org/members", middleware.OrgAdmin)
	tam.RegisterRoute("/api/v1/org/billing", middleware.OrgAdmin)

	// Platform admins (manage everything)
	tam.RegisterRoute("/api/v1/platform/admins", middleware.PlatformAdmin)
	tam.RegisterRoute("/api/v1/platform/stats", middleware.PlatformAdmin)
	tam.RegisterRoute("/api/v1/platform/orgs", middleware.PlatformAdmin)
	tam.RegisterRoute("/api/v1/platform/audit", middleware.PlatformAdmin)
}
```

---

## Recommended Approach for Your Stack

**Use Option 3 (Hybrid)** because:

1. **You already have WorkOS setup** in atoms.tech
2. **JWT already includes org role** from WorkOS
3. **PostgreSQL is available** for audit logging
4. **Clean separation of concerns**: WorkOS handles org auth, DB handles platform auth
5. **Scalable**: Add more permission types (e.g., "super_admin", "support_admin") to database later

---

## Quick Start: Add Platform Admin Management

### 1. Create Database Table

```sql
CREATE TABLE platform_admins (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	workos_user_id TEXT NOT NULL UNIQUE,
	email TEXT NOT NULL UNIQUE,
	is_active BOOLEAN DEFAULT true,
	added_at TIMESTAMPTZ DEFAULT NOW()
);
```

### 2. Update `lib/auth/authkit.go`

```go
func (av *AuthKitValidator) ValidateToken(ctx context.Context, tokenString string) (*AuthKitUser, error) {
	// ... existing validation ...

	var isPlatformAdmin bool
	_ = av.db.QueryRowContext(
		ctx,
		"SELECT true FROM platform_admins WHERE workos_user_id = $1 AND is_active = true LIMIT 1",
		claims.Sub,
	).Scan(&isPlatformAdmin)

	return &AuthKitUser{
		ID:              claims.Sub,
		OrgID:           claims.Org,
		Email:           claims.Email,
		Role:            claims.Role,
		Permissions:     claims.Permissions,
		IsOrgAdmin:      claims.Role == "admin",
		IsPlatformAdmin: isPlatformAdmin,
	}, nil
}
```

### 3. Add Platform Admin Endpoints

```go
router.HandleFunc("POST /api/v1/platform/admins", func(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("authkit_user").(*auth.AuthKitUser)
	if !user.IsPlatformAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// Add to platform_admins table
	_, err := db.ExecContext(r.Context(),
		"INSERT INTO platform_admins (workos_user_id, email) VALUES ($1, $2)",
		req.Email, req.Email, // Use email as workos_user_id for now
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "added"})
})
```

### 4. Test It

```bash
# Add someone as platform admin
curl -X POST http://localhost:3284/api/v1/platform/admins \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@company.com"}'

# Now that user has IsPlatformAdmin = true
# and can access all /api/v1/platform/* endpoints
```

---

## Security Considerations

1. **Audit Logging**: Log all platform admin actions
2. **Rate Limiting**: Stricter limits on admin endpoints
3. **Token Rotation**: Platform admin tokens expire faster
4. **IP Whitelisting**: Optional - restrict platform admin access to known IPs
5. **Multi-factor Auth**: Require 2FA for platform admins via WorkOS
6. **Approval Workflow**: Require 2+ admins to approve new platform admins

---

## Summary Table

| Feature | Option 1 (WorkOS) | Option 2 (DB) | Option 3 (Hybrid) |
|---------|-------------------|---------------|-------------------|
| Setup Complexity | Medium | Low | Low |
| Dependencies | WorkOS API | PostgreSQL | Both |
| Audit Logging | Manual | Built-in | Built-in |
| Org Admins | ✅ WorkOS | ❌ Need to sync | ✅ WorkOS |
| Platform Admins | ✅ WorkOS Org | ✅ Database | ✅ Database |
| Scalability | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |
| **Recommended** | - | - | ✅ YES |

Choose **Option 3 (Hybrid)** for best balance of features, maintainability, and scalability.
