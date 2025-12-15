# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.1] - 2024-12-30 (Experimental Release)

### Fixed
- **Critical batching bug**: `Track()` was sending events immediately even when `EnableBatching: true`
  - Events are now properly queued and sent in batches
  - Reduces API calls by up to 100x (e.g., 1 call per 100 events instead of 100 calls)
  - Dramatically improves throughput for high-volume scenarios
  - **Breaking change**: `Track()` now returns `nil` when batching is enabled (events are queued)
  - Use `TrackImmediate()` if you need synchronous response for individual events

### Changed
- Updated default `BatchSize` from 10 to 100 for better performance
- Updated default `MaxRetries` from 3 to 10 for better reliability

### Added
- `AllowedCustomers` configuration option for filtering events by customer ID
- Comprehensive test suite (`tests/functional_test.go`)
- Real-world benchmark suite (`tests/benchmark_test.go`)
- Test documentation (`tests/README.md`)
- Examples documentation (`examples/README.md`)

## [0.1.0] - 2024-12-16 (Experimental Release)

### Added
- Initial release
- `Track()` - Track usage events with automatic batching
- `TrackImmediate()` - Track events immediately without batching
- `Flush()` - Manually flush pending events
- `Shutdown()` - Graceful shutdown with event flushing
- Automatic retry with exponential backoff
- Idempotency support to prevent duplicate events
- Metadata support for custom event data
- Context support for cancellation and timeouts
- Debug mode for troubleshooting
- Zero external dependencies (only standard library)

### Configuration Options
- `EnableBatching` - Toggle automatic batching
- `BatchSize` - Events per batch (default: 10)
- `BatchInterval` - Flush interval (default: 5s)
- `EnableRetry` - Toggle automatic retry
- `MaxRetries` - Max retry attempts (default: 3)
- `Debug` - Enable debug logging
- `HTTPClient` - Custom HTTP client

### Examples
- Basic usage example
- HTTP middleware example
- Gin framework integration
- gRPC interceptor example

### Known Issues
- Batching does not work as expected - events are sent immediately even when batching is enabled (fixed in v0.1.1)
