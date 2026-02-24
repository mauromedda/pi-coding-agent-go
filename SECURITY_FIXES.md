# Security Vulnerabilities and Fixes

## Critical Issues (Fix Immediately)

### 1. Command Injection (Multiple Files)

**Files affected:**
- `internal/git/worktree.go:143`
- `internal/tools/bash.go:106` 
- `internal/statusline/statusline.go:78`
- `internal/revert/revert.go:90`

**Problem:** User input passed directly to exec.Command without sanitization

**Fix:** Sanitize and validate all command arguments

```go
// Before (VULNERABLE)
cmd := exec.CommandContext(ctx, "git", args...)

// After (SECURE)
func sanitizeGitArgs(args []string) ([]string, error) {
    allowedCommands := map[string]bool{
        "status": true, "diff": true, "log": true, 
        "rev-parse": true, "ls-files": true,
    }
    
    if len(args) == 0 || !allowedCommands[args[0]] {
        return nil, fmt.Errorf("git command not allowed: %v", args)
    }
    
    // Sanitize each argument
    sanitized := make([]string, len(args))
    for i, arg := range args {
        if strings.Contains(arg, ";") || strings.Contains(arg, "|") || strings.Contains(arg, "&") {
            return nil, fmt.Errorf("invalid character in git argument: %s", arg)
        }
        sanitized[i] = arg
    }
    return sanitized, nil
}

// Usage
sanitizedArgs, err := sanitizeGitArgs(args)
if err != nil {
    return fmt.Errorf("invalid git command: %w", err)
}
cmd := exec.CommandContext(ctx, "git", sanitizedArgs...)
```

### 2. Path Traversal Vulnerabilities (G304, G703)

**Files affected:**
- `internal/memory/auto.go:58`
- `internal/memory/memory.go:175`
- `pkg/tui/theme/loader.go:57`
- `internal/ide/editor.go:38`

**Problem:** File paths not validated, allowing access to arbitrary files

**Fix:** Implement strict path validation

```go
// Add to internal/permission/sandbox.go
func ValidateReadPath(path string) error {
    // Resolve path to prevent symlink attacks
    resolved, err := filepath.Abs(path)
    if err != nil {
        return fmt.Errorf("invalid path: %w", err)
    }
    
    // Check for path traversal
    if strings.Contains(resolved, "..") {
        return fmt.Errorf("path traversal detected: %s", path)
    }
    
    // Restrict to allowed directories
    allowedPrefixes := []string{
        filepath.Join(os.Getenv("HOME"), ".pi-go"),
        filepath.Join(os.Getenv("HOME"), ".claude"),
        "/tmp/pi-go", // Only for temp files
    }
    
    allowed := false
    for _, prefix := range allowedPrefixes {
        if strings.HasPrefix(resolved, prefix) {
            allowed = true
            break
        }
    }
    
    if !allowed {
        return fmt.Errorf("access denied to path: %s", path)
    }
    
    return nil
}
```

### 3. Hardcoded Credentials (G101)

**Files affected:**
- `internal/mcp/oauth_test.go` (lines 278, 304, 344)

**Problem:** Test credentials could be mistaken for real ones

**Fix:** Use environment variables for all credentials

```go
// Replace hardcoded test credentials
cfg := OAuthConfig{
    ClientID: os.Getenv("TEST_CLIENT_ID"),
    AuthURL:  os.Getenv("TEST_AUTH_URL"), 
    TokenURL: os.Getenv("TEST_TOKEN_URL"),
    Scopes:   []string{"read"},
}

// Set defaults for tests
if cfg.ClientID == "" {
    cfg.ClientID = "test-client-id"
}
```

### 4. Insecure File Permissions (G306, G301, G302)

**Files affected:**
- `cmd/pi-go/update.go:240` (0755 for executable)
- `internal/memory/auto.go:27` (0755 for directory)
- Multiple test files using 0644

**Fix:** Use secure permissions

```go
// For executables (after verification)
if err := os.Chmod(tmpPath, 0755); err != nil { // This is OK for executables

// For config directories  
_ = os.MkdirAll(dir, 0700) // More restrictive

// For config files
os.WriteFile(path, data, 0600) // User-only access
```

### 5. SSRF Vulnerabilities (G704)

**Files affected:**
- `internal/export/gist.go:70`
- `internal/perf/probe.go:72`  
- `pkg/ai/provider/vertex/vertex.go:96`

**Problem:** HTTP requests to user-controlled URLs

**Fix:** URL validation and allowlist

```go
func validateURL(rawURL string) error {
    u, err := url.Parse(rawURL)
    if err != nil {
        return fmt.Errorf("invalid URL: %w", err)
    }
    
    // Only allow HTTPS
    if u.Scheme != "https" {
        return fmt.Errorf("only HTTPS URLs allowed")
    }
    
    // Allowlist domains
    allowedHosts := []string{
        "api.github.com",
        "api.anthropic.com", 
        "api.openai.com",
        "generativelanguage.googleapis.com",
    }
    
    allowed := false
    for _, host := range allowedHosts {
        if u.Host == host {
            allowed = true
            break
        }
    }
    
    if !allowed {
        return fmt.Errorf("host not allowed: %s", u.Host)
    }
    
    return nil
}
```

## High Priority Issues

### 6. Lack of Input Validation in Authentication

**File:** `internal/config/auth.go:125`

**Fix:** Validate shell commands for API key resolution

```go
func (a *AuthStore) resolveCommandKey(cmd string) (string, error) {
    // Validate command before execution
    if err := validateShellCommand(cmd); err != nil {
        return "", err
    }
    
    // Use restricted shell environment
    out, err := exec.Command("/bin/sh", "-c", cmd).Output()
    // ... rest of function
}

func validateShellCommand(cmd string) error {
    // Block dangerous commands
    dangerous := []string{"|", ";", "&", "`", "$", "rm", "curl", "wget"}
    cmdLower := strings.ToLower(cmd)
    
    for _, danger := range dangerous {
        if strings.Contains(cmdLower, danger) {
            return fmt.Errorf("command contains dangerous pattern: %s", danger)
        }
    }
    return nil
}
```

### 7. Insecure HTTP Server Configuration (G112)

**Files affected:**
- `internal/mcp/oauth.go:98`
- `internal/sandbox/network.go:48`

**Fix:** Add security headers and timeouts

```go
srv := &http.Server{
    Handler: mux,
    ReadHeaderTimeout: 10 * time.Second,
    ReadTimeout:       30 * time.Second,
    WriteTimeout:      30 * time.Second,
    IdleTimeout:       60 * time.Second,
    MaxHeaderBytes:    1 << 20, // 1MB
}
```

### 8. XSS Prevention (G203, G705)

**Files affected:**
- `internal/export/html.go:60`
- `internal/tools/find_references.go:145`

**Fix:** Proper HTML escaping

```go
// Use html.EscapeString instead of template.HTML
func escapeHTML(input string) string {
    escaped := html.EscapeString(input)
    return strings.ReplaceAll(escaped, "\n", "<br>\n")
}
```

## Medium Priority Issues

### 9. TLS Configuration (G402)

**File:** `internal/sandbox/network_test.go:46`

**Fix:** Remove InsecureSkipVerify in production code

```go
// Only for tests, add build constraint
//go:build test

TLSClientConfig: &tls.Config{InsecureSkipVerify: true}
```

### 10. Integer Overflow (G115)

**Files affected:** Multiple conversion warnings

**Fix:** Add bounds checking before conversions

```go
func safeIntToRune(n int) (rune, error) {
    if n < 0 || n > 0x10FFFF {
        return 0, fmt.Errorf("invalid rune value: %d", n)
    }
    return rune(n), nil
}
```

## Implementation Priority

1. **Immediate (Critical)**: Command injection, path traversal, SSRF
2. **This Week**: Authentication validation, secure permissions  
3. **Next Sprint**: HTTP server security, XSS prevention
4. **Ongoing**: Input validation, error handling improvements

## Testing Security Fixes

Add these security tests:

```go
func TestCommandInjectionPrevention(t *testing.T) {
    maliciousArgs := []string{"status", "; rm -rf /"}
    _, err := sanitizeGitArgs(maliciousArgs)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "invalid character")
}

func TestPathTraversalPrevention(t *testing.T) {
    maliciousPath := "../../etc/passwd"
    err := ValidateReadPath(maliciousPath)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "path traversal detected")
}
```