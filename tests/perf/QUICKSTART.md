# Quick Start Guide - Performance Benchmarks

## Prerequisites

1. Go 1.23.2 or later
2. Redis server (optional, but recommended)

## Running Your First Benchmark

### 1. Start Redis (Optional)

```bash
# Using Docker
docker run -d -p 6379:6379 redis:latest

# Or using local Redis
redis-server
```

### 2. Set Environment Variables

```bash
export REDIS_URL="redis://localhost:6379/0"
```

### 3. Run All Benchmarks

```bash
cd /Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi
go test -bench=. -benchmem ./tests/perf/
```

### 4. Run Specific Benchmark Categories

**Session Management:**
```bash
go test -bench=BenchmarkSession ./tests/perf/
```

**Authentication:**
```bash
go test -bench=BenchmarkAuth ./tests/perf/
```

**Redis Operations:**
```bash
go test -bench=BenchmarkRedis ./tests/perf/
```

**MCP Operations:**
```bash
go test -bench=BenchmarkMCP ./tests/perf/ -benchtime=10s
```

## Understanding the Output

Example output:
```
BenchmarkCreateSession-8    500    5234567 ns/op    baseline_ratio:1.047    4096 B/op    32 allocs/op
```

Interpretation:
- **BenchmarkCreateSession-8**: Benchmark name (8 = number of CPUs)
- **500**: Number of iterations run
- **5234567 ns/op**: Average time per operation (5.23ms)
- **baseline_ratio:1.047**: 4.7% slower than Phase 1 baseline
- **4096 B/op**: Bytes allocated per operation
- **32 allocs/op**: Number of allocations per operation

## Performance Targets

### Good Performance
- baseline_ratio < 1.2 (within 20% of baseline)
- Low allocation counts (< 100 allocs/op for most operations)
- Consistent results across runs

### Warning Signs
- baseline_ratio > 1.5 (50% slower than baseline)
- High allocation counts (> 500 allocs/op)
- High variation between runs

## Common Use Cases

### 1. Detect Performance Regressions

Run before code changes:
```bash
go test -bench=. -benchmem ./tests/perf/ > baseline.txt
```

Make code changes, then run again:
```bash
go test -bench=. -benchmem ./tests/perf/ > current.txt
```

Compare results:
```bash
# Install benchstat if not already installed
go install golang.org/x/perf/cmd/benchstat@latest

benchstat baseline.txt current.txt
```

### 2. Find Performance Bottlenecks

Run with CPU profiling:
```bash
go test -bench=BenchmarkCreateSession -cpuprofile=cpu.prof ./tests/perf/
go tool pprof cpu.prof
```

In pprof interactive mode:
```
(pprof) top10      # Show top 10 functions by CPU usage
(pprof) list main  # Show source code with annotations
(pprof) web        # Generate visual graph (requires graphviz)
```

### 3. Analyze Memory Usage

Run with memory profiling:
```bash
go test -bench=BenchmarkCreateSession -memprofile=mem.prof ./tests/perf/
go tool pprof mem.prof
```

In pprof:
```
(pprof) top10          # Top 10 memory allocators
(pprof) list myFunc    # Show allocations in specific function
```

### 4. Test Concurrent Performance

Run with different CPU counts:
```bash
go test -bench=BenchmarkConcurrent -cpu=1,2,4,8 ./tests/perf/
```

This will run the benchmark with 1, 2, 4, and 8 CPU cores to see how performance scales.

### 5. Long-Running Stability Test

Run benchmarks for extended period:
```bash
go test -bench=. -benchtime=60s ./tests/perf/
```

## Benchmark Workflow

### Daily Development
```bash
# Quick check
go test -bench=. -benchmem ./tests/perf/ -benchtime=1s
```

### Pre-Commit
```bash
# More thorough check
go test -bench=. -benchmem ./tests/perf/ -benchtime=5s -count=3
```

### CI/CD Pipeline
```bash
# Statistical analysis
go test -bench=. -benchmem ./tests/perf/ -benchtime=10s -count=10 > results.txt
benchstat results.txt
```

## Troubleshooting

### "Redis not available" - Benchmarks Skipped

**Solution 1**: Install and start Redis
```bash
# macOS
brew install redis
redis-server

# Ubuntu/Debian
sudo apt-get install redis-server
sudo systemctl start redis
```

**Solution 2**: Use Docker
```bash
docker run -d -p 6379:6379 redis:latest
```

**Solution 3**: Set correct Redis URL
```bash
export REDIS_URL="redis://localhost:6379/0"
```

### Inconsistent Results

**Solution 1**: Run multiple times and use benchstat
```bash
go test -bench=BenchmarkCreateSession -count=10 ./tests/perf/ > results.txt
benchstat results.txt
```

**Solution 2**: Close other applications to reduce noise

**Solution 3**: Lock CPU frequency (Linux)
```bash
# Disable CPU frequency scaling
sudo cpupower frequency-set --governor performance
```

### Out of Memory Errors

**Solution**: Reduce benchmark iterations
```bash
go test -bench=BenchmarkMemoryPressure -benchtime=100x ./tests/perf/
```

## Next Steps

- Read the full [README.md](README.md) for comprehensive documentation
- Explore individual benchmark implementations in `benchmarks_test.go`
- Add custom benchmarks for your specific use cases
- Integrate benchmarks into your CI/CD pipeline

## Tips for Best Results

1. **Close unnecessary applications** before running benchmarks
2. **Run on dedicated hardware** for production benchmarks
3. **Use consistent environment** for comparing results over time
4. **Look at trends**, not single-run results
5. **Profile when you see regressions** to understand the root cause
6. **Document baseline changes** when making intentional performance trade-offs

## Getting Help

If you encounter issues:

1. Check the [README.md](README.md) for detailed documentation
2. Review benchmark code in `benchmarks_test.go`
3. Run with `-v` flag for verbose output: `go test -v -bench=...`
4. Check that all dependencies are installed: `go mod tidy`

## Example Session

```bash
# 1. Setup environment
export REDIS_URL="redis://localhost:6379/0"
cd /Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi

# 2. Run quick test
go test -bench=BenchmarkCreateSession -benchmem ./tests/perf/

# 3. If performance looks good, run full suite
go test -bench=. -benchmem ./tests/perf/

# 4. If you see issues, profile
go test -bench=BenchmarkCreateSession -cpuprofile=cpu.prof ./tests/perf/
go tool pprof cpu.prof

# 5. Compare with baseline (if you have one)
benchstat baseline.txt current.txt
```

Congratulations! You're now ready to benchmark AgentAPI performance.
