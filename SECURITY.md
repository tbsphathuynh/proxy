# Security Policy

## Supported Versions

We actively support the following versions with security updates:

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security vulnerability, please follow these steps:

### 1. Do NOT create a public GitHub issue

Security vulnerabilities should not be reported through public GitHub issues, discussions, or pull requests.

### 2. Send a private report

Send an email to [security@parson.dev](mailto:security@parson.dev) with:

- A clear description of the vulnerability
- Steps to reproduce the issue
- Potential impact assessment
- Any suggested fixes (if available)

### 3. Response Timeline

- **Acknowledgment**: We'll acknowledge receipt within 24 hours
- **Initial Assessment**: We'll provide an initial assessment within 72 hours
- **Progress Updates**: We'll send progress updates every 7 days
- **Resolution**: We aim to resolve critical vulnerabilities within 30 days

### 4. Disclosure Process

1. **Private Discussion**: We'll work with you to understand and validate the vulnerability
2. **Fix Development**: We'll develop and test a fix
3. **Coordinated Disclosure**: We'll coordinate public disclosure timing with you
4. **Credit**: We'll provide appropriate credit for responsible disclosure (if desired)

## Security Considerations

### Network Security

- **TLS Encryption**: Always use TLS in production environments
- **Certificate Validation**: Ensure proper certificate validation for backend connections
- **Network Isolation**: Deploy proxy in isolated network segments when possible

### Authentication & Authorization

- **Rate Limiting**: Configure appropriate rate limits to prevent abuse
- **Header Filtering**: Filter sensitive headers before forwarding requests
- **Access Logs**: Enable comprehensive access logging for security monitoring

### Configuration Security

- **Secrets Management**: Use environment variables or secure vaults for sensitive configuration
- **Least Privilege**: Run proxy with minimal required privileges
- **Regular Updates**: Keep proxy and dependencies updated

### Runtime Security

- **Resource Limits**: Configure appropriate memory and connection limits
- **Health Monitoring**: Implement health checks and monitoring
- **Circuit Breaker**: Use circuit breaker patterns to prevent cascade failures

## Security Features

### Built-in Protections

- **Request Size Limits**: Configurable limits on request body size
- **Connection Limits**: Maximum concurrent connection limits
- **Timeout Protection**: Configurable timeouts for all operations
- **Header Size Limits**: Protection against large header attacks

### Middleware Security

- **Rate Limiting**: Token bucket algorithm for request rate limiting
- **Request Validation**: Input validation and sanitization
- **Response Filtering**: Configurable response header filtering
- **Audit Logging**: Comprehensive audit trail with OpenTelemetry

### Monitoring & Alerting

- **Security Metrics**: Expose security-relevant metrics for monitoring
- **Trace Correlation**: Full request tracing for security incident investigation
- **Alert Integration**: Support for security alert integrations

## Best Practices

### Deployment

1. **Use Container Security**:
   ```dockerfile
   # Use non-root user
   USER 1000:1000
   
   # Use distroless base images
   FROM gcr.io/distroless/base-debian11
   ```

2. **Network Configuration**:
   ```yaml
   # Limit network exposure
   server:
     bindAddress: "127.0.0.1"  # Not 0.0.0.0 in production
   ```

3. **Resource Limits**:
   ```yaml
   # Configure appropriate limits
   server:
     maxConnections: 1000
     readTimeout: 30s
     writeTimeout: 30s
   ```

### Configuration

1. **Environment Variables**:
   ```bash
   # Use environment variables for secrets
   export PROXY_TLS_CERT_FILE=/path/to/cert.pem
   export PROXY_TLS_KEY_FILE=/path/to/key.pem
   ```

2. **File Permissions**:
   ```bash
   # Secure configuration files
   chmod 600 config.yaml
   chmod 600 /path/to/cert.pem
   chmod 600 /path/to/key.pem
   ```

### Monitoring

1. **Security Logs**:
   ```json
   {
     "level": "warn",
     "msg": "Rate limit exceeded",
     "client_ip": "192.168.1.100",
     "trace_id": "abc123"
   }
   ```

2. **Metrics to Monitor**:
   - Request rate per client IP
   - Error rate by status code
   - Response time percentiles
   - Connection count trends

## Vulnerability History

We will maintain a history of security vulnerabilities and their fixes:

- **CVE-YYYY-XXXXX**: [Description] - Fixed in version X.Y.Z

## Contact

For security-related questions or concerns:

- **Security Email**: [security@parson.dev](mailto:security@parson.dev)
- **PGP Key**: [Link to PGP key if available]
- **Response Time**: We aim to respond within 24 hours

## Acknowledgments

We thank the security research community for responsible disclosure of vulnerabilities. Contributors who follow our responsible disclosure process will be acknowledged in our security advisories (with their permission).