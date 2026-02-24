// ABOUTME: URL security validation to prevent SSRF attacks
// ABOUTME: Validates URLs against allowlists and blocks dangerous schemes/hosts

package permission

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// URLValidator provides secure URL validation to prevent SSRF attacks
type URLValidator struct {
	allowedSchemes []string
	allowedHosts   []string
	blockedHosts   []string
	blockedCIDRs   []*net.IPNet
}

// NewURLValidator creates a new URL validator with security policies
func NewURLValidator() *URLValidator {
	// Parse blocked CIDR ranges for private networks
	blockedCIDRs := make([]*net.IPNet, 0)
	
	// Private IP ranges that should be blocked
	privateCIDRs := []string{
		"10.0.0.0/8",       // Private network
		"172.16.0.0/12",    // Private network  
		"192.168.0.0/16",   // Private network
		"127.0.0.0/8",      // Loopback
		"169.254.0.0/16",   // Link-local
		"224.0.0.0/4",      // Multicast
		"240.0.0.0/4",      // Reserved
		"0.0.0.0/8",        // Invalid
		"::1/128",          // IPv6 loopback
		"fe80::/10",        // IPv6 link-local
		"ff00::/8",         // IPv6 multicast
	}
	
	for _, cidr := range privateCIDRs {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err == nil {
			blockedCIDRs = append(blockedCIDRs, ipnet)
		}
	}

	return &URLValidator{
		allowedSchemes: []string{"https"}, // Only HTTPS by default
		allowedHosts: []string{
			"api.github.com",
			"github.com",
			"api.anthropic.com", 
			"api.openai.com",
			"generativelanguage.googleapis.com",
			"googleapis.com",
			"api.search.brave.com",
		},
		blockedHosts: []string{
			"localhost",
			"metadata.google.internal",
			"169.254.169.254", // AWS metadata
			"metadata.amazon.com",
		},
		blockedCIDRs: blockedCIDRs,
	}
}

// ValidateURL validates a URL for security issues
func (v *URLValidator) ValidateURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("empty URL")
	}

	// Parse URL
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Check scheme
	if err := v.validateScheme(parsed.Scheme); err != nil {
		return fmt.Errorf("invalid scheme: %w", err)
	}

	// Check host
	if err := v.validateHost(parsed.Host); err != nil {
		return fmt.Errorf("invalid host: %w", err)
	}

	// Check for dangerous URL patterns
	if err := v.checkDangerousPatterns(rawURL); err != nil {
		return fmt.Errorf("dangerous URL pattern: %w", err)
	}

	return nil
}

// validateScheme checks if the URL scheme is allowed
func (v *URLValidator) validateScheme(scheme string) error {
	if scheme == "" {
		return fmt.Errorf("missing scheme")
	}

	scheme = strings.ToLower(scheme)
	
	for _, allowed := range v.allowedSchemes {
		if scheme == allowed {
			return nil
		}
	}

	return fmt.Errorf("scheme %q not allowed (allowed: %v)", scheme, v.allowedSchemes)
}

// validateHost checks if the host is allowed and not blocked
func (v *URLValidator) validateHost(host string) error {
	if host == "" {
		return fmt.Errorf("missing host")
	}

	// Remove port if present
	hostname := host
	if colonPos := strings.LastIndex(host, ":"); colonPos > 0 {
		// Make sure it's a port, not IPv6
		if !strings.Contains(host[:colonPos], "]") {
			hostname = host[:colonPos]
		}
	}

	hostname = strings.ToLower(hostname)

	// Check against blocked hosts first
	for _, blocked := range v.blockedHosts {
		if hostname == blocked {
			return fmt.Errorf("host %q is blocked", hostname)
		}
	}

	// Check if it's an IP address and validate against blocked CIDRs
	if ip := net.ParseIP(hostname); ip != nil {
		for _, blockedNet := range v.blockedCIDRs {
			if blockedNet.Contains(ip) {
				return fmt.Errorf("IP address %s is in blocked range %s", hostname, blockedNet.String())
			}
		}
	} else {
		// For domain names, resolve and check IPs
		if err := v.validateDomainIPs(hostname); err != nil {
			return fmt.Errorf("domain IP validation failed: %w", err)
		}
	}

	// Check against allowed hosts
	for _, allowed := range v.allowedHosts {
		if hostname == allowed {
			return nil
		}
		
		// Allow subdomains of allowed hosts
		if strings.HasSuffix(hostname, "."+allowed) {
			return nil
		}
	}

	return fmt.Errorf("host %q not in allowlist", hostname)
}

// validateDomainIPs resolves domain and validates all resolved IPs
func (v *URLValidator) validateDomainIPs(domain string) error {
	ips, err := net.LookupIP(domain)
	if err != nil {
		// Don't fail on DNS errors, but log them
		return nil
	}

	for _, ip := range ips {
		for _, blockedNet := range v.blockedCIDRs {
			if blockedNet.Contains(ip) {
				return fmt.Errorf("domain %s resolves to blocked IP %s", domain, ip.String())
			}
		}
	}

	return nil
}

// checkDangerousPatterns looks for dangerous URL patterns
func (v *URLValidator) checkDangerousPatterns(rawURL string) error {
	urlLower := strings.ToLower(rawURL)
	
	// Check for URL encoding tricks
	dangerousPatterns := []string{
		"%00",           // Null byte
		"%0a", "%0d",    // Newlines
		"%2f%2f",        // Double slash
		"%2e%2e",        // Double dot
		"@",             // User info
		"\\",            // Backslash (Windows path)
		"file://",       // File scheme
		"ftp://",        // FTP scheme
		"gopher://",     // Gopher scheme
		"data:",         // Data scheme
		"javascript:",   // JavaScript scheme
		"vbscript:",     // VBScript scheme
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(urlLower, pattern) {
			return fmt.Errorf("contains dangerous pattern: %s", pattern)
		}
	}

	// Check for suspicious port numbers
	parsed, _ := url.Parse(rawURL)
	if port := parsed.Port(); port != "" {
		if err := v.validatePort(port); err != nil {
			return err
		}
	}

	return nil
}

// validatePort checks if a port number is safe
func (v *URLValidator) validatePort(port string) error {
	// Only allow common web ports
	allowedPorts := map[string]bool{
		"80":   true,  // HTTP
		"443":  true,  // HTTPS
		"8080": true,  // Alternative HTTP
		"8443": true,  // Alternative HTTPS
	}

	if !allowedPorts[port] {
		return fmt.Errorf("port %s not allowed", port)
	}

	return nil
}

// AddAllowedHost adds a host to the allowlist
func (v *URLValidator) AddAllowedHost(host string) {
	host = strings.ToLower(host)
	for _, existing := range v.allowedHosts {
		if existing == host {
			return // Already exists
		}
	}
	v.allowedHosts = append(v.allowedHosts, host)
}

// AddBlockedHost adds a host to the blocklist
func (v *URLValidator) AddBlockedHost(host string) {
	host = strings.ToLower(host)
	for _, existing := range v.blockedHosts {
		if existing == host {
			return // Already exists
		}
	}
	v.blockedHosts = append(v.blockedHosts, host)
}

// ValidateHTTPURL is a convenience method for validating HTTP URLs
func ValidateHTTPURL(rawURL string) error {
	validator := NewURLValidator()
	validator.allowedSchemes = []string{"http", "https"}
	return validator.ValidateURL(rawURL)
}

// ValidateAPIURL validates URLs specifically for API calls
func ValidateAPIURL(rawURL string) error {
	validator := NewURLValidator()
	// API calls should only use HTTPS
	validator.allowedSchemes = []string{"https"}
	
	// Add common API hosts
	apiHosts := []string{
		"api.github.com",
		"github.com", 
		"raw.githubusercontent.com",
		"api.anthropic.com",
		"api.openai.com",
		"generativelanguage.googleapis.com",
		"api.search.brave.com",
		"httpbin.org", // For testing
	}
	
	for _, host := range apiHosts {
		validator.AddAllowedHost(host)
	}
	
	return validator.ValidateURL(rawURL)
}

// ValidateWebhookURL validates URLs for webhook endpoints
func ValidateWebhookURL(rawURL string) error {
	validator := NewURLValidator()
	
	// Webhooks must use HTTPS
	validator.allowedSchemes = []string{"https"}
	
	// Be more restrictive for webhooks - only known safe domains
	validator.allowedHosts = []string{
		"hooks.slack.com",
		"discord.com",
		"api.telegram.org",
		"hooks.zapier.com",
	}
	
	return validator.ValidateURL(rawURL)
}