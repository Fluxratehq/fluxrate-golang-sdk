# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2024-12-15

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

