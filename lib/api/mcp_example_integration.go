package api

// This file demonstrates how to integrate the MCP API into your application

/*
Example integration in main server setup:

```go
package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coder/agentapi/lib/api"
	"github.com/coder/agentapi/lib/mcp"
	"github.com/coder/agentapi/lib/session"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// 1. Initialize database
	db, err := sql.Open("sqlite3", "./data/agentapi.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Run migrations
	if err := runMigrations(db); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// 2. Initialize FastMCP client
	fastmcpClient, err := mcp.NewFastMCPClient()
	if err != nil {
		log.Fatal("Failed to create FastMCP client:", err)
	}
	defer fastmcpClient.Close()

	// 3. Initialize session manager
	sessionMgr := session.NewSessionManager("./data/workspaces")

	// 4. Initialize audit logger
	auditLogger := api.NewAuditLogger()

	// 5. Get encryption key from environment
	encryptionKey := os.Getenv("MCP_ENCRYPTION_KEY")
	if encryptionKey == "" {
		log.Fatal("MCP_ENCRYPTION_KEY environment variable must be set")
	}

	// 6. Create MCP handler
	mcpHandler, err := api.NewMCPHandler(
		db,
		fastmcpClient,
		sessionMgr,
		auditLogger,
		encryptionKey,
	)
	if err != nil {
		log.Fatal("Failed to create MCP handler:", err)
	}

	// 7. Setup router
	router := chi.NewRouter()

	// Middleware
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Timeout(60 * time.Second))

	// Health check
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 8. Register MCP routes
	api.RegisterMCPRoutes(router, mcpHandler)

	// 9. Start server
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("Server starting on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal("Server error:", err)
	}

	log.Println("Server stopped")
}

func runMigrations(db *sql.DB) error {
	// Read and execute migration SQL
	migrationSQL, err := os.ReadFile("migrations/001_create_mcp_configurations.sql")
	if err != nil {
		return err
	}

	_, err = db.Exec(string(migrationSQL))
	return err
}
```

---

Example with JWT authentication middleware:

```go
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string `json:"user_id"`
	OrgID  string `json:"org_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func JWTAuth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing authorization header", http.StatusUnauthorized)
				return
			}

			// Check Bearer scheme
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			// Parse and validate token
			token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
				// Validate signing method
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method")
				}
				return []byte(jwtSecret), nil
			})

			if err != nil {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(*Claims)
			if !ok || !token.Valid {
				http.Error(w, "Invalid token claims", http.StatusUnauthorized)
				return
			}

			// Add claims to context
			ctx := context.WithValue(r.Context(), "userID", claims.UserID)
			ctx = context.WithValue(ctx, "orgID", claims.OrgID)
			ctx = context.WithValue(ctx, "email", claims.Email)
			ctx = context.WithValue(ctx, "role", claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Update the route registration to use JWT auth:
func setupRoutesWithAuth(jwtSecret string) *chi.Mux {
	router := chi.NewRouter()

	// Public routes
	router.Get("/health", healthCheck)

	// Protected routes
	router.Group(func(r chi.Router) {
		r.Use(JWTAuth(jwtSecret))

		// MCP routes require authentication
		api.RegisterMCPRoutes(r, mcpHandler)
	})

	return router
}
```

---

Example client code (JavaScript/TypeScript):

```typescript
// mcp-client.ts
interface MCPConfiguration {
  name: string;
  type: 'http' | 'sse' | 'stdio';
  endpoint?: string;
  command?: string;
  args?: string[];
  auth_type: 'none' | 'bearer' | 'oauth' | 'api_key';
  auth_token?: string;
  auth_header?: string;
  config?: Record<string, any>;
  scope: 'org' | 'user';
  enabled: boolean;
  description?: string;
}

class MCPClient {
  constructor(
    private baseURL: string,
    private authToken: string
  ) {}

  private async request(
    method: string,
    path: string,
    body?: any
  ): Promise<any> {
    const response = await fetch(`${this.baseURL}${path}`, {
      method,
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.authToken}`,
      },
      body: body ? JSON.stringify(body) : undefined,
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Request failed');
    }

    if (response.status === 204) {
      return null;
    }

    return response.json();
  }

  async createMCP(config: MCPConfiguration) {
    return this.request('POST', '/api/v1/mcp/configurations', config);
  }

  async listMCPs(filters?: { type?: string; enabled?: boolean }) {
    const params = new URLSearchParams();
    if (filters?.type) params.append('type', filters.type);
    if (filters?.enabled !== undefined) {
      params.append('enabled', filters.enabled.toString());
    }

    const query = params.toString() ? `?${params}` : '';
    return this.request('GET', `/api/v1/mcp/configurations${query}`);
  }

  async getMCP(id: string) {
    return this.request('GET', `/api/v1/mcp/configurations/${id}`);
  }

  async updateMCP(id: string, updates: Partial<MCPConfiguration>) {
    return this.request('PUT', `/api/v1/mcp/configurations/${id}`, updates);
  }

  async deleteMCP(id: string) {
    return this.request('DELETE', `/api/v1/mcp/configurations/${id}`);
  }

  async testConnection(config: Omit<MCPConfiguration, 'scope' | 'enabled'>) {
    return this.request('POST', '/api/v1/mcp/test', config);
  }
}

// Usage example
const client = new MCPClient('https://api.example.com', 'your-jwt-token');

// Test connection before creating
const testResult = await client.testConnection({
  name: 'Render API',
  type: 'http',
  endpoint: 'https://api.render.com/mcp',
  auth_type: 'bearer',
  auth_token: 'rnd_xxxxx',
});

if (testResult.success) {
  console.log('Available tools:', testResult.tools);

  // Create the configuration
  const mcp = await client.createMCP({
    name: 'Render API',
    type: 'http',
    endpoint: 'https://api.render.com/mcp',
    auth_type: 'bearer',
    auth_token: 'rnd_xxxxx',
    scope: 'org',
    enabled: true,
    description: 'Render cloud platform integration',
  });

  console.log('Created MCP:', mcp.id);
}
```

---

Example with session integration:

```go
// When creating a session, load and connect to configured MCPs

func (api *MultiTenantAPI) CreateSessionWithMCPs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := getUserIDFromContext(ctx)
	orgID := getOrgIDFromContext(ctx)

	// Create session
	userSession, err := api.SessionManager.CreateSession(ctx, userID, orgID, agentType, config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Load enabled MCP configurations for this user/org
	mcpConfigs, err := api.loadEnabledMCPs(ctx, userID, orgID)
	if err != nil {
		log.Printf("Failed to load MCPs: %v", err)
		// Continue without MCPs rather than failing
	}

	// Connect to each MCP
	for _, mcpConfig := range mcpConfigs {
		// Decrypt auth token
		decryptedToken, err := mcpHandler.decrypt(mcpConfig.AuthToken)
		if err != nil {
			log.Printf("Failed to decrypt token for MCP %s: %v", mcpConfig.ID, err)
			continue
		}

		// Create auth map
		authMap := make(map[string]string)
		if mcpConfig.AuthHeader != "" && decryptedToken != "" {
			authMap[mcpConfig.AuthHeader] = decryptedToken
		} else if decryptedToken != "" {
			authMap["token"] = decryptedToken
		}

		// Connect to MCP
		mcpCfg := mcp.MCPConfig{
			ID:       mcpConfig.ID,
			Name:     mcpConfig.Name,
			Type:     mcpConfig.Type,
			Endpoint: mcpConfig.Endpoint,
			AuthType: mcpConfig.AuthType,
			Config:   mcpConfig.Config,
			Auth:     authMap,
		}

		if err := api.FastMCPClient.ConnectMCP(ctx, mcpCfg); err != nil {
			log.Printf("Failed to connect to MCP %s: %v", mcpConfig.ID, err)
			// Continue with other MCPs
			continue
		}

		log.Printf("Connected to MCP %s for session %s", mcpConfig.ID, userSession.ID)
	}

	// Store MCP configs in session
	userSession.SetMCPs(convertToSessionMCPs(mcpConfigs))

	json.NewEncoder(w).Encode(userSession)
}

func (api *MultiTenantAPI) loadEnabledMCPs(ctx context.Context, userID, orgID string) ([]MCPConfiguration, error) {
	query := `
		SELECT id, name, type, endpoint, command, args, auth_type, auth_token,
		       auth_header, config
		FROM mcp_configurations
		WHERE enabled = 1 AND (org_id = ? OR user_id = ?)
		ORDER BY scope DESC, created_at ASC
	`

	rows, err := api.DB.QueryContext(ctx, query, orgID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []MCPConfiguration
	// ... scan and parse configs

	return configs, nil
}
```

*/
