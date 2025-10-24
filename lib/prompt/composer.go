package prompt

import (
	"bytes"
	"fmt"
	"html"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

// Composer handles system prompt composition and security
type Composer struct {
	GlobalPrompts []SystemPromptConfig
	OrgPrompts    []SystemPromptConfig
	UserPrompts   []SystemPromptConfig
	Validator     *Validator
}

// SystemPromptConfig represents a system prompt configuration
type SystemPromptConfig struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Content   string            `json:"content"`
	Template  string            `json:"template,omitempty"`
	Variables map[string]any    `json:"variables"`
	Scope     PromptScope       `json:"scope"`
	Priority  int               `json:"priority"`
	OrgID     string            `json:"orgId,omitempty"`
	UserID    string            `json:"userId,omitempty"`
	IsActive  bool              `json:"isActive"`
}

type PromptScope string

const (
	PromptScopeGlobal PromptScope = "global"
	PromptScopeOrg    PromptScope = "org"
	PromptScopeUser   PromptScope = "user"
)

// ComposeSystemPrompt composes the final system prompt for a user
func (c *Composer) ComposeSystemPrompt(userID, orgID string, userVariables map[string]any) (string, error) {
	// Collect all applicable prompts
	var prompts []SystemPromptConfig
	
	// Add global prompts
	for _, prompt := range c.GlobalPrompts {
		if prompt.IsActive {
			prompts = append(prompts, prompt)
		}
	}
	
	// Add org prompts
	for _, prompt := range c.OrgPrompts {
		if prompt.IsActive && prompt.OrgID == orgID {
			prompts = append(prompts, prompt)
		}
	}
	
	// Add user prompts
	for _, prompt := range c.UserPrompts {
		if prompt.IsActive && prompt.UserID == userID {
			prompts = append(prompts, prompt)
		}
	}
	
	// Sort by priority (higher priority first)
	sort.Slice(prompts, func(i, j int) bool {
		return prompts[i].Priority > prompts[j].Priority
	})
	
	// Compose the final prompt
	var parts []string
	for _, prompt := range prompts {
		content, err := c.processPrompt(prompt, userVariables)
		if err != nil {
			return "", fmt.Errorf("failed to process prompt %s: %w", prompt.Name, err)
		}
		
		if content != "" {
			parts = append(parts, content)
		}
	}
	
	return strings.Join(parts, "\n\n"), nil
}

// processPrompt processes a single prompt with template and variables
func (c *Composer) processPrompt(prompt SystemPromptConfig, userVariables map[string]any) (string, error) {
	content := prompt.Content
	
	// If template is provided, use it
	if prompt.Template != "" {
		tmpl, err := template.New(prompt.Name).Parse(prompt.Template)
		if err != nil {
			return "", fmt.Errorf("failed to parse template: %w", err)
		}
		
		// Merge variables
		variables := make(map[string]any)
		for k, v := range prompt.Variables {
			variables[k] = v
		}
		for k, v := range userVariables {
			variables[k] = v
		}
		
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, variables); err != nil {
			return "", fmt.Errorf("failed to execute template: %w", err)
		}
		content = buf.String()
	}
	
	// Validate and sanitize content
	if c.Validator != nil {
		if err := c.Validator.Validate(content); err != nil {
			return "", fmt.Errorf("prompt validation failed: %w", err)
		}
		content = c.Validator.Sanitize(content)
	}
	
	return content, nil
}

// Validator handles prompt validation and sanitization
type Validator struct {
	// Dangerous patterns to detect
	dangerousPatterns []*regexp.Regexp
	// Allowed HTML tags (if any)
	allowedHTMLTags map[string]bool
}

// NewValidator creates a new prompt validator
func NewValidator() *Validator {
	return &Validator{
		dangerousPatterns: []*regexp.Regexp{
			// Prompt injection patterns
			regexp.MustCompile(`(?i)(ignore|forget|disregard).*(previous|above|instructions|prompt)`),
			regexp.MustCompile(`(?i)(you are now|act as|pretend to be)`),
			regexp.MustCompile(`(?i)(system|admin|root).*(prompt|instruction)`),
			regexp.MustCompile(`(?i)(jailbreak|escape|bypass)`),
			// Code injection patterns
			regexp.MustCompile(`(?i)(execute|run|eval).*(code|script|command)`),
			regexp.MustCompile(`(?i)(<script|javascript:|vbscript:)`),
			// Data exfiltration patterns
			regexp.MustCompile(`(?i)(send|transmit|upload).*(data|information|secrets)`),
			regexp.MustCompile(`(?i)(api[_-]?key|token|password|secret).*[:=]`),
		},
		allowedHTMLTags: map[string]bool{
			"p": true, "br": true, "strong": true, "em": true, "code": true,
		},
	}
}

// Validate checks if a prompt contains dangerous patterns
func (v *Validator) Validate(content string) error {
	for _, pattern := range v.dangerousPatterns {
		if pattern.MatchString(content) {
			return fmt.Errorf("dangerous pattern detected: %s", pattern.String())
		}
	}
	return nil
}

// Sanitize cleans potentially dangerous content from prompts
func (v *Validator) Sanitize(content string) string {
	// HTML escape to prevent XSS
	content = html.EscapeString(content)
	
	// Remove any remaining script tags
	scriptRegex := regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	content = scriptRegex.ReplaceAllString(content, "")
	
	// Remove javascript: and vbscript: protocols
	jsRegex := regexp.MustCompile(`(?i)(javascript|vbscript):`)
	content = jsRegex.ReplaceAllString(content, "")
	
	// Normalize whitespace
	content = strings.TrimSpace(content)
	content = regexp.MustCompile(`\s+`).ReplaceAllString(content, " ")
	
	return content
}

// AddPrompt adds a new prompt to the composer
func (c *Composer) AddPrompt(prompt SystemPromptConfig) {
	switch prompt.Scope {
	case PromptScopeGlobal:
		c.GlobalPrompts = append(c.GlobalPrompts, prompt)
	case PromptScopeOrg:
		c.OrgPrompts = append(c.OrgPrompts, prompt)
	case PromptScopeUser:
		c.UserPrompts = append(c.UserPrompts, prompt)
	}
}

// RemovePrompt removes a prompt by ID
func (c *Composer) RemovePrompt(id string) {
	c.GlobalPrompts = removePromptByID(c.GlobalPrompts, id)
	c.OrgPrompts = removePromptByID(c.OrgPrompts, id)
	c.UserPrompts = removePromptByID(c.UserPrompts, id)
}

// UpdatePrompt updates an existing prompt
func (c *Composer) UpdatePrompt(prompt SystemPromptConfig) {
	c.RemovePrompt(prompt.ID)
	c.AddPrompt(prompt)
}

func removePromptByID(prompts []SystemPromptConfig, id string) []SystemPromptConfig {
	var result []SystemPromptConfig
	for _, prompt := range prompts {
		if prompt.ID != id {
			result = append(result, prompt)
		}
	}
	return result
}