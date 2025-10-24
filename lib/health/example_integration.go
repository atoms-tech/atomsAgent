package health

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/coder/agentapi/lib/mcp"
)

// ExampleBasicSetup demonstrates basic health check setup
func ExampleBasicSetup() {
	// Initialize database
	db, err := sql.Open("sqlite3", "./app.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize FastMCP client
	fastmcpClient, err := mcp.NewFastMCPClient()
	if err != nil {
		log.Fatal(err)
	}
	defer fastmcpClient.Close()

	// Create health checker
	checker := NewHealthChecker(db, fastmcpClient)

	// Create HTTP handler
	handler := NewHandler(checker)

	// Register routes
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Start server
	fmt.Println("Health checks available at:")
	fmt.Println("  http://localhost:8080/health - Detailed health status")
	fmt.Println("  http://localhost:8080/ready  - Readiness probe")
	fmt.Println("  http://localhost:8080/live   - Liveness probe")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

// ExampleCustomCheck demonstrates adding a custom health check
func ExampleCustomCheck() {
	checker := NewHealthChecker(nil, nil)

	// Register custom check with a wrapper
	checker.RegisterCheck("redis", HealthCheckFunc(func(ctx context.Context) error {
		// Implement Redis ping logic here
		// This is a simplified example
		// addr := "localhost:6379"
		// Example: redis.Ping(ctx, addr)
		return nil
	}))

	// Perform health check
	status := checker.Check(context.Background())
	fmt.Printf("Overall Status: %s\n", status.Overall)
}

// HealthCheckFunc is an adapter to use functions as HealthCheck
type HealthCheckFunc func(ctx context.Context) error

// Check implements HealthCheck interface
func (f HealthCheckFunc) Check(ctx context.Context) error {
	return f(ctx)
}

// ExampleKubernetesProbes demonstrates Kubernetes integration
func ExampleKubernetesProbes() {
	checker := NewHealthChecker(nil, nil)
	handler := NewHandler(checker)

	mux := http.NewServeMux()

	// Liveness probe - checks if the application is alive
	// Kubernetes will restart the pod if this fails
	mux.HandleFunc("/healthz", handler.Live)

	// Readiness probe - checks if the application is ready to serve traffic
	// Kubernetes will remove the pod from service if this fails
	mux.HandleFunc("/readyz", handler.Ready)

	// Detailed health check for monitoring
	mux.HandleFunc("/health", handler.Health)

	fmt.Println("Kubernetes probes configured:")
	fmt.Println("  Liveness:  /healthz")
	fmt.Println("  Readiness: /readyz")
	fmt.Println("  Health:    /health")
}

// ExampleAdvancedSetup demonstrates advanced configuration
func ExampleAdvancedSetup() {
	db, _ := sql.Open("sqlite3", ":memory:")
	defer db.Close()

	checker := NewHealthChecker(db, nil)

	// Add custom memory threshold
	memCheck := &MemoryCheck{
		MaxMemoryUsagePercent: 85.0, // Alert at 85% usage
	}
	checker.RegisterCheck("memory", memCheck)

	// Add external service check
	checker.RegisterCheck("external-api", HealthCheckFunc(func(ctx context.Context) error {
		client := &http.Client{
			Timeout: 3 * time.Second,
		}

		req, err := http.NewRequestWithContext(ctx, "GET", "https://api.example.com/health", nil)
		if err != nil {
			return err
		}

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("external API returned status %d", resp.StatusCode)
		}

		return nil
	}))

	// Perform health check
	status := checker.Check(context.Background())

	// Print detailed status
	fmt.Printf("Overall Status: %s\n", status.Overall)
	fmt.Printf("Timestamp: %s\n", status.Timestamp)
	fmt.Println("\nIndividual Checks:")
	for name, check := range status.Checks {
		fmt.Printf("  %s: %s (%.2fms)\n",
			name,
			check.Status,
			float64(check.Duration.Microseconds())/1000.0)
		if check.Error != "" {
			fmt.Printf("    Error: %s\n", check.Error)
		}
	}
}

// ExampleWithMiddleware demonstrates using health checks with middleware
func ExampleWithMiddleware() {
	checker := NewHealthChecker(nil, nil)
	handler := NewHandler(checker)

	mux := http.NewServeMux()

	// Add timeout middleware to health endpoints
	mux.HandleFunc("/health", WithTimeout(10*time.Second, handler.Health))
	mux.HandleFunc("/ready", WithTimeout(5*time.Second, handler.Ready))
	mux.HandleFunc("/live", WithTimeout(5*time.Second, handler.Live))

	fmt.Println("Health endpoints with timeout middleware configured")
}

// ExampleProgrammaticCheck demonstrates programmatic health checking
func ExampleProgrammaticCheck() {
	db, _ := sql.Open("sqlite3", ":memory:")
	defer db.Close()

	checker := NewHealthChecker(db, nil)

	// Check health programmatically
	ctx := context.Background()
	status := checker.Check(ctx)

	if !checker.Ready(ctx) {
		fmt.Println("System is not ready")
		// Take action: delay startup, alert, etc.
	}

	// Check specific conditions
	if status.Overall == StatusDown {
		fmt.Println("CRITICAL: System is down")
		// Send alert
	} else if status.Overall == StatusDegraded {
		fmt.Println("WARNING: System is degraded")
		// Log warning
	}

	// Check individual components
	if dbCheck, ok := status.Checks["database"]; ok {
		if dbCheck.Status != StatusUp {
			fmt.Printf("Database check failed: %s\n", dbCheck.Error)
		}
	}
}

// ExampleMonitoringIntegration demonstrates integration with monitoring systems
func ExampleMonitoringIntegration() {
	checker := NewHealthChecker(nil, nil)

	// Periodic health check for monitoring
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			status := checker.Check(context.Background())

			// Send metrics to monitoring system
			for name, check := range status.Checks {
				// Example: Send to Prometheus, DataDog, etc.
				metric := map[string]interface{}{
					"component": name,
					"status":    check.Status,
					"duration":  check.Duration.Milliseconds(),
				}

				if check.Error != "" {
					metric["error"] = check.Error
				}

				// sendToMonitoring(metric)
				_ = metric
			}
		}
	}()

	fmt.Println("Monitoring integration started")
}

// ExampleGracefulShutdown demonstrates health checks during graceful shutdown
func ExampleGracefulShutdown() {
	checker := NewHealthChecker(nil, nil)
	handler := NewHandler(checker)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Graceful shutdown
	go func() {
		// Wait for shutdown signal
		// <-shutdownChan

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Stop accepting new connections
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	fmt.Println("Server with graceful shutdown configured")
}
