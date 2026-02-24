# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |

## Reporting a Vulnerability

Please report security vulnerabilities to the maintainers privately.

## Security Measures Implemented

### 1. Command Injection Prevention
- All git commands are validated and sanitized
- Bash commands are restricted to a safe allowlist
- Dangerous patterns are blocked

### 2. Path Traversal Protection
- All file paths are validated against allowed directories
- Symlink resolution prevents directory escapes
- Path traversal patterns are blocked

### 3. SSRF Prevention
- HTTP requests are validated against allowlists
- Private IP ranges are blocked
- Only HTTPS is allowed for external APIs

### 4. Input Validation
- All user inputs are sanitized and validated
- File uploads are restricted
- URL schemes are limited to safe protocols

### 5. Secure Defaults
- Restrictive file permissions (600/700)
- Limited environment variables for subprocesses
- Timeout configurations for all network operations

## Security Testing

Run security tests with:
```bash
go test ./internal/git -run "TestSanitize|TestValidate"
go test ./internal/tools -run "TestValidate|TestSanitize"
go test ./internal/permission -run "TestValidate"
```

## Security Hardening Checklist

- [ ] All dependencies are up to date
- [ ] File permissions are restrictive (600/700)
- [ ] Environment variables don't contain secrets
- [ ] HTTP clients have proper timeouts
- [ ] All user inputs are validated
- [ ] External API calls are restricted to allowlists