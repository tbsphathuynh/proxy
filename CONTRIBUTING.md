# Contributing to Proxy

We love your input! We want to make contributing to this project as easy and transparent as possible, whether it's:

- Reporting a bug
- Discussing the current state of the code
- Submitting a fix
- Proposing new features
- Becoming a maintainer

## Development Process

We use GitHub to host code, to track issues and feature requests, as well as accept pull requests.

### Code Standards

1. **Code Quality**: All code must be well-documented with complexity analysis
2. **Testing**: Every function must have comprehensive tests
3. **Design Patterns**: Use appropriate design patterns throughout
4. **Performance**: Consider time and space complexity for all implementations
5. **Documentation**: Include detailed comments explaining why code exists

### Pull Request Process

1. Fork the repo and create your branch from `master`
2. If you've added code that should be tested, add tests
3. If you've changed APIs, update the documentation
4. Ensure the test suite passes
5. Make sure your code lints
6. Issue that pull request!

### Commit Message Convention

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

Types:
- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Changes that do not affect the meaning of the code
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `perf`: A code change that improves performance
- `test`: Adding missing tests or correcting existing tests
- `chore`: Changes to the build process or auxiliary tools

Examples:
- `feat(cache): add LRU eviction policy`
- `fix(loadbalancer): handle empty backend list`
- `docs(readme): update installation instructions`

### Pre-commit Requirements

Before committing, ensure:

1. **Formatting**: `go fmt ./...`
2. **Linting**: `golangci-lint run`
3. **Testing**: `go test -race ./...`
4. **Coverage**: Maintain >80% test coverage
5. **Benchmarks**: Run `go test -bench=.` for performance-critical changes

### Development Setup

1. **Prerequisites**:
   ```bash
   go version # >= 1.22
   golangci-lint --version
   pre-commit --version
   ```

2. **Clone and setup**:
   ```bash
   git clone https://github.com/WillKirkmanM/proxy.git
   cd proxy
   go mod download
   pre-commit install
   ```

3. **Run tests**:
   ```bash
   make test
   make test-integration
   make benchmark
   ```

4. **Development workflow**:
   ```bash
   # Create feature branch
   git checkout -b feat/my-feature
   
   # Make changes with tests
   # ...
   
   # Run pre-commit checks
   pre-commit run --all-files
   
   # Commit with conventional message
   git commit -m "feat(feature): add awesome functionality"
   
   # Push and create PR
   git push origin feat/my-feature
   ```

### Architecture Guidelines

1. **Package Structure**:
   ```
   internal/
   ├── config/          # Configuration management
   ├── middleware/      # HTTP middleware components
   ├── loadbalancer/    # Load balancing algorithms
   ├── health/          # Health checking
   ├── logging/         # Structured logging
   └── tracing/         # OpenTelemetry integration
   ```

2. **Design Patterns**:
   - **Strategy Pattern**: For load balancing algorithms
   - **Decorator Pattern**: For middleware chaining
   - **Factory Pattern**: For component creation
   - **Observer Pattern**: For health monitoring

3. **Error Handling**:
   - Use structured errors with context
   - Log errors with trace correlation
   - Implement circuit breaker for resilience

4. **Performance Considerations**:
   - Document time/space complexity
   - Use sync.Pool for object reuse
   - Implement proper caching strategies
   - Profile critical paths

### Testing Standards

1. **Unit Tests**:
   - Test all public functions
   - Mock external dependencies
   - Use table-driven tests where appropriate
   - Include edge cases and error conditions

2. **Integration Tests**:
   - Test component interactions
   - Use real HTTP servers for proxy testing
   - Verify middleware chain behavior

3. **Benchmarks**:
   - Benchmark performance-critical code
   - Include memory allocation metrics
   - Compare against baseline performance

### Documentation

1. **Code Comments**:
   - Explain why, not what
   - Include complexity analysis
   - Document assumptions and constraints

2. **API Documentation**:
   - Use Go doc conventions
   - Include examples where helpful
   - Document thread safety guarantees

3. **README Updates**:
   - Keep installation instructions current
   - Update feature lists
   - Include configuration examples

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.

## Questions?

Feel free to open an issue or start a discussion if you have questions about contributing!