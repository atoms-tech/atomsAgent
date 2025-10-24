package perf

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/coder/agentapi/lib/auth"
	"github.com/coder/agentapi/lib/mcp"
	"github.com/coder/agentapi/lib/redis"
	"github.com/coder/agentapi/lib/session"
	"github.com/golang-jwt/jwt/v5"
)

// Phase 1 baseline metrics (for comparison and regression tracking)
const (
	// Session management baselines (microseconds)
	baselineCreateSession  = 5000 // 5ms
	baselineGetSession     = 100  // 100µs
	baselineCleanupSession = 3000 // 3ms

	// Authentication baselines (microseconds)
	baselineValidateJWT = 500 // 500µs
	baselineRoleCheck   = 50  // 50µs

	// MCP operations baselines (milliseconds)
	baselineCallTool   = 100 // 100ms
	baselineListTools  = 50  // 50ms
	baselineConnectMCP = 200 // 200ms

	// Redis operations baselines (microseconds)
	baselineRedisSet         = 1000 // 1ms
	baselineRedisGet         = 500  // 500µs
	baselineRedisTransaction = 2000 // 2ms

	// Rate limiting baselines (microseconds)
	baselineRateLimitCheck = 100 // 100µs
)

// Test fixtures
var (
	testSessionManager *session.SessionManagerV2
	testRedisClient    *redis.RedisClient
	testAuthMiddleware *auth.AuthMiddleware
	testLogger         *slog.Logger
	testWorkspaceRoot  string
	testRSAPrivateKey  *rsa.PrivateKey
	testRSAPublicKey   *rsa.PublicKey
)

// setupBenchmarkFixtures initializes test fixtures for benchmarking
func setupBenchmarkFixtures(b *testing.B) {
	b.Helper()

	// Create temporary workspace directory
	var err error
	testWorkspaceRoot, err = os.MkdirTemp("", "agentapi-bench-*")
	if err != nil {
		b.Fatalf("Failed to create temp workspace: %v", err)
	}

	// Initialize logger
	testLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Quiet during benchmarks
	}))

	// Initialize session manager
	testSessionManager = session.NewSessionManagerV2WithLogger(
		testWorkspaceRoot,
		1000, // max concurrent sessions
		testLogger,
	)

	// Initialize Redis client (using test/mock configuration)
	redisConfig := redis.DefaultConfig()
	redisConfig.URL = getTestRedisURL()
	testRedisClient, err = redis.NewRedisClient(redisConfig)
	if err != nil {
		b.Logf("Redis not available, using in-memory only: %v", err)
		testRedisClient = nil
	}

	// Generate RSA keys for JWT testing
	testRSAPrivateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		b.Fatalf("Failed to generate RSA key: %v", err)
	}
	testRSAPublicKey = &testRSAPrivateKey.PublicKey

	// Setup auth middleware (will be initialized per benchmark)
	// We don't initialize it here because it requires JWKS URL
}

// cleanupBenchmarkFixtures cleans up test fixtures
func cleanupBenchmarkFixtures(b *testing.B) {
	b.Helper()

	if testWorkspaceRoot != "" {
		os.RemoveAll(testWorkspaceRoot)
	}

	if testRedisClient != nil {
		testRedisClient.Close()
	}
}

// getTestRedisURL returns the Redis URL for testing
func getTestRedisURL() string {
	if url := os.Getenv("REDIS_URL"); url != "" {
		return url
	}
	// Default to local Redis for testing
	return "redis://localhost:6379/0"
}

// createTestJWT creates a test JWT token
func createTestJWT(b *testing.B, claims *auth.Claims) string {
	b.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-key-id"

	tokenString, err := token.SignedString(testRSAPrivateKey)
	if err != nil {
		b.Fatalf("Failed to sign token: %v", err)
	}

	return tokenString
}

// ============================================================================
// Session Management Benchmarks
// ============================================================================

// BenchmarkCreateSession measures session creation time
func BenchmarkCreateSession(b *testing.B) {
	setupBenchmarkFixtures(b)
	defer cleanupBenchmarkFixtures(b)

	ctx := context.Background()
	userID := "user-bench-001"
	orgID := "org-bench-001"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		sess, err := testSessionManager.CreateSession(ctx, userID, orgID)
		if err != nil {
			b.Fatalf("CreateSession failed: %v", err)
		}

		// Cleanup immediately to avoid memory bloat
		testSessionManager.CleanupSession(ctx, sess.ID)
	}

	b.StopTimer()

	// Calculate and report performance vs baseline
	nsPerOp := b.Elapsed().Nanoseconds() / int64(b.N)
	baselineNs := baselineCreateSession * 1000
	performanceRatio := float64(nsPerOp) / float64(baselineNs)

	b.ReportMetric(performanceRatio, "baseline_ratio")
	if performanceRatio > 1.5 {
		b.Logf("WARNING: Performance degraded %.2fx vs baseline", performanceRatio)
	}
}

// BenchmarkGetSession measures session retrieval performance
func BenchmarkGetSession(b *testing.B) {
	setupBenchmarkFixtures(b)
	defer cleanupBenchmarkFixtures(b)

	ctx := context.Background()
	userID := "user-bench-001"
	orgID := "org-bench-001"

	// Create a session once
	sess, err := testSessionManager.CreateSession(ctx, userID, orgID)
	if err != nil {
		b.Fatalf("CreateSession failed: %v", err)
	}
	defer testSessionManager.CleanupSession(ctx, sess.ID)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := testSessionManager.GetSession(sess.ID)
		if err != nil {
			b.Fatalf("GetSession failed: %v", err)
		}
	}

	b.StopTimer()

	// Performance comparison
	nsPerOp := b.Elapsed().Nanoseconds() / int64(b.N)
	baselineNs := baselineGetSession * 1000
	performanceRatio := float64(nsPerOp) / float64(baselineNs)

	b.ReportMetric(performanceRatio, "baseline_ratio")
	if performanceRatio > 1.5 {
		b.Logf("WARNING: Performance degraded %.2fx vs baseline", performanceRatio)
	}
}

// BenchmarkCleanupSession measures session cleanup time
func BenchmarkCleanupSession(b *testing.B) {
	setupBenchmarkFixtures(b)
	defer cleanupBenchmarkFixtures(b)

	ctx := context.Background()
	userID := "user-bench-001"
	orgID := "org-bench-001"

	// Pre-create sessions
	sessions := make([]*session.Session, b.N)
	for i := 0; i < b.N; i++ {
		sess, err := testSessionManager.CreateSession(ctx, userID, orgID)
		if err != nil {
			b.Fatalf("CreateSession failed: %v", err)
		}
		sessions[i] = sess
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := testSessionManager.CleanupSession(ctx, sessions[i].ID)
		if err != nil {
			b.Fatalf("CleanupSession failed: %v", err)
		}
	}

	b.StopTimer()

	// Performance comparison
	nsPerOp := b.Elapsed().Nanoseconds() / int64(b.N)
	baselineNs := baselineCleanupSession * 1000
	performanceRatio := float64(nsPerOp) / float64(baselineNs)

	b.ReportMetric(performanceRatio, "baseline_ratio")
	if performanceRatio > 1.5 {
		b.Logf("WARNING: Performance degraded %.2fx vs baseline", performanceRatio)
	}
}

// BenchmarkSessionManagerConcurrent measures concurrent session operations
func BenchmarkSessionManagerConcurrent(b *testing.B) {
	setupBenchmarkFixtures(b)
	defer cleanupBenchmarkFixtures(b)

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			userID := fmt.Sprintf("user-%d", i)
			orgID := fmt.Sprintf("org-%d", i%10) // 10 orgs
			i++

			// Create session
			sess, err := testSessionManager.CreateSession(ctx, userID, orgID)
			if err != nil {
				b.Errorf("CreateSession failed: %v", err)
				continue
			}

			// Get session
			_, err = testSessionManager.GetSession(sess.ID)
			if err != nil {
				b.Errorf("GetSession failed: %v", err)
			}

			// Cleanup
			testSessionManager.CleanupSession(ctx, sess.ID)
		}
	})
}

// ============================================================================
// Authentication Benchmarks
// ============================================================================

// BenchmarkValidateJWT measures JWT validation performance
func BenchmarkValidateJWT(b *testing.B) {
	setupBenchmarkFixtures(b)
	defer cleanupBenchmarkFixtures(b)

	// Create mock key manager
	keyManager := &mockKeyManager{
		key: testRSAPublicKey,
	}

	// Create test claims
	claims := &auth.Claims{
		Sub:   "user-123",
		Email: "test@example.com",
		OrgID: "org-456",
		Role:  auth.RoleUser,
		Exp:   time.Now().Add(1 * time.Hour).Unix(),
		Iat:   time.Now().Unix(),
	}

	// Generate test token
	tokenString := createTestJWT(b, claims)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &auth.Claims{}, func(token *jwt.Token) (interface{}, error) {
			return keyManager.key, nil
		})

		if err != nil {
			b.Fatalf("Token validation failed: %v", err)
		}

		if !token.Valid {
			b.Fatalf("Token is invalid")
		}
	}

	b.StopTimer()

	// Performance comparison
	nsPerOp := b.Elapsed().Nanoseconds() / int64(b.N)
	baselineNs := baselineValidateJWT * 1000
	performanceRatio := float64(nsPerOp) / float64(baselineNs)

	b.ReportMetric(performanceRatio, "baseline_ratio")
	if performanceRatio > 1.5 {
		b.Logf("WARNING: Performance degraded %.2fx vs baseline", performanceRatio)
	}
}

// BenchmarkRoleCheck measures role-based access check performance
func BenchmarkRoleCheck(b *testing.B) {
	setupBenchmarkFixtures(b)
	defer cleanupBenchmarkFixtures(b)

	// Create context with claims
	claims := &auth.Claims{
		Sub:   "user-123",
		Email: "test@example.com",
		OrgID: "org-456",
		Role:  auth.RoleAdmin,
		Exp:   time.Now().Add(1 * time.Hour).Unix(),
		Iat:   time.Now().Unix(),
	}

	ctx := context.WithValue(context.Background(), auth.ContextKeyClaims, claims)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		isAdmin := auth.IsAdmin(ctx)
		if !isAdmin {
			b.Fatalf("Expected admin role")
		}
	}

	b.StopTimer()

	// Performance comparison
	nsPerOp := b.Elapsed().Nanoseconds() / int64(b.N)
	baselineNs := baselineRoleCheck * 1000
	performanceRatio := float64(nsPerOp) / float64(baselineNs)

	b.ReportMetric(performanceRatio, "baseline_ratio")
	if performanceRatio > 1.5 {
		b.Logf("WARNING: Performance degraded %.2fx vs baseline", performanceRatio)
	}
}

// BenchmarkGetUserFromContext measures context extraction performance
func BenchmarkGetUserFromContext(b *testing.B) {
	setupBenchmarkFixtures(b)
	defer cleanupBenchmarkFixtures(b)

	claims := &auth.Claims{
		Sub:   "user-123",
		Email: "test@example.com",
		OrgID: "org-456",
		Role:  auth.RoleUser,
	}

	ctx := context.WithValue(context.Background(), auth.ContextKeyClaims, claims)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		userID, orgID, err := auth.GetUserFromContext(ctx)
		if err != nil {
			b.Fatalf("GetUserFromContext failed: %v", err)
		}
		if userID != "user-123" || orgID != "org-456" {
			b.Fatalf("Invalid user/org ID")
		}
	}
}

// ============================================================================
// MCP Operations Benchmarks
// ============================================================================

// BenchmarkCallTool measures tool execution latency (mocked)
func BenchmarkCallTool(b *testing.B) {
	setupBenchmarkFixtures(b)
	defer cleanupBenchmarkFixtures(b)

	ctx := context.Background()

	// Create mock MCP client
	mockClient := &mockMCPClient{
		delay: 1 * time.Millisecond, // Simulate network latency
	}

	toolName := "test-tool"
	args := map[string]any{
		"param1": "value1",
		"param2": 42,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result, err := mockClient.CallTool(ctx, toolName, args)
		if err != nil {
			b.Fatalf("CallTool failed: %v", err)
		}
		if result == nil {
			b.Fatalf("Expected non-nil result")
		}
	}

	b.StopTimer()

	// Performance comparison
	nsPerOp := b.Elapsed().Nanoseconds() / int64(b.N)
	baselineNs := baselineCallTool * 1000000 // Convert ms to ns
	performanceRatio := float64(nsPerOp) / float64(baselineNs)

	b.ReportMetric(performanceRatio, "baseline_ratio")
}

// BenchmarkListTools measures list tools performance (mocked)
func BenchmarkListTools(b *testing.B) {
	setupBenchmarkFixtures(b)
	defer cleanupBenchmarkFixtures(b)

	ctx := context.Background()

	// Create mock MCP client with 50 tools
	mockClient := &mockMCPClient{
		toolCount: 50,
		delay:     500 * time.Microsecond,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		tools, err := mockClient.ListTools(ctx)
		if err != nil {
			b.Fatalf("ListTools failed: %v", err)
		}
		if len(tools) != 50 {
			b.Fatalf("Expected 50 tools, got %d", len(tools))
		}
	}

	b.StopTimer()

	// Performance comparison
	nsPerOp := b.Elapsed().Nanoseconds() / int64(b.N)
	baselineNs := baselineListTools * 1000000
	performanceRatio := float64(nsPerOp) / float64(baselineNs)

	b.ReportMetric(performanceRatio, "baseline_ratio")
}

// BenchmarkConnectMCP measures MCP connection time (mocked)
func BenchmarkConnectMCP(b *testing.B) {
	setupBenchmarkFixtures(b)
	defer cleanupBenchmarkFixtures(b)

	ctx := context.Background()

	config := mcp.MCPConfig{
		ID:       "test-mcp",
		Name:     "Test MCP",
		Type:     "http",
		Endpoint: "http://localhost:8080",
		AuthType: "none",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mockClient := &mockMCPClient{
			delay: 2 * time.Millisecond, // Simulate connection overhead
		}

		err := mockClient.Connect(ctx, config)
		if err != nil {
			b.Fatalf("Connect failed: %v", err)
		}

		mockClient.Disconnect(ctx)
	}

	b.StopTimer()

	// Performance comparison
	nsPerOp := b.Elapsed().Nanoseconds() / int64(b.N)
	baselineNs := baselineConnectMCP * 1000000
	performanceRatio := float64(nsPerOp) / float64(baselineNs)

	b.ReportMetric(performanceRatio, "baseline_ratio")
}

// ============================================================================
// Redis Operations Benchmarks
// ============================================================================

// BenchmarkRedisSet measures Redis set operation performance
func BenchmarkRedisSet(b *testing.B) {
	if testRedisClient == nil {
		b.Skip("Redis not available")
	}

	setupBenchmarkFixtures(b)
	defer cleanupBenchmarkFixtures(b)

	ctx := context.Background()
	key := "bench:set"
	value := "test-value-" + generateRandomString(100)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := testRedisClient.Set(ctx, fmt.Sprintf("%s:%d", key, i), value, 1*time.Minute)
		if err != nil {
			b.Fatalf("Redis Set failed: %v", err)
		}
	}

	b.StopTimer()

	// Cleanup
	for i := 0; i < b.N; i++ {
		testRedisClient.Delete(ctx, fmt.Sprintf("%s:%d", key, i))
	}

	// Performance comparison
	nsPerOp := b.Elapsed().Nanoseconds() / int64(b.N)
	baselineNs := baselineRedisSet * 1000
	performanceRatio := float64(nsPerOp) / float64(baselineNs)

	b.ReportMetric(performanceRatio, "baseline_ratio")
	if performanceRatio > 1.5 {
		b.Logf("WARNING: Performance degraded %.2fx vs baseline", performanceRatio)
	}
}

// BenchmarkRedisGet measures Redis get operation performance
func BenchmarkRedisGet(b *testing.B) {
	if testRedisClient == nil {
		b.Skip("Redis not available")
	}

	setupBenchmarkFixtures(b)
	defer cleanupBenchmarkFixtures(b)

	ctx := context.Background()
	key := "bench:get"
	value := "test-value"

	// Pre-populate key
	err := testRedisClient.Set(ctx, key, value, 1*time.Minute)
	if err != nil {
		b.Fatalf("Redis Set failed: %v", err)
	}
	defer testRedisClient.Delete(ctx, key)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result, err := testRedisClient.Get(ctx, key)
		if err != nil {
			b.Fatalf("Redis Get failed: %v", err)
		}
		if result != value {
			b.Fatalf("Expected %s, got %s", value, result)
		}
	}

	b.StopTimer()

	// Performance comparison
	nsPerOp := b.Elapsed().Nanoseconds() / int64(b.N)
	baselineNs := baselineRedisGet * 1000
	performanceRatio := float64(nsPerOp) / float64(baselineNs)

	b.ReportMetric(performanceRatio, "baseline_ratio")
	if performanceRatio > 1.5 {
		b.Logf("WARNING: Performance degraded %.2fx vs baseline", performanceRatio)
	}
}

// BenchmarkRedisTransaction measures transactional operations
func BenchmarkRedisTransaction(b *testing.B) {
	if testRedisClient == nil {
		b.Skip("Redis not available")
	}

	setupBenchmarkFixtures(b)
	defer cleanupBenchmarkFixtures(b)

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key1 := fmt.Sprintf("bench:tx1:%d", i)
		key2 := fmt.Sprintf("bench:tx2:%d", i)

		// Simulate a transaction with multiple operations
		err := testRedisClient.Set(ctx, key1, "value1", 1*time.Minute)
		if err != nil {
			b.Fatalf("Transaction Set 1 failed: %v", err)
		}

		err = testRedisClient.Set(ctx, key2, "value2", 1*time.Minute)
		if err != nil {
			b.Fatalf("Transaction Set 2 failed: %v", err)
		}

		_, err = testRedisClient.Get(ctx, key1)
		if err != nil {
			b.Fatalf("Transaction Get failed: %v", err)
		}
	}

	b.StopTimer()

	// Performance comparison
	nsPerOp := b.Elapsed().Nanoseconds() / int64(b.N)
	baselineNs := baselineRedisTransaction * 1000
	performanceRatio := float64(nsPerOp) / float64(baselineNs)

	b.ReportMetric(performanceRatio, "baseline_ratio")
	if performanceRatio > 1.5 {
		b.Logf("WARNING: Performance degraded %.2fx vs baseline", performanceRatio)
	}
}

// BenchmarkRedisConcurrent measures concurrent Redis operations
func BenchmarkRedisConcurrent(b *testing.B) {
	if testRedisClient == nil {
		b.Skip("Redis not available")
	}

	setupBenchmarkFixtures(b)
	defer cleanupBenchmarkFixtures(b)

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("bench:concurrent:%d", i)
			value := fmt.Sprintf("value-%d", i)
			i++

			// Set
			err := testRedisClient.Set(ctx, key, value, 1*time.Minute)
			if err != nil {
				b.Errorf("Redis Set failed: %v", err)
				continue
			}

			// Get
			result, err := testRedisClient.Get(ctx, key)
			if err != nil {
				b.Errorf("Redis Get failed: %v", err)
				continue
			}

			if result != value {
				b.Errorf("Expected %s, got %s", value, result)
			}

			// Cleanup
			testRedisClient.Delete(ctx, key)
		}
	})
}

// ============================================================================
// Rate Limiting Benchmarks
// ============================================================================

// BenchmarkRateLimitCheck measures rate limit check performance
func BenchmarkRateLimitCheck(b *testing.B) {
	setupBenchmarkFixtures(b)
	defer cleanupBenchmarkFixtures(b)

	ctx := context.Background()

	// Simple in-memory rate limiter mock
	limiter := &mockRateLimiter{
		limit:  100,
		window: 1 * time.Second,
	}

	userID := "user-123"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		allowed := limiter.Allow(ctx, userID)
		if !allowed && i < 100 {
			// First 100 should be allowed
			b.Fatalf("Rate limiter unexpectedly denied request")
		}
	}

	b.StopTimer()

	// Performance comparison
	nsPerOp := b.Elapsed().Nanoseconds() / int64(b.N)
	baselineNs := baselineRateLimitCheck * 1000
	performanceRatio := float64(nsPerOp) / float64(baselineNs)

	b.ReportMetric(performanceRatio, "baseline_ratio")
	if performanceRatio > 1.5 {
		b.Logf("WARNING: Performance degraded %.2fx vs baseline", performanceRatio)
	}
}

// BenchmarkRateLimitConcurrent measures concurrent rate limiting
func BenchmarkRateLimitConcurrent(b *testing.B) {
	setupBenchmarkFixtures(b)
	defer cleanupBenchmarkFixtures(b)

	ctx := context.Background()

	limiter := &mockRateLimiter{
		limit:  1000,
		window: 1 * time.Second,
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			userID := fmt.Sprintf("user-%d", i%10) // 10 users
			i++
			limiter.Allow(ctx, userID)
		}
	})
}

// ============================================================================
// Integration Benchmarks (End-to-End)
// ============================================================================

// BenchmarkEndToEndSessionWithRedis measures full session lifecycle with Redis
func BenchmarkEndToEndSessionWithRedis(b *testing.B) {
	if testRedisClient == nil {
		b.Skip("Redis not available")
	}

	setupBenchmarkFixtures(b)
	defer cleanupBenchmarkFixtures(b)

	ctx := context.Background()

	// Setup Redis session store
	store := redis.NewSessionStore(testRedisClient, 1*time.Hour)
	testSessionManager.SetSessionStore(store, true)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		userID := fmt.Sprintf("user-%d", i)
		orgID := fmt.Sprintf("org-%d", i%10)

		// Create session (with Redis persistence)
		sess, err := testSessionManager.CreateSession(ctx, userID, orgID)
		if err != nil {
			b.Fatalf("CreateSession failed: %v", err)
		}

		// Retrieve from Redis
		_, err = testSessionManager.GetSession(sess.ID)
		if err != nil {
			b.Fatalf("GetSession failed: %v", err)
		}

		// Cleanup (remove from Redis)
		err = testSessionManager.CleanupSession(ctx, sess.ID)
		if err != nil {
			b.Fatalf("CleanupSession failed: %v", err)
		}
	}
}

// ============================================================================
// Mock Implementations
// ============================================================================

// mockKeyManager implements a simple key manager for testing
type mockKeyManager struct {
	key *rsa.PublicKey
}

func (m *mockKeyManager) GetKey(kid string) (*rsa.PublicKey, error) {
	return m.key, nil
}

// mockMCPClient implements a mock MCP client for testing
type mockMCPClient struct {
	toolCount int
	delay     time.Duration
}

func (m *mockMCPClient) Connect(ctx context.Context, config mcp.MCPConfig) error {
	time.Sleep(m.delay)
	return nil
}

func (m *mockMCPClient) Disconnect(ctx context.Context) error {
	return nil
}

func (m *mockMCPClient) CallTool(ctx context.Context, toolName string, args map[string]any) (map[string]any, error) {
	time.Sleep(m.delay)
	return map[string]any{
		"status": "success",
		"result": "mock result",
	}, nil
}

func (m *mockMCPClient) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	time.Sleep(m.delay)
	tools := make([]mcp.Tool, m.toolCount)
	for i := 0; i < m.toolCount; i++ {
		tools[i] = mcp.Tool{
			Name:        fmt.Sprintf("tool-%d", i),
			Description: fmt.Sprintf("Description for tool %d", i),
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"param": map[string]any{"type": "string"},
				},
			},
		}
	}
	return tools, nil
}

// mockRateLimiter implements a simple rate limiter for testing
type mockRateLimiter struct {
	limit  int
	window time.Duration
	counts map[string]int
}

func (m *mockRateLimiter) Allow(ctx context.Context, key string) bool {
	if m.counts == nil {
		m.counts = make(map[string]int)
	}
	m.counts[key]++
	return m.counts[key] <= m.limit
}

// ============================================================================
// Helper Functions
// ============================================================================

// generateRandomString generates a random string of specified length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return base64.URLEncoding.EncodeToString(b)[:length]
}

// ============================================================================
// Benchmark Comparison Report
// ============================================================================

// BenchmarkReport generates a performance report comparing against baselines
func BenchmarkReport(b *testing.B) {
	b.Skip("Use for manual reporting only")

	fmt.Println("\n=== Performance Benchmark Report ===")
	fmt.Println("\nBaseline Metrics (Phase 1):")
	fmt.Printf("  Session Create:     %dµs\n", baselineCreateSession)
	fmt.Printf("  Session Get:        %dµs\n", baselineGetSession)
	fmt.Printf("  Session Cleanup:    %dµs\n", baselineCleanupSession)
	fmt.Printf("  JWT Validation:     %dµs\n", baselineValidateJWT)
	fmt.Printf("  Role Check:         %dµs\n", baselineRoleCheck)
	fmt.Printf("  MCP Call Tool:      %dms\n", baselineCallTool)
	fmt.Printf("  MCP List Tools:     %dms\n", baselineListTools)
	fmt.Printf("  MCP Connect:        %dms\n", baselineConnectMCP)
	fmt.Printf("  Redis Set:          %dµs\n", baselineRedisSet)
	fmt.Printf("  Redis Get:          %dµs\n", baselineRedisGet)
	fmt.Printf("  Redis Transaction:  %dµs\n", baselineRedisTransaction)
	fmt.Printf("  Rate Limit Check:   %dµs\n", baselineRateLimitCheck)
	fmt.Println("\nRun benchmarks with: go test -bench=. -benchmem ./tests/perf/")
	fmt.Println("For detailed profiling: go test -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof ./tests/perf/")
	fmt.Println("Analyze with: go tool pprof cpu.prof")
}

// ============================================================================
// Benchmark Suites for Specific Scenarios
// ============================================================================

// BenchmarkHighConcurrency tests system under high concurrent load
func BenchmarkHighConcurrency(b *testing.B) {
	setupBenchmarkFixtures(b)
	defer cleanupBenchmarkFixtures(b)

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	b.SetParallelism(100) // High concurrency

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			userID := fmt.Sprintf("user-%d", i)
			orgID := fmt.Sprintf("org-%d", i%10)
			i++

			sess, err := testSessionManager.CreateSession(ctx, userID, orgID)
			if err != nil {
				b.Errorf("CreateSession failed: %v", err)
				continue
			}

			testSessionManager.CleanupSession(ctx, sess.ID)
		}
	})
}

// BenchmarkMemoryPressure tests memory allocation under load
func BenchmarkMemoryPressure(b *testing.B) {
	setupBenchmarkFixtures(b)
	defer cleanupBenchmarkFixtures(b)

	ctx := context.Background()
	sessions := make([]*session.Session, 0, b.N)

	b.ResetTimer()
	b.ReportAllocs()

	// Create many sessions to test memory pressure
	for i := 0; i < b.N; i++ {
		userID := fmt.Sprintf("user-%d", i)
		orgID := fmt.Sprintf("org-%d", i%10)

		sess, err := testSessionManager.CreateSession(ctx, userID, orgID)
		if err != nil {
			b.Fatalf("CreateSession failed: %v", err)
		}
		sessions = append(sessions, sess)
	}

	b.StopTimer()

	// Cleanup all sessions
	for _, sess := range sessions {
		testSessionManager.CleanupSession(ctx, sess.ID)
	}

	// Report memory usage
	var m testing.BenchmarkResult
	b.ReportMetric(float64(len(sessions)), "sessions_created")
}
