# SDK Tests

This directory contains functional tests and benchmarks for the Fluxrate Go SDK.

## Test Types

- **Functional Tests** (`functional_test.go`) - Use **mock HTTP clients** for fast, isolated unit tests
- **Benchmarks** (`benchmark_test.go`) - Use **real HTTP requests** to measure actual performance

## Running Tests

### Run all functional tests
```bash
go test ./tests/
```

### Run with verbose output
```bash
go test -v ./tests/
```

### Run specific test
```bash
go test -run TestCustomerFiltering ./tests/
```

## Running Benchmarks

⚠️ **Note:** Benchmarks make **real API requests** to Fluxrate. Set your credentials first:

```bash
export BILLING_API_KEY="sk_live_your_key_here"
export BILLING_METER_TOKEN="your_meter_token_here"
```

### Run all benchmarks (Recommended 10s for stable results)
```bash
go test -bench=. -benchtime=10s ./tests/
```

### Run specific benchmark
```bash
go test -bench=BenchmarkTrackConcurrent -benchtime=10s ./tests/
```

### Run with custom duration
```bash
go test -bench=. -benchtime=30s ./tests/
```

### Run with memory stats
```bash
go test -bench=. -benchmem ./tests/
```

**Without credentials**, benchmarks will be skipped:
```
?   	github.com/Fluxratehq/fluxrate-golang-sdk/tests [no test files]
```

## Expected Output

### Functional Tests (with mocks)
```
=== RUN   TestSDKInitialization
=== RUN   TestCustomerFiltering
=== RUN   TestTrackingMethods
=== RUN   TestBatching
=== RUN   TestGracefulShutdown
=== RUN   TestContextCancellation
=== RUN   TestValidation
--- PASS: TestSDKInitialization (0.00s)
--- PASS: TestCustomerFiltering (0.00s)
--- PASS: TestTrackingMethods (0.00s)
--- PASS: TestBatching (0.10s)
--- PASS: TestGracefulShutdown (0.00s)
--- PASS: TestContextCancellation (0.10s)
--- PASS: TestValidation (0.00s)
PASS
ok      github.com/Fluxratehq/fluxrate-golang-sdk/tests 0.580s
```

### Benchmarks (with real API)
```
BenchmarkTrackImmediate-8                           1234      95234 ns/op
BenchmarkTrackWithBatching-8                        2103      52187 ns/op
BenchmarkTrackWithMetadata-8                        1156     102343 ns/op
BenchmarkTrackConcurrent-8                          3421      35124 ns/op
BenchmarkTrackConcurrentWithBatching-8              5234      23456 ns/op
BenchmarkCustomerFiltering/AllowedCustomer-8        1234      97123 ns/op
BenchmarkCustomerFiltering/DisallowedCustomer-8   1234567        987 ns/op
BenchmarkFlush-8                                   12345      98765 ns/op
BenchmarkSDKInitialization-8                        5432     221234 ns/op
BenchmarkHighThroughput-8                           4321      27654 ns/op
PASS
ok      github.com/Fluxratehq/fluxrate-golang-sdk/tests 45.123s
```

**Note:** Real benchmark times will vary based on:
- Network latency to Fluxrate API
- API server response times
- Your internet connection speed
- Concurrent load on the API

## Understanding Benchmark Results

Benchmarks show **ns/op** (nanoseconds per operation). To calculate **requests per second**:

**Formula:** `req/s = 1,000,000,000 / ns/op`

### Example with Real API:
```
BenchmarkTrackConcurrent-8    3421    35124 ns/op
```

- **35124 ns/op** → `1,000,000,000 / 35124` = **~28,470 req/s**
- **-8** = 8 CPU cores used
- **Actual throughput** ≈ **28,470 req/s** (already accounts for concurrency)

### Finding Maximum Throughput

Run the concurrent high-throughput benchmark with longer duration:

```bash
# Run for 30 seconds to get stable results
go test -bench=BenchmarkHighThroughput -benchtime=30s ./tests/

# Example output:
# BenchmarkHighThroughput-8    890123    27654 ns/op
# Throughput = 1,000,000,000 / 27654 = ~36,160 req/s
```

### Comparing Scenarios

```bash
# Compare batching vs immediate
go test -bench="BenchmarkTrack(Immediate|WithBatching)" -benchtime=10s ./tests/
```

**Interpretation:** Lower ns/op = better performance. Real-world latency will be higher than mock tests.

## Test Coverage

The test suite covers:

- ✅ SDK initialization and configuration (mocked)
- ✅ Customer filtering (mocked)
- ✅ Event tracking - immediate and batched (mocked)
- ✅ Metadata and idempotency support (mocked)
- ✅ Manual batch flushing (mocked)
- ✅ Graceful shutdown (mocked)
- ✅ Context cancellation (mocked)
- ✅ Concurrent access patterns (mocked)
- ✅ **Real-world performance benchmarks** (requires credentials)
