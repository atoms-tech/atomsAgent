# Performance Benchmarks for AgentAPI

This directory contains comprehensive Go benchmarks for testing the performance of the AgentAPI system.

## Overview

The benchmark suite covers the following areas:

1. **Session Management**: Creation, retrieval, and cleanup of user sessions
2. **Authentication**: JWT validation and role-based access checks
3. **MCP Operations**: Tool execution, listing, and connection management
4. **Redis Operations**: Set, get, and transactional operations
5. **Rate Limiting**: Performance of rate limit checks
6. **Integration Tests**: End-to-end performance scenarios

## Running Benchmarks

### Basic Usage

Run all benchmarks:
```bash
go test -bench=. -benchmem ./tests/perf/
```

### Run Specific Benchmarks

Run only session benchmarks:
```bash
go test -bench=BenchmarkSession -benchmem ./tests/perf/
```

Run only authentication benchmarks:
```bash
go test -bench=BenchmarkAuth -benchmem ./tests/perf/
```

Run only Redis benchmarks:
```bash
go test -bench=BenchmarkRedis -benchmem ./tests/perf/
```

### Advanced Options

Run benchmarks with CPU profiling:
```bash
go test -bench=. -benchmem -cpuprofile=cpu.prof ./tests/perf/
```

Run benchmarks with memory profiling:
```bash
go test -bench=. -benchmem -memprofile=mem.prof ./tests/perf/
```

Run benchmarks with both CPU and memory profiling:
```bash
go test -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof ./tests/perf/
```

Analyze CPU profile:
```bash
go tool pprof cpu.prof
```

Analyze memory profile:
```bash
go tool pprof mem.prof
```

### Benchmark Time Control

Run benchmarks for a specific duration:
```bash
go test -bench=. -benchtime=10s ./tests/perf/
```

Run benchmarks for a specific number of iterations:
```bash
go test -bench=. -benchtime=1000x ./tests/perf/
```

### Parallel Execution

Control the number of parallel workers:
```bash
go test -bench=. -benchmem -cpu=1,2,4,8 ./tests/perf/
```

## Benchmark Metrics

Each benchmark reports the following metrics:

- **ns/op**: Nanoseconds per operation
- **B/op**: Bytes allocated per operation
- **allocs/op**: Number of allocations per operation
- **baseline_ratio**: Performance ratio compared to Phase 1 baseline

### Understanding baseline_ratio

The `baseline_ratio` metric compares current performance against Phase 1 baselines:

- **< 1.0**: Performance is better than baseline (faster)
- **= 1.0**: Performance matches baseline
- **> 1.0**: Performance is worse than baseline (slower)

**Warning**: If baseline_ratio > 1.5, a warning is logged indicating performance degradation.

## Phase 1 Baselines

The following baseline metrics are used for comparison:

| Operation | Baseline | Unit |
|-----------|----------|------|
| Session Create | 5000 | µs |
| Session Get | 100 | µs |
| Session Cleanup | 3000 | µs |
| JWT Validation | 500 | µs |
| Role Check | 50 | µs |
| MCP Call Tool | 100 | ms |
| MCP List Tools | 50 | ms |
| MCP Connect | 200 | ms |
| Redis Set | 1000 | µs |
| Redis Get | 500 | µs |
| Redis Transaction | 2000 | µs |
| Rate Limit Check | 100 | µs |

## Prerequisites

### Redis

Some benchmarks require a running Redis instance. Set the Redis URL via environment variable:

```bash
export REDIS_URL="redis://localhost:6379/0"
```

If Redis is not available, the Redis benchmarks will be skipped automatically.

### Environment Variables

Optional environment variables:

- `REDIS_URL`: Redis connection URL (default: `redis://localhost:6379/0`)
- `SUPABASE_URL`: Supabase URL for authentication tests (optional)

## Benchmark Categories

### Session Management Benchmarks

- `BenchmarkCreateSession`: Measures session creation performance
- `BenchmarkGetSession`: Measures session retrieval performance
- `BenchmarkCleanupSession`: Measures session cleanup performance
- `BenchmarkSessionManagerConcurrent`: Tests concurrent session operations

### Authentication Benchmarks

- `BenchmarkValidateJWT`: Measures JWT token validation performance
- `BenchmarkRoleCheck`: Measures role-based access check performance
- `BenchmarkGetUserFromContext`: Measures context extraction performance

### MCP Operations Benchmarks

- `BenchmarkCallTool`: Measures tool execution latency
- `BenchmarkListTools`: Measures tool listing performance
- `BenchmarkConnectMCP`: Measures MCP connection time

### Redis Operations Benchmarks

- `BenchmarkRedisSet`: Measures Redis set operation performance
- `BenchmarkRedisGet`: Measures Redis get operation performance
- `BenchmarkRedisTransaction`: Measures transactional operations
- `BenchmarkRedisConcurrent`: Tests concurrent Redis operations

### Rate Limiting Benchmarks

- `BenchmarkRateLimitCheck`: Measures rate limit check performance
- `BenchmarkRateLimitConcurrent`: Tests concurrent rate limiting

### Integration Benchmarks

- `BenchmarkEndToEndSessionWithRedis`: Full session lifecycle with Redis persistence
- `BenchmarkHighConcurrency`: System performance under high concurrent load
- `BenchmarkMemoryPressure`: Memory allocation under load

## Interpreting Results

### Example Output

```
BenchmarkCreateSession-8             500    5234567 ns/op    baseline_ratio:1.047    4096 B/op    32 allocs/op
```

This means:
- Ran 500 iterations
- Each operation took 5.23ms (5234567 nanoseconds)
- Performance is 4.7% slower than baseline (ratio 1.047)
- Each operation allocated 4096 bytes
- Each operation made 32 memory allocations

### Performance Goals

- Keep baseline_ratio < 1.2 (within 20% of baseline)
- Minimize allocations per operation
- Maintain consistent performance under concurrent load

## Continuous Integration

To run benchmarks in CI and detect regressions:

```bash
# Run benchmarks and save results
go test -bench=. -benchmem ./tests/perf/ > current.txt

# Compare with baseline
go test -bench=. -benchmem ./tests/perf/ > baseline.txt
benchcmp baseline.txt current.txt
```

Or use `benchstat` for statistical analysis:

```bash
go test -bench=. -benchmem -count=10 ./tests/perf/ > results.txt
benchstat results.txt
```

## Troubleshooting

### "Redis not available" Skips

If you see Redis benchmarks being skipped:

1. Ensure Redis is running: `redis-cli ping`
2. Check Redis URL: `echo $REDIS_URL`
3. Verify connectivity: `redis-cli -u $REDIS_URL ping`

### Out of Memory Errors

If benchmarks fail with OOM errors:

1. Reduce iteration count: `-benchtime=100x`
2. Run benchmarks individually
3. Increase system memory limits

### Inconsistent Results

For more stable results:

1. Run multiple iterations: `-count=10`
2. Use `benchstat` for statistical analysis
3. Close other applications
4. Run on a dedicated benchmark server

## Best Practices

1. **Run on consistent hardware**: Use the same machine/environment for comparisons
2. **Disable CPU scaling**: Lock CPU frequency for consistent results
3. **Close unnecessary applications**: Minimize background processes
4. **Use sufficient iterations**: Let the benchmark run long enough for stable results
5. **Analyze trends**: Look for trends across multiple runs, not single runs
6. **Profile when needed**: Use CPU/memory profiling to identify bottlenecks

## Contributing

When adding new benchmarks:

1. Follow the naming convention: `Benchmark<Component><Operation>`
2. Include baseline metrics if applicable
3. Add proper documentation
4. Include cleanup code to prevent resource leaks
5. Use `b.ResetTimer()` after setup code
6. Report allocations with `b.ReportAllocs()`

## References

- [Go Testing Package](https://pkg.go.dev/testing)
- [Benchmarking Best Practices](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go)
- [Profiling Go Programs](https://go.dev/blog/pprof)
