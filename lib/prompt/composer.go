package prompt

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"html"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"
)

// PromptComposer manages system prompt composition with database persistence
type PromptComposer struct {
	db        *sql.DB
	sanitizer *PromptSanitizer
	cache     *PromptCache
	mu        sync.RWMutex
}

// NewPromptComposer creates a new prompt composer with database support
func NewPromptComposer(db *sql.DB) *PromptComposer {
	return &PromptComposer{
		db:        db,
		sanitizer: NewPromptSanitizer(),
		cache:     NewPromptCache(5 * time.Minute),
	}
}

// SystemPrompt represents a system prompt in the database
type SystemPrompt struct {
	ID        string
	Scope     PromptScope
	Content   string
	Template  string
	OrgID     *string
	UserID    *string
	Priority  int
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PromptScope defines the scope of a prompt
type PromptScope string

const (
	ScopeGlobal PromptScope = "global"
	ScopeOrg    PromptScope = "org"
	ScopeUser   PromptScope = "user"
)

// ValidateScope checks if a scope is valid
func (s PromptScope) IsValid() error {
	switch s {
	case ScopeGlobal, ScopeOrg, ScopeUser:
		return nil
	default:
		return fmt.Errorf("invalid scope: %s (must be global, org, or user)", s)
	}
}

// TemplateContext provides context for template rendering
type TemplateContext struct {
	UserID      string
	OrgID       string
	Date        string
	Time        string
	Environment map[string]string
}

// NewTemplateContext creates a new template context
func NewTemplateContext(userID, orgID string) *TemplateContext {
	now := time.Now()
	return &TemplateContext{
		UserID:      userID,
		OrgID:       orgID,
		Date:        now.Format("2006-01-02"),
		Time:        now.Format("15:04:05"),
		Environment: make(map[string]string),
	}
}

// AddEnvironmentVar adds an environment variable to the context
func (tc *TemplateContext) AddEnvironmentVar(key, value string) {
	tc.Environment[key] = value
}

// LoadEnvironment loads specified environment variables
func (tc *TemplateContext) LoadEnvironment(vars ...string) {
	for _, v := range vars {
		if val := os.Getenv(v); val != "" {
			tc.Environment[v] = val
		}
	}
}

// ComposePrompt fetches and composes all applicable prompts for a user
func (pc *PromptComposer) ComposePrompt(ctx context.Context, userID, orgID string) (string, error) {
	// Check cache first
	if cached, found := pc.cache.Get(userID, orgID); found {
		return cached, nil
	}

	// Fetch all applicable prompts
	prompts, err := pc.FetchPrompts(ctx, userID, orgID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch prompts: %w", err)
	}

	// Sort by priority (higher first)
	sort.Slice(prompts, func(i, j int) bool {
		return prompts[i].Priority > prompts[j].Priority
	})

	// Create template context
	tmplCtx := NewTemplateContext(userID, orgID)

	// Optionally load common environment variables
	tmplCtx.LoadEnvironment("ENVIRONMENT", "APP_VERSION", "REGION")

	// Process each prompt
	var parts []string
	for _, prompt := range prompts {
		if !prompt.Enabled {
			continue
		}

		content, err := pc.processPrompt(prompt, tmplCtx)
		if err != nil {
			return "", fmt.Errorf("failed to process prompt %s: %w", prompt.ID, err)
		}

		if strings.TrimSpace(content) != "" {
			parts = append(parts, content)
		}
	}

	// Compose final prompt
	composed := strings.Join(parts, "\n\n")

	// Sanitize for prompt injection
	sanitized, err := pc.sanitizer.Sanitize(composed)
	if err != nil {
		return "", fmt.Errorf("sanitization failed: %w", err)
	}

	// Cache the result
	pc.cache.Set(userID, orgID, sanitized)

	return sanitized, nil
}

// processPrompt processes a single prompt with template rendering
func (pc *PromptComposer) processPrompt(prompt *SystemPrompt, ctx *TemplateContext) (string, error) {
	content := prompt.Content

	// If template is provided, render it
	if prompt.Template != "" {
		tmpl, err := template.New(prompt.ID).Parse(prompt.Template)
		if err != nil {
			return "", fmt.Errorf("template parse error: %w", err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, ctx); err != nil {
			return "", fmt.Errorf("template execution error: %w", err)
		}
		content = buf.String()
	}

	return content, nil
}

// FetchPrompts retrieves all applicable prompts for a user from the database
func (pc *PromptComposer) FetchPrompts(ctx context.Context, userID, orgID string) ([]*SystemPrompt, error) {
	query := `
		SELECT id, scope, content, template, org_id, user_id, priority, enabled, created_at, updated_at
		FROM system_prompts
		WHERE enabled = true
		AND (
			scope = 'global'
			OR (scope = 'org' AND org_id = $1)
			OR (scope = 'user' AND user_id = $2)
		)
		ORDER BY priority DESC
	`

	rows, err := pc.db.QueryContext(ctx, query, orgID, userID)
	if err != nil {
		return nil, fmt.Errorf("database query failed: %w", err)
	}
	defer rows.Close()

	var prompts []*SystemPrompt
	for rows.Next() {
		var p SystemPrompt
		var template sql.NullString
		err := rows.Scan(
			&p.ID, &p.Scope, &p.Content, &template,
			&p.OrgID, &p.UserID, &p.Priority, &p.Enabled,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("row scan failed: %w", err)
		}

		if template.Valid {
			p.Template = template.String
		}

		prompts = append(prompts, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return prompts, nil
}

// CreatePrompt creates a new system prompt
func (pc *PromptComposer) CreatePrompt(ctx context.Context, prompt *SystemPrompt) error {
	// Validate scope
	if err := prompt.Scope.IsValid(); err != nil {
		return err
	}

	// Validate scope-specific requirements
	if err := pc.validatePromptScope(prompt); err != nil {
		return err
	}

	// Validate content
	if err := pc.validateContent(prompt); err != nil {
		return err
	}

	query := `
		INSERT INTO system_prompts (id, scope, content, template, org_id, user_id, priority, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	now := time.Now()
	_, err := pc.db.ExecContext(ctx, query,
		prompt.ID, prompt.Scope, prompt.Content, toNullString(prompt.Template),
		prompt.OrgID, prompt.UserID, prompt.Priority, prompt.Enabled,
		now, now,
	)

	if err != nil {
		return fmt.Errorf("failed to create prompt: %w", err)
	}

	// Invalidate cache
	pc.cache.Clear()

	return nil
}

// UpdatePrompt updates an existing system prompt
func (pc *PromptComposer) UpdatePrompt(ctx context.Context, id string, prompt *SystemPrompt) error {
	// Validate scope
	if err := prompt.Scope.IsValid(); err != nil {
		return err
	}

	// Validate scope-specific requirements
	if err := pc.validatePromptScope(prompt); err != nil {
		return err
	}

	// Validate content
	if err := pc.validateContent(prompt); err != nil {
		return err
	}

	query := `
		UPDATE system_prompts
		SET scope = $1, content = $2, template = $3, org_id = $4, user_id = $5,
		    priority = $6, enabled = $7, updated_at = $8
		WHERE id = $9
	`

	result, err := pc.db.ExecContext(ctx, query,
		prompt.Scope, prompt.Content, toNullString(prompt.Template),
		prompt.OrgID, prompt.UserID, prompt.Priority, prompt.Enabled,
		time.Now(), id,
	)

	if err != nil {
		return fmt.Errorf("failed to update prompt: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("prompt not found: %s", id)
	}

	// Invalidate cache
	pc.cache.Clear()

	return nil
}

// DeletePrompt deletes a system prompt
func (pc *PromptComposer) DeletePrompt(ctx context.Context, id string) error {
	query := `DELETE FROM system_prompts WHERE id = $1`

	result, err := pc.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete prompt: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("prompt not found: %s", id)
	}

	// Invalidate cache
	pc.cache.Clear()

	return nil
}

// GetPrompt retrieves a single prompt by ID
func (pc *PromptComposer) GetPrompt(ctx context.Context, id string) (*SystemPrompt, error) {
	query := `
		SELECT id, scope, content, template, org_id, user_id, priority, enabled, created_at, updated_at
		FROM system_prompts
		WHERE id = $1
	`

	var p SystemPrompt
	var template sql.NullString

	err := pc.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.Scope, &p.Content, &template,
		&p.OrgID, &p.UserID, &p.Priority, &p.Enabled,
		&p.CreatedAt, &p.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("prompt not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt: %w", err)
	}

	if template.Valid {
		p.Template = template.String
	}

	return &p, nil
}

// validatePromptScope validates scope-specific requirements
func (pc *PromptComposer) validatePromptScope(prompt *SystemPrompt) error {
	switch prompt.Scope {
	case ScopeGlobal:
		if prompt.OrgID != nil || prompt.UserID != nil {
			return fmt.Errorf("global prompts cannot have orgID or userID")
		}
	case ScopeOrg:
		if prompt.OrgID == nil {
			return fmt.Errorf("org prompts must have orgID")
		}
		if prompt.UserID != nil {
			return fmt.Errorf("org prompts cannot have userID")
		}
	case ScopeUser:
		if prompt.UserID == nil {
			return fmt.Errorf("user prompts must have userID")
		}
		if prompt.OrgID == nil {
			return fmt.Errorf("user prompts must have orgID")
		}
	}
	return nil
}

// validateContent validates prompt content and template
func (pc *PromptComposer) validateContent(prompt *SystemPrompt) error {
	if prompt.Content == "" && prompt.Template == "" {
		return fmt.Errorf("prompt must have either content or template")
	}

	// Validate template syntax if provided
	if prompt.Template != "" {
		_, err := template.New("validation").Parse(prompt.Template)
		if err != nil {
			return fmt.Errorf("invalid template syntax: %w", err)
		}
	}

	return nil
}

// PromptSanitizer handles sanitization of composed prompts
type PromptSanitizer struct {
	dangerousPatterns []*regexp.Regexp
}

// NewPromptSanitizer creates a new sanitizer
func NewPromptSanitizer() *PromptSanitizer {
	return &PromptSanitizer{
		dangerousPatterns: []*regexp.Regexp{
			// Prompt injection patterns
			regexp.MustCompile(`(?i)(ignore|forget|disregard)\s+(all\s+)?(previous|prior|above)\s+(instructions?|prompts?|commands?)`),
			regexp.MustCompile(`(?i)(you\s+are\s+now|act\s+as|pretend\s+to\s+be)\s+`),
			regexp.MustCompile(`(?i)(system|admin|root)\s+(prompt|instruction|mode)`),
			regexp.MustCompile(`(?i)jailbreak|bypass\s+restrictions?|escape\s+sandbox`),

			// Code injection patterns
			regexp.MustCompile(`(?i)(execute|run|eval)\s+(code|script|command)`),
			regexp.MustCompile(`(?i)<script[^>]*>|javascript:|vbscript:|data:text/html`),

			// Data exfiltration patterns
			regexp.MustCompile(`(?i)(send|transmit|upload|exfiltrate)\s+(data|information|secrets?)`),
			regexp.MustCompile(`(?i)(api[_-]?key|token|password|secret|credential)\s*[:=]\s*["\']?\w+`),

			// Role manipulation
			regexp.MustCompile(`(?i)new\s+(system\s+)?role|override\s+instructions?`),
			regexp.MustCompile(`(?i)end\s+of\s+(instructions?|prompt|system)`),
		},
	}
}

// Sanitize cleans and validates a prompt for injection attacks
func (ps *PromptSanitizer) Sanitize(content string) (string, error) {
	// Check for dangerous patterns
	for _, pattern := range ps.dangerousPatterns {
		if matches := pattern.FindAllString(content, -1); len(matches) > 0 {
			// Replace dangerous patterns with [REDACTED]
			content = pattern.ReplaceAllString(content, "[REDACTED]")
		}
	}

	// HTML escape dangerous characters
	content = html.EscapeString(content)

	// Remove control characters except newlines, tabs, and carriage returns
	content = removeControlChars(content)

	// Normalize excessive whitespace
	content = normalizeWhitespace(content)

	return content, nil
}

// Validate checks if content contains dangerous patterns
func (ps *PromptSanitizer) Validate(content string) error {
	for _, pattern := range ps.dangerousPatterns {
		if pattern.MatchString(content) {
			return fmt.Errorf("dangerous pattern detected: %s", pattern.String())
		}
	}
	return nil
}

// PromptCache provides caching for composed prompts
type PromptCache struct {
	cache map[string]*cacheEntry
	ttl   time.Duration
	mu    sync.RWMutex
}

type cacheEntry struct {
	value     string
	expiresAt time.Time
}

// NewPromptCache creates a new prompt cache
func NewPromptCache(ttl time.Duration) *PromptCache {
	cache := &PromptCache{
		cache: make(map[string]*cacheEntry),
		ttl:   ttl,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a cached prompt
func (pc *PromptCache) Get(userID, orgID string) (string, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	key := cacheKey(userID, orgID)
	entry, found := pc.cache[key]
	if !found {
		return "", false
	}

	if time.Now().After(entry.expiresAt) {
		return "", false
	}

	return entry.value, true
}

// Set stores a prompt in the cache
func (pc *PromptCache) Set(userID, orgID, value string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	key := cacheKey(userID, orgID)
	pc.cache[key] = &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(pc.ttl),
	}
}

// Clear clears all cached prompts
func (pc *PromptCache) Clear() {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.cache = make(map[string]*cacheEntry)
}

// cleanup removes expired entries periodically
func (pc *PromptCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		pc.mu.Lock()
		now := time.Now()
		for key, entry := range pc.cache {
			if now.After(entry.expiresAt) {
				delete(pc.cache, key)
			}
		}
		pc.mu.Unlock()
	}
}

// Helper functions

func cacheKey(userID, orgID string) string {
	return fmt.Sprintf("%s:%s", orgID, userID)
}

func toNullString(s string) sql.NullString {
	return sql.NullString{
		String: s,
		Valid:  s != "",
	}
}

func removeControlChars(s string) string {
	var result strings.Builder
	for _, r := range s {
		// Allow newline, carriage return, tab
		if r == '\n' || r == '\r' || r == '\t' {
			result.WriteRune(r)
			continue
		}
		// Skip other control characters
		if r < 32 || r == 127 {
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

func normalizeWhitespace(s string) string {
	// Replace multiple spaces with single space
	s = regexp.MustCompile(` +`).ReplaceAllString(s, " ")
	// Replace multiple newlines with double newline
	s = regexp.MustCompile(`\n{3,}`).ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

// InitializeSchema creates the database schema for system prompts
func InitializeSchema(ctx context.Context, db *sql.DB) error {
	schema := `
		CREATE TABLE IF NOT EXISTS system_prompts (
			id VARCHAR(255) PRIMARY KEY,
			scope VARCHAR(50) NOT NULL CHECK (scope IN ('global', 'org', 'user')),
			content TEXT NOT NULL,
			template TEXT,
			org_id VARCHAR(255),
			user_id VARCHAR(255),
			priority INTEGER NOT NULL DEFAULT 0,
			enabled BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

			-- Constraints
			CONSTRAINT valid_global_scope CHECK (
				scope != 'global' OR (org_id IS NULL AND user_id IS NULL)
			),
			CONSTRAINT valid_org_scope CHECK (
				scope != 'org' OR (org_id IS NOT NULL AND user_id IS NULL)
			),
			CONSTRAINT valid_user_scope CHECK (
				scope != 'user' OR (org_id IS NOT NULL AND user_id IS NOT NULL)
			)
		);

		CREATE INDEX IF NOT EXISTS idx_system_prompts_scope ON system_prompts(scope);
		CREATE INDEX IF NOT EXISTS idx_system_prompts_org_id ON system_prompts(org_id) WHERE org_id IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_system_prompts_user_id ON system_prompts(user_id) WHERE user_id IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_system_prompts_enabled ON system_prompts(enabled) WHERE enabled = true;
	`

	_, err := db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	return nil
}
