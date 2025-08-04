# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2025-01-03

### Added
- Health endpoint at `/health` for service monitoring
- Graceful shutdown with proper connection draining
- Metrics flushing during shutdown when metrics are enabled
- Comprehensive integration tests covering full request flow
- Test coverage increased from 10.7% to 73.0%
- Documentation for architecture and future improvements

### Fixed
- All golangci-lint issues resolved
- Test stability improvements
- Proper error handling throughout codebase

### Known Issues
- Metrics collector may not initialize properly when enabled in configuration
- Some edge cases in error paths remain difficult to test

## [0.1.0] - 2024-07-20

### Added
- Initial release
- API gateway with clean dependency injection architecture
- Rate limiting per API key (requests/second)
- Token usage limiting per API key (tokens/minute)
- Configurable API key mapping (client → upstream)
- Request validation middleware
- Metrics collection and export (JSON/CSV)
- Docker support
- Comprehensive test suite

### Security
- API key masking in logs
- Constant-time key comparison
- Request size limits