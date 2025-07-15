# Contributing to Nexus

Thank you for your interest in contributing to Nexus! This document provides guidelines and information for contributors.

## Code of Conduct

By participating in this project, you agree to abide by our code of conduct. Please be respectful and professional in all interactions.

## Getting Started

### Prerequisites

- **Go 1.22 or later** - Download from [golang.org](https://golang.org/dl/)
- **Git** - For version control
- **Make** - For build automation (optional but recommended)

### Development Setup

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/your-username/nexus.git
   cd nexus
   ```

3. **Add the upstream remote**:
   ```bash
   git remote add upstream https://github.com/jamesprial/nexus.git
   ```

4. **Install dependencies**:
   ```bash
   make deps
   ```

5. **Run tests** to ensure everything works:
   ```bash
   make test
   ```

6. **Build the project**:
   ```bash
   make build
   ```

## Development Workflow

### Creating a Feature Branch

```bash
# Get the latest changes
git checkout main
git pull upstream main

# Create your feature branch
git checkout -b feature/your-feature-name
```

### Making Changes

1. **Write tests first** (TDD approach preferred)
2. **Make your changes** with clear, focused commits
3. **Run tests frequently**:
   ```bash
   make test
   ```
4. **Run linting**:
   ```bash
   make lint
   ```
5. **Test your changes manually**:
   ```bash
   make build
   ./build/nexus
   ```

### Commit Guidelines

We follow [Conventional Commits](https://www.conventionalcommits.org/) format:

- `feat: add new feature`
- `fix: bug fix`
- `docs: documentation changes`
- `refactor: code refactoring`
- `test: add or modify tests`
- `chore: maintenance tasks`

**Examples:**
```bash
git commit -m "feat: add token-based rate limiting"
git commit -m "fix: resolve memory leak in client tracking"
git commit -m "docs: update configuration examples"
```

### Submitting Changes

1. **Push your branch**:
   ```bash
   git push origin feature/your-feature-name
   ```

2. **Create a Pull Request** on GitHub with:
   - Clear title and description
   - Reference to any related issues
   - Screenshots/examples if UI changes
   - List of changes made

3. **Address review feedback** promptly

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run tests with race detection
go test -race ./...

# Run specific test
go test ./internal/proxy -v
```

### Writing Tests

- **Unit tests** for all new functions/methods
- **Integration tests** for complete workflows
- **Table-driven tests** for multiple test cases
- **Mock external dependencies** when needed

Example test structure:
```go
func TestTokenLimiter(t *testing.T) {
    tests := []struct {
        name     string
        tpm      int
        expected float64
    }{
        {"basic conversion", 60, 1.0},
        {"zero tokens", 0, 0.0},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            limiter := NewTokenLimiter(tt.tpm, 10)
            if limiter.tps != tt.expected {
                t.Errorf("expected %f, got %f", tt.expected, limiter.tps)
            }
        })
    }
}
```

## Code Style

### Go Conventions

- Follow standard Go formatting (`go fmt`)
- Use meaningful variable and function names
- Add comments for exported functions
- Keep functions focused and small
- Handle errors appropriately

### Project Structure

```
nexus/
├── cmd/gateway/          # Application entrypoint
├── internal/
│   ├── config/          # Configuration management
│   ├── container/       # Dependency injection
│   ├── gateway/         # Core gateway logic
│   ├── proxy/           # Rate limiting and proxying
│   └── interfaces/      # Shared interfaces
├── tests/               # Integration tests
└── config.yaml          # Default configuration
```

### Error Handling

- Always check and handle errors
- Use descriptive error messages
- Wrap errors with context when needed
- Log errors appropriately

```go
// Good
cfg, err := config.Load(configPath)
if err != nil {
    return fmt.Errorf("failed to load config from %s: %w", configPath, err)
}

// Bad
cfg, _ := config.Load(configPath)
```

## Documentation

### Code Documentation

- Document all exported functions and types
- Use clear, concise comments
- Include examples in godoc format

```go
// NewTokenLimiter creates a new TokenLimiter with the specified limits.
// tpm: tokens per minute limit
// burst: maximum burst allowance in tokens
func NewTokenLimiter(tpm, burst int) *TokenLimiter {
    // implementation
}
```

### User Documentation

- Update relevant documentation when making changes
- Add examples for new features
- Keep README.md and USAGE.md up to date

## Performance Considerations

- **Avoid memory leaks** - clean up resources properly
- **Use efficient data structures** - maps for lookups, slices for iteration
- **Profile performance** for critical paths
- **Consider concurrency** - use goroutines and channels appropriately

## Security

- **Validate all inputs** from HTTP requests
- **Sanitize error messages** to avoid information leakage
- **Use secure defaults** in configuration
- **Follow OWASP guidelines** for web applications

## Release Process

1. **Version bumping** follows semantic versioning (semver)
2. **Changelog** is updated with notable changes
3. **Release notes** describe new features and fixes
4. **Binaries** are built for multiple platforms

## Getting Help

- **GitHub Issues** - Report bugs or request features
- **GitHub Discussions** - Ask questions or discuss ideas
- **Code Review** - Learn from feedback on your PRs

## Recognition

Contributors are recognized in:
- GitHub contributor statistics
- Release notes for significant contributions
- Special thanks in documentation

## Areas for Contribution

### High Priority
- Token counting accuracy improvements
- Memory usage optimization
- Additional rate limiting strategies
- Metrics and monitoring features

### Medium Priority
- Support for additional AI providers
- Configuration validation
- Performance benchmarking
- Documentation improvements

### Good First Issues
- Add configuration examples
- Improve error messages
- Add unit tests
- Update documentation

## Questions?

If you have questions about contributing, please:
1. Check existing GitHub issues and discussions
2. Create a new issue with the "question" label
3. Reach out to maintainers in your PR or issue

Thank you for contributing to Nexus!