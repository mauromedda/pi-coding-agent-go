// ABOUTME: Shared HTTP client with retry logic and SSE streaming support
// ABOUTME: Provides exponential backoff on 429/5xx; respects HTTP_PROXY/HTTPS_PROXY

package httputil

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"time"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai/internal/sse"
)

const (
	maxRetries     = 3
	baseBackoffMs  = 500
	maxBackoffMs   = 10000
)

// Client wraps an http.Client with retry logic and default headers.
type Client struct {
	httpClient *http.Client
	baseURL    string
	headers    map[string]string
}

// NewClient creates a new HTTP client with the given base URL and default headers.
// Proxy support comes from the stdlib's default transport (HTTP_PROXY, HTTPS_PROXY).
func NewClient(baseURL string, headers map[string]string) *Client {
	if headers == nil {
		headers = make(map[string]string)
	}
	return &Client{
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 30 * time.Second,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
			},
		},
		baseURL: baseURL,
		headers: headers,
	}
}

// BaseURL returns the base URL configured on this client.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// Do sends an HTTP request with retry on 429 and 5xx status codes.
// It returns the response from the last attempt, even if retries were exhausted.
// If body implements io.Seeker, it is rewound before each retry attempt.
func (c *Client) Do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	seeker, _ := body.(io.Seeker)
	var lastResp *http.Response

	for attempt := range maxRetries {
		if err := rewindBody(seeker, attempt); err != nil {
			return nil, fmt.Errorf("failed to rewind request body: %w", err)
		}

		req, err := c.buildRequest(ctx, method, path, body)
		if err != nil {
			return nil, err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("http request failed: %w", err)
		}

		if !isRetryable(resp.StatusCode) {
			return resp, nil
		}

		// Close the body of the retryable response before retrying.
		resp.Body.Close()
		lastResp = resp

		if attempt < maxRetries-1 {
			if err := sleepWithContext(ctx, backoff(attempt)); err != nil {
				return nil, fmt.Errorf("context cancelled during retry backoff: %w", err)
			}
		}
	}

	// Retries exhausted: make one final request to return a readable response.
	if err := rewindBody(seeker, maxRetries); err != nil {
		return lastResp, fmt.Errorf("failed to rewind request body: %w", err)
	}

	req, err := c.buildRequest(ctx, method, path, body)
	if err != nil {
		return lastResp, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed after retries: %w", err)
	}

	return resp, nil
}

// StreamSSE sends an HTTP request and returns an SSE reader for the response body.
// The caller must close the returned *http.Response when done.
func (c *Client) StreamSSE(
	ctx context.Context,
	method, path string,
	body io.Reader,
) (*sse.Reader, *http.Response, error) {
	resp, err := c.Do(ctx, method, path, body)
	if err != nil {
		return nil, nil, fmt.Errorf("SSE stream request failed: %w", err)
	}

	return sse.NewReader(resp.Body), resp, nil
}

// buildRequest creates an http.Request with default headers applied.
func (c *Client) buildRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s %s: %w", method, path, err)
	}

	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	return req, nil
}

// rewindBody resets a seekable body to the beginning for retry attempts.
// It is a no-op on the first attempt (attempt == 0) or if seeker is nil.
func rewindBody(seeker io.Seeker, attempt int) error {
	if seeker == nil || attempt == 0 {
		return nil
	}
	_, err := seeker.Seek(0, io.SeekStart)
	return err
}

// isRetryable returns true for status codes that warrant a retry.
func isRetryable(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= 500
}

// backoff returns the backoff duration for the given attempt using exponential backoff.
func backoff(attempt int) time.Duration {
	ms := float64(baseBackoffMs) * math.Pow(2, float64(attempt))
	if ms > maxBackoffMs {
		ms = maxBackoffMs
	}
	return time.Duration(ms) * time.Millisecond
}

// sleepWithContext waits for the given duration or until the context is cancelled.
func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
