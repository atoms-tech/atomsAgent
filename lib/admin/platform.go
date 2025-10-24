package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// PlatformAdminService manages platform-wide administrators
type PlatformAdminService struct {
	db     *sql.DB
	logger *slog.Logger
}

// PlatformAdmin represents a platform administrator
type PlatformAdmin struct {
	ID        string    `json:"id"`
	WorkOSID  string    `json:"workos_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	AddedAt   time.Time `json:"added_at"`
	AddedBy   *string   `json:"added_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewPlatformAdminService creates a new platform admin service
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
		ON CONFLICT (email) DO UPDATE SET 
			is_active = true, 
			updated_at = NOW(),
			workos_user_id = EXCLUDED.workos_user_id,
			name = EXCLUDED.name
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

// GetAdminByEmail returns a platform admin by email
func (pas *PlatformAdminService) GetAdminByEmail(ctx context.Context, email string) (*PlatformAdmin, error) {
	query := `
		SELECT id, workos_user_id, email, name, is_active, added_at, added_by, created_at, updated_at
		FROM platform_admins
		WHERE email = $1 AND is_active = true
	`

	var admin PlatformAdmin
	err := pas.db.QueryRowContext(ctx, query, email).Scan(
		&admin.ID, &admin.WorkOSID, &admin.Email, &admin.Name, &admin.IsActive,
		&admin.AddedAt, &admin.AddedBy, &admin.CreatedAt, &admin.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get admin: %w", err)
	}

	return &admin, nil
}

// IsPlatformAdmin checks if a user is a platform admin
func (pas *PlatformAdminService) IsPlatformAdmin(ctx context.Context, workosUserID string) (bool, error) {
	query := `SELECT true FROM platform_admins WHERE workos_user_id = $1 AND is_active = true LIMIT 1`

	var exists bool
	err := pas.db.QueryRowContext(ctx, query, workosUserID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to check platform admin status: %w", err)
	}

	return exists, nil
}

// GetPlatformStats returns platform-wide statistics
func (pas *PlatformAdminService) GetPlatformStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Count total platform admins
	var adminCount int
	err := pas.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM platform_admins WHERE is_active = true").Scan(&adminCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count admins: %w", err)
	}
	stats["total_platform_admins"] = adminCount

	// Count total audit log entries
	var auditCount int
	err = pas.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM admin_audit_log").Scan(&auditCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count audit entries: %w", err)
	}
	stats["total_audit_entries"] = auditCount

	// Get recent admin actions (last 24 hours)
	var recentActions int
	err = pas.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM admin_audit_log WHERE created_at > NOW() - INTERVAL '24 hours'").Scan(&recentActions)
	if err != nil {
		return nil, fmt.Errorf("failed to count recent actions: %w", err)
	}
	stats["recent_admin_actions"] = recentActions

	return stats, nil
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

// GetAuditLog returns audit log entries with pagination
func (pas *PlatformAdminService) GetAuditLog(ctx context.Context, limit, offset int) ([]map[string]interface{}, error) {
	query := `
		SELECT al.id, al.action, al.target_org_id, al.target_user_id, al.details, al.ip_address, al.created_at,
		       pa.email as admin_email, pa.name as admin_name
		FROM admin_audit_log al
		LEFT JOIN platform_admins pa ON al.admin_id = pa.id
		ORDER BY al.created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := pas.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log: %w", err)
	}
	defer rows.Close()

	var entries []map[string]interface{}
	for rows.Next() {
		var id, action, targetOrgID, targetUserID, ipAddress, adminEmail, adminName string
		var detailsJSON []byte
		var createdAt time.Time

		err := rows.Scan(&id, &action, &targetOrgID, &targetUserID, &detailsJSON, &ipAddress, &createdAt, &adminEmail, &adminName)
		if err != nil {
			return nil, err
		}

		var details map[string]interface{}
		if len(detailsJSON) > 0 {
			json.Unmarshal(detailsJSON, &details)
		}

		entry := map[string]interface{}{
			"id":             id,
			"action":         action,
			"target_org_id":  targetOrgID,
			"target_user_id": targetUserID,
			"details":        details,
			"ip_address":     ipAddress,
			"created_at":     createdAt,
			"admin_email":    adminEmail,
			"admin_name":     adminName,
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}
