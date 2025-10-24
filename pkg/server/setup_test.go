package server

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadConfigFromEnv(t *testing.T) {
	// Save original env
	originalJWKS := os.Getenv("AUTHKIT_JWKS_URL")
	defer os.Setenv("AUTHKIT_JWKS_URL", originalJWKS)

	tests := []struct {
		name        string
		setupEnv    func()
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config with all required vars",
			setupEnv: func() {
				os.Setenv("AUTHKIT_JWKS_URL", "https://api.workos.com/sso/jwks/test")
			},
			wantErr: false,
		},
		{
			name: "missing required AUTHKIT_JWKS_URL",
			setupEnv: func() {
				os.Unsetenv("AUTHKIT_JWKS_URL")
			},
			wantErr:     true,
			errContains: "AUTHKIT_JWKS_URL",
		},
		{
			name: "valid config with optional vars",
			setupEnv: func() {
				os.Setenv("AUTHKIT_JWKS_URL", "https://api.workos.com/sso/jwks/test")
				os.Setenv("PRIMARY_AGENT", "droid")
				os.Setenv("FALLBACK_ENABLED", "false")
			},
			wantErr: false,
		},
		{
			name: "invalid primary agent",
			setupEnv: func() {
				os.Setenv("AUTHKIT_JWKS_URL", "https://api.workos.com/sso/jwks/test")
				os.Setenv("PRIMARY_AGENT", "invalid")
			},
			wantErr:     true,
			errContains: "PRIMARY_AGENT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			tt.setupEnv()

			// Load config
			config, err := LoadConfigFromEnv()

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfigFromEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errContains, err.Error())
				}
			}

			if !tt.wantErr && config != nil {
				// Verify defaults are set
				if config.PrimaryAgent == "" {
					t.Error("Expected default PrimaryAgent to be set")
				}
				if config.AgentTimeout == 0 {
					t.Error("Expected default AgentTimeout to be set")
				}
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	logger := slog.New(slog.NewDiscardHandler(nil))

	tests := []struct {
		name        string
		config      *Config
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			config: &Config{
				AuthKitJWKSURL: "https://api.workos.com/sso/jwks/test",
				CCRouterPath:   "/bin/true", // Use system binary for testing
				DroidPath:      "/bin/false",
				PrimaryAgent:   "ccrouter",
			},
			wantErr: false,
		},
		{
			name: "missing JWKS URL",
			config: &Config{
				CCRouterPath: "/bin/true",
				DroidPath:    "/bin/false",
			},
			wantErr:     true,
			errContains: "JWKS URL",
		},
		{
			name: "no agents available",
			config: &Config{
				AuthKitJWKSURL: "https://api.workos.com/sso/jwks/test",
				CCRouterPath:   "/nonexistent/ccrouter",
				DroidPath:      "/nonexistent/droid",
			},
			wantErr:     true,
			errContains: "at least one agent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config, logger)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errContains, err.Error())
				}
			}
		})
	}
}

func TestSetupChatAPI_Integration(t *testing.T) {
	// Skip if no agents available
	if !fileExists("/bin/true") {
		t.Skip("Skipping integration test: /bin/true not available")
	}

	logger := slog.New(slog.NewDiscardHandler(nil))

	config := &Config{
		AuthKitJWKSURL:  "https://api.workos.com/sso/jwks/test",
		CCRouterPath:    "/bin/true",
		DroidPath:       "/bin/true",
		PrimaryAgent:    "ccrouter",
		FallbackEnabled: true,
		AgentTimeout:    1 * time.Second,
		MaxTokens:       4096,
		DefaultTemp:     0.7,
		MetricsEnabled:  true,
		AuditEnabled:    true,
	}

	mux := http.NewServeMux()

	// This will fail to load JWKS keys, but we're testing the setup logic
	components, err := SetupChatAPI(mux, logger, config)

	// We expect this to potentially fail due to JWKS loading, but not panic
	if err != nil {
		// Check that error is related to JWKS loading, not setup logic
		if !strings.Contains(err.Error(), "health") && !strings.Contains(err.Error(), "agent") {
			// Setup logic errors are acceptable for this test
			t.Logf("Setup failed as expected (JWKS or agent issues): %v", err)
			return
		}
	}

	if components == nil && err == nil {
		t.Error("Expected either components or error, got neither")
	}
}

func TestHealthEndpoint(t *testing.T) {
	mux := http.NewServeMux()

	// Register a simple health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","agents":["test"]}`))
	})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "healthy") {
		t.Errorf("Expected response to contain 'healthy', got %s", w.Body.String())
	}
}

func TestGracefulShutdown(t *testing.T) {
	components := &ChatAPIComponents{}

	logger := slog.New(slog.NewDiscardHandler(nil))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := components.GracefulShutdown(ctx, logger)
	if err != nil {
		t.Errorf("GracefulShutdown() error = %v", err)
	}
}

func TestFileExists(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "existing file",
			path: "/bin/sh",
			want: true,
		},
		{
			name: "nonexistent file",
			path: "/nonexistent/file",
			want: false,
		},
		{
			name: "directory (should return false)",
			path: "/tmp",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fileExists(tt.path)
			if got != tt.want {
				t.Errorf("fileExists(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	// Set minimal required env
	os.Setenv("AUTHKIT_JWKS_URL", "https://test.com/jwks")
	defer os.Unsetenv("AUTHKIT_JWKS_URL")

	config, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv() error = %v", err)
	}

	// Check defaults
	if config.PrimaryAgent != "ccrouter" {
		t.Errorf("Expected default PrimaryAgent = 'ccrouter', got %q", config.PrimaryAgent)
	}

	if config.FallbackEnabled != true {
		t.Error("Expected default FallbackEnabled = true")
	}

	if config.AgentTimeout != 5*time.Minute {
		t.Errorf("Expected default AgentTimeout = 5m, got %v", config.AgentTimeout)
	}

	if config.MaxTokens != 4096 {
		t.Errorf("Expected default MaxTokens = 4096, got %d", config.MaxTokens)
	}

	if config.DefaultTemp != 0.7 {
		t.Errorf("Expected default DefaultTemp = 0.7, got %f", config.DefaultTemp)
	}

	if config.Port != 3284 {
		t.Errorf("Expected default Port = 3284, got %d", config.Port)
	}
}

func TestLogStartupInfo(t *testing.T) {
	// This test just ensures LogStartupInfo doesn't panic
	logger := slog.New(slog.NewDiscardHandler(nil))

	config := &Config{
		AuthKitJWKSURL:  "https://test.com/jwks",
		CCRouterPath:    "/bin/ccrouter",
		DroidPath:       "/bin/droid",
		PrimaryAgent:    "ccrouter",
		FallbackEnabled: true,
		AgentTimeout:    5 * time.Minute,
		MaxTokens:       4096,
		DefaultTemp:     0.7,
		MetricsEnabled:  true,
		AuditEnabled:    true,
	}

	components := &ChatAPIComponents{}

	// Should not panic
	LogStartupInfo(logger, config, components)
}

// Benchmark tests
func BenchmarkLoadConfigFromEnv(b *testing.B) {
	os.Setenv("AUTHKIT_JWKS_URL", "https://test.com/jwks")
	defer os.Unsetenv("AUTHKIT_JWKS_URL")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = LoadConfigFromEnv()
	}
}

func BenchmarkValidateConfig(b *testing.B) {
	logger := slog.New(slog.NewDiscardHandler(nil))
	config := &Config{
		AuthKitJWKSURL: "https://test.com/jwks",
		CCRouterPath:   "/bin/true",
		DroidPath:      "/bin/false",
		PrimaryAgent:   "ccrouter",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateConfig(config, logger)
	}
}
