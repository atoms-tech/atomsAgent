# Example Benchmark Output

This document shows example output from running the performance benchmarks.

## Quick Test Run

```bash
$ go test -bench=BenchmarkCreateSession -benchtime=1s ./tests/perf/

goos: darwin
goarch: arm64
pkg: github.com/coder/agentapi/tests/perf
BenchmarkCreateSession-8    	     500	   5234567 ns/op	baseline_ratio:1.047	    4096 B/op	      32 allocs/op
PASS
ok  	github.com/coder/agentapi/tests/perf	3.145s
```

## Full Benchmark Suite

```bash
$ go test -bench=. -benchmem ./tests/perf/

goos: darwin
goarch: arm64
pkg: github.com/coder/agentapi/tests/perf

# Session Management Benchmarks
BenchmarkCreateSession-8                	     500	   5234567 ns/op	baseline_ratio:1.047	    4096 B/op	      32 allocs/op
BenchmarkGetSession-8                   	 5000000	       245 ns/op	baseline_ratio:2.450	       0 B/op	       0 allocs/op
BenchmarkCleanupSession-8               	     800	   2876543 ns/op	baseline_ratio:0.959	    2048 B/op	      18 allocs/op
BenchmarkSessionManagerConcurrent-8     	   10000	    456789 ns/op	               	    3072 B/op	      28 allocs/op

# Authentication Benchmarks
BenchmarkValidateJWT-8                  	   30000	     78912 ns/op	baseline_ratio:0.158	     512 B/op	       8 allocs/op
BenchmarkRoleCheck-8                    	100000000	      12.5 ns/op	baseline_ratio:0.250	       0 B/op	       0 allocs/op
BenchmarkGetUserFromContext-8           	50000000	      34.2 ns/op	               	       0 B/op	       0 allocs/op

# MCP Operations Benchmarks
BenchmarkCallTool-8                     	   10000	   1234567 ns/op	baseline_ratio:0.012	    1024 B/op	      12 allocs/op
BenchmarkListTools-8                    	   20000	    567890 ns/op	baseline_ratio:0.011	    8192 B/op	      52 allocs/op
BenchmarkConnectMCP-8                   	    5000	   2345678 ns/op	baseline_ratio:0.012	    2048 B/op	      24 allocs/op

# Redis Operations Benchmarks
BenchmarkRedisSet-8                     	   15000	    876543 ns/op	baseline_ratio:0.877	     256 B/op	       6 allocs/op
BenchmarkRedisGet-8                     	   20000	    456789 ns/op	baseline_ratio:0.914	     128 B/op	       4 allocs/op
BenchmarkRedisTransaction-8             	    8000	   1876543 ns/op	baseline_ratio:0.938	     512 B/op	      14 allocs/op
BenchmarkRedisConcurrent-8              	   12000	    678901 ns/op	               	     384 B/op	      10 allocs/op

# Rate Limiting Benchmarks
BenchmarkRateLimitCheck-8               	100000000	      89.4 ns/op	baseline_ratio:0.894	       0 B/op	       0 allocs/op
BenchmarkRateLimitConcurrent-8          	50000000	      123 ns/op	               	       0 B/op	       0 allocs/op

# Integration Benchmarks
BenchmarkEndToEndSessionWithRedis-8     	     300	   6789012 ns/op	               	    6144 B/op	      58 allocs/op
BenchmarkHighConcurrency-8              	    2000	    987654 ns/op	               	    4096 B/op	      35 allocs/op
BenchmarkMemoryPressure-8               	     100	  12345678 ns/op	               	  524288 B/op	    1024 allocs/op

PASS
ok  	github.com/coder/agentapi/tests/perf	45.234s
```

## With CPU Profiling

```bash
$ go test -bench=BenchmarkCreateSession -cpuprofile=cpu.prof ./tests/perf/

goos: darwin
goarch: arm64
pkg: github.com/coder/agentapi/tests/perf
BenchmarkCreateSession-8    	     500	   5234567 ns/op	baseline_ratio:1.047	    4096 B/op	      32 allocs/op
PASS
ok  	github.com/coder/agentapi/tests/perf	3.145s

$ go tool pprof cpu.prof
File: perf.test
Type: cpu
Time: Oct 23, 2025 at 11:49pm (PDT)
Duration: 3.14s, Total samples = 2.98s (94.90%)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top10
Showing nodes accounting for 2580ms, 86.58% of 2980ms total
Dropped 89 nodes (cum <= 14.90ms)
Showing top 10 nodes out of 78
      flat  flat%   sum%        cum   cum%
     780ms 26.17% 26.17%      780ms 26.17%  os.MkdirAll
     520ms 17.45% 43.62%      520ms 17.45%  github.com/google/uuid.New
     380ms 12.75% 56.37%      380ms 12.75%  sync.(*Map).Store
     230ms  7.72% 64.09%      230ms  7.72%  runtime.mallocgc
     180ms  6.04% 70.13%      180ms  6.04%  time.Now
     150ms  5.03% 75.17%      150ms  5.03%  crypto/rand.Read
     120ms  4.03% 79.19%      120ms  4.03%  path/filepath.Join
     100ms  3.36% 82.55%      100ms  3.36%  os.RemoveAll
      80ms  2.68% 85.23%       80ms  2.68%  encoding/json.Marshal
      40ms  1.34% 86.58%       40ms  1.34%  strings.TrimSuffix
```

## Statistical Analysis with benchstat

```bash
$ go test -bench=BenchmarkCreateSession -count=10 ./tests/perf/ > results.txt
$ benchstat results.txt

name                 time/op
CreateSession-8      5.23ms ± 2%

name                 alloc/op
CreateSession-8      4.10kB ± 0%

name                 allocs/op
CreateSession-8       32.0 ± 0%
```

## Comparing Before and After

```bash
# Before changes
$ go test -bench=. -benchmem ./tests/perf/ > before.txt

# After changes
$ go test -bench=. -benchmem ./tests/perf/ > after.txt

# Compare
$ benchstat before.txt after.txt

name                         old time/op    new time/op    delta
CreateSession-8                5.23ms ± 2%    4.87ms ± 3%   -6.89%  (p=0.000 n=10+10)
GetSession-8                    245ns ± 1%     238ns ± 2%   -2.86%  (p=0.001 n=10+10)
CleanupSession-8               2.88ms ± 1%    2.95ms ± 2%   +2.43%  (p=0.003 n=10+10)
ValidateJWT-8                  78.9µs ± 3%    76.2µs ± 2%   -3.42%  (p=0.000 n=10+10)
RoleCheck-8                    12.5ns ± 1%    12.3ns ± 1%   -1.60%  (p=0.023 n=10+10)

name                         old alloc/op   new alloc/op   delta
CreateSession-8                4.10kB ± 0%    3.84kB ± 0%   -6.34%  (p=0.000 n=10+10)
GetSession-8                    0.00B          0.00B          ~     (all equal)
CleanupSession-8               2.05kB ± 0%    2.05kB ± 0%     ~     (all equal)
ValidateJWT-8                    512B ± 0%      512B ± 0%     ~     (all equal)
RoleCheck-8                     0.00B          0.00B          ~     (all equal)

name                         old allocs/op  new allocs/op  delta
CreateSession-8                  32.0 ± 0%      30.0 ± 0%   -6.25%  (p=0.000 n=10+10)
GetSession-8                     0.00           0.00          ~     (all equal)
CleanupSession-8                 18.0 ± 0%      18.0 ± 0%     ~     (all equal)
ValidateJWT-8                    8.00 ± 0%      8.00 ± 0%     ~     (all equal)
RoleCheck-8                      0.00           0.00          ~     (all equal)
```

## Performance Regression Detected

```bash
$ go test -bench=. -benchmem ./tests/perf/

BenchmarkCreateSession-8    	     300	   7856234 ns/op	baseline_ratio:1.571	    5120 B/op	      45 allocs/op
WARNING: Performance degraded 1.57x vs baseline

BenchmarkGetSession-8       	 3000000	       389 ns/op	baseline_ratio:3.890	     128 B/op	       2 allocs/op
WARNING: Performance degraded 3.89x vs baseline
```

## Memory Profiling

```bash
$ go test -bench=BenchmarkMemoryPressure -memprofile=mem.prof ./tests/perf/

$ go tool pprof mem.prof
File: perf.test
Type: alloc_space
Time: Oct 23, 2025 at 11:49pm (PDT)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top10
Showing nodes accounting for 512.50MB, 95.23% of 538.20MB total
Dropped 23 nodes (cum <= 2.69MB)
      flat  flat%   sum%        cum   cum%
  156.30MB 29.04% 29.04%  156.30MB 29.04%  github.com/coder/agentapi/lib/session.(*SessionManagerV2).CreateSession
   98.40MB 18.28% 47.32%   98.40MB 18.28%  os.MkdirAll
   87.50MB 16.26% 63.58%   87.50MB 16.26%  github.com/google/uuid.New
   65.20MB 12.11% 75.69%   65.20MB 12.11%  sync.(*Map).Store
   45.80MB  8.51% 84.20%   45.80MB  8.51%  time.Now
   32.10MB  5.97% 90.17%   32.10MB  5.97%  encoding/json.Marshal
   27.20MB  5.05% 95.23%   27.20MB  5.05%  crypto/rand.Read
```

## Concurrent Performance Scaling

```bash
$ go test -bench=BenchmarkHighConcurrency -cpu=1,2,4,8 ./tests/perf/

goos: darwin
goarch: arm64
pkg: github.com/coder/agentapi/tests/perf
BenchmarkHighConcurrency       	    2000	    987654 ns/op	    4096 B/op	      35 allocs/op
BenchmarkHighConcurrency-2     	    3800	    512345 ns/op	    4096 B/op	      35 allocs/op
BenchmarkHighConcurrency-4     	    7200	    268901 ns/op	    4096 B/op	      35 allocs/op
BenchmarkHighConcurrency-8     	   12000	    156789 ns/op	    4096 B/op	      35 allocs/op
PASS
ok  	github.com/coder/agentapi/tests/perf	18.234s
```

## Performance Over Time

```bash
# Week 1
BenchmarkCreateSession-8    	     500	   5234567 ns/op	baseline_ratio:1.047

# Week 2 (after optimization)
BenchmarkCreateSession-8    	     600	   4567890 ns/op	baseline_ratio:0.914

# Week 3 (after feature addition)
BenchmarkCreateSession-8    	     550	   4890123 ns/op	baseline_ratio:0.978

# Week 4 (stable)
BenchmarkCreateSession-8    	     600	   4456789 ns/op	baseline_ratio:0.891
```

## Redis vs In-Memory Comparison

```bash
# With Redis
$ export REDIS_URL="redis://localhost:6379/0"
$ go test -bench=BenchmarkEndToEndSessionWithRedis ./tests/perf/
BenchmarkEndToEndSessionWithRedis-8    	     300	   6789012 ns/op	    6144 B/op	      58 allocs/op

# Without Redis (in-memory only)
$ unset REDIS_URL
$ go test -bench=BenchmarkEndToEndSessionWithRedis ./tests/perf/
--- SKIP: BenchmarkEndToEndSessionWithRedis
    benchmarks_test.go:805: Redis not available
```

## Interpreting the Results

### Good Performance
```
BenchmarkGetSession-8    	 5000000	       245 ns/op	baseline_ratio:2.450	       0 B/op	       0 allocs/op
```
- Fast execution (245ns)
- Zero allocations (very good!)
- Note: baseline_ratio > 1 but this is for a very fast operation (<1µs)

### Concerning Performance
```
BenchmarkCreateSession-8    	     300	   7856234 ns/op	baseline_ratio:1.571	    5120 B/op	      45 allocs/op
WARNING: Performance degraded 1.57x vs baseline
```
- Slower than baseline (warning triggered)
- Higher allocation count than expected
- Should investigate with profiling

### Excellent Performance
```
BenchmarkRoleCheck-8    	100000000	      12.5 ns/op	baseline_ratio:0.250	       0 B/op	       0 allocs/op
```
- Very fast (12.5ns)
- 4x better than baseline!
- Zero allocations
- Excellent optimization

## Summary Report

After running all benchmarks:

```
=== Performance Benchmark Summary ===

Total Benchmarks Run: 20
Time: 45.234s

Session Management:
  ✓ CreateSession:  5.23ms (1.05x baseline) - OK
  ✓ GetSession:     245ns (2.45x baseline) - OK (very fast)
  ✓ CleanupSession: 2.88ms (0.96x baseline) - EXCELLENT

Authentication:
  ✓ ValidateJWT:    78.9µs (0.16x baseline) - EXCELLENT
  ✓ RoleCheck:      12.5ns (0.25x baseline) - EXCELLENT

MCP Operations:
  ✓ CallTool:       1.23ms (0.01x baseline) - EXCELLENT (mocked)
  ✓ ListTools:      568µs (0.01x baseline) - EXCELLENT (mocked)
  ✓ ConnectMCP:     2.35ms (0.01x baseline) - EXCELLENT (mocked)

Redis Operations:
  ✓ RedisSet:       877µs (0.88x baseline) - EXCELLENT
  ✓ RedisGet:       457µs (0.91x baseline) - EXCELLENT
  ✓ Transaction:    1.88ms (0.94x baseline) - EXCELLENT

Rate Limiting:
  ✓ RateLimitCheck: 89.4ns (0.89x baseline) - EXCELLENT

Overall Status: ✅ ALL BENCHMARKS PASSING
No performance regressions detected.
```
