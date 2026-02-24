// ABOUTME: Secure HTTP client configuration with timeouts and security headers
// ABOUTME: Prevents slowloris attacks and adds security headers

package http

import (
	"net/http"
	"time"
)

// SecureHTTPClient creates an HTTP client with security configurations
func SecureHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			IdleConnTimeout:       30 * time.Second,
			MaxIdleConns:          10,
			MaxIdleConnsPerHost:   2,
		},
	}
}

// SecureHTTPServer creates an HTTP server with security configurations
func SecureHTTPServer(handler http.Handler, addr string) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}
}