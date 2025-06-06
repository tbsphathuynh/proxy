
<p align="center">
  <img src="https://avatars.githubusercontent.com/u/138057124?s=200&v=4" width="150" />
</p>
<h1 align="center">Proxy</h1>

<p align="center">
  
</p>

<h4 align="center">
  <a href="#get-started">Features</a>
  ·
  <a href="https://docs.parsonlabs.com/">Architecture</a>
  ·
  <a href="https://github.com/WillKirkmanM/music/releases">Quick Start</a>
  ·
  <a href="https://github.com/WillKirkmanM/music/releases">Configuration</a>
  ·
  <a href="https://github.com/WillKirkmanM/music/releases">Testing</a>
  ·
  <a href="https://github.com/WillKirkmanM/music/releases">Performance</a>
  ·
  <a href="https://github.com/WillKirkmanM/music/releases">Contributing</a>
</h4>

<p align="center">A high-performance, extensible reverse proxy server written in Go. Designed with clean architectural patterns and robust concurrency primitives, it supports dynamic load balancing, fault tolerance, and performance-critical features. Core components are decoupled using interfaces and Go idioms, enabling modularity and testability.</p>

## Features

- **High Performance**: Efficient request routing with O(log n) lookup time
- **Load Balancing**: Multiple algorithms (Round Robin, Least Connections, Weighted Round Robin)
- **Health Checks**: Automatic backend health monitoring
- **Circuit Breaker**: Fault tolerance with exponential backoff
- **Rate Limiting**: Token bucket algorithm for request throttling
- **Caching**: LRU cache for response optimisation
- **Metrics**: Comprehensive monitoring and statistics
- **Configuration**: YAML-based configuration management

## Architecture

The proxy follows several design patterns:

- **Strategy Pattern**: Pluggable load balancing algorithms
- **Observer Pattern**: Health check notifications
- **Circuit Breaker Pattern**: Fault tolerance
- **Factory Pattern**: Backend creation
- **Singleton Pattern**: Configuration management

## Quick Start

```bash
# Build the project
go build -o proxy cmd/proxy/main.go

# Run with default configuration
./proxy

# Run with custom configuration
./proxy -config config.yaml
```

## Configuration

```yaml
server:
  port: 8080
  timeout: 30s

backends:
  - url: "http://localhost:3001"
    weight: 1
  - url: "http://localhost:3002"
    weight: 2

load_balancer:
  algorithm: "round_robin" # round_robin, least_connections, weighted_round_robin

health_check:
  interval: 30s
  timeout: 10s
  path: "/health"

rate_limit:
  requests_per_second: 100
  burst: 10

cache:
  max_size: 1000
  ttl: 300s
```

## Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./...
```

## Performance

- **Throughput**: 10,000+ requests/second
- **Latency**: Sub-millisecond routing overhead
- **Memory**: Efficient O(1) backend selection
- **Scalability**: Horizontal scaling support

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add comprehensive tests
4. Ensure all tests pass
5. Submit a pull request

## License

MIT License