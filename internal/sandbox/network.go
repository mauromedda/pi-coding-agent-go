// ABOUTME: HTTP CONNECT proxy that filters outbound connections by domain allowlist
// ABOUTME: Binds to localhost:0; use ProxyEnv() to inject HTTP_PROXY/HTTPS_PROXY into child processes

package sandbox

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
)

// maxConcurrentConns limits the number of simultaneous CONNECT tunnel goroutines.
const maxConcurrentConns = 100

// NetworkFilter is an HTTP proxy that only allows connections to allowlisted domains.
// It handles both plain HTTP requests and HTTPS CONNECT tunnels.
type NetworkFilter struct {
	allowedDomains map[string]bool
	listener       net.Listener
	server         *http.Server
	connSem        chan struct{} // semaphore limiting concurrent CONNECT tunnels

	mu   sync.Mutex
	addr string // filled after Start
}

// NewNetworkFilter creates a filter that permits only the given domains.
// Domain matching is case-insensitive and ignores port suffixes.
func NewNetworkFilter(allowedDomains []string) *NetworkFilter {
	m := make(map[string]bool, len(allowedDomains))
	for _, d := range allowedDomains {
		m[strings.ToLower(d)] = true
	}
	return &NetworkFilter{
		allowedDomains: m,
		connSem:        make(chan struct{}, maxConcurrentConns),
	}
}

// Start binds the proxy to localhost:0 and begins serving.
// Returns the listener address (host:port).
func (nf *NetworkFilter) Start() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("network filter listen: %w", err)
	}

	nf.mu.Lock()
	nf.listener = ln
	nf.addr = ln.Addr().String()
	nf.server = &http.Server{Handler: nf}
	nf.mu.Unlock()

	go func() { _ = nf.server.Serve(ln) }()

	return nf.addr, nil
}

// Stop gracefully shuts down the proxy server.
func (nf *NetworkFilter) Stop(ctx context.Context) error {
	nf.mu.Lock()
	srv := nf.server
	nf.mu.Unlock()

	if srv == nil {
		return nil
	}
	return srv.Shutdown(ctx)
}

// ProxyEnv returns environment variables that direct HTTP clients through this proxy.
func (nf *NetworkFilter) ProxyEnv() []string {
	nf.mu.Lock()
	addr := nf.addr
	nf.mu.Unlock()

	proxyURL := "http://" + addr
	return []string{
		"HTTP_PROXY=" + proxyURL,
		"HTTPS_PROXY=" + proxyURL,
	}
}

// ServeHTTP dispatches incoming proxy requests.
// CONNECT method = HTTPS tunnel; others = plain HTTP forwarding.
func (nf *NetworkFilter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		nf.handleConnect(w, r)
		return
	}
	nf.handleHTTP(w, r)
}

// handleConnect processes HTTPS CONNECT tunnel requests.
func (nf *NetworkFilter) handleConnect(w http.ResponseWriter, r *http.Request) {
	host := extractHost(r.Host)
	if !nf.isAllowed(host) {
		http.Error(w, "domain not allowed", http.StatusForbidden)
		return
	}

	// Acquire connection semaphore; reject if at capacity.
	select {
	case nf.connSem <- struct{}{}:
	default:
		http.Error(w, "too many concurrent connections", http.StatusServiceUnavailable)
		return
	}

	targetConn, err := net.DialTimeout("tcp", r.Host, 10*1e9) // 10s
	if err != nil {
		<-nf.connSem
		http.Error(w, "dial target: "+err.Error(), http.StatusBadGateway)
		return
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		<-nf.connSem
		targetConn.Close()
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}

	// Signal success before hijacking.
	w.WriteHeader(http.StatusOK)

	clientConn, _, err := hj.Hijack()
	if err != nil {
		<-nf.connSem
		targetConn.Close()
		return
	}

	// Each direction gets a goroutine; release semaphore when both complete.
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		transfer(targetConn, clientConn)
	}()
	go func() {
		defer wg.Done()
		transfer(clientConn, targetConn)
	}()
	go func() {
		wg.Wait()
		<-nf.connSem
	}()
}

// handleHTTP forwards plain HTTP requests after domain check.
func (nf *NetworkFilter) handleHTTP(w http.ResponseWriter, r *http.Request) {
	host := extractHost(r.Host)
	if !nf.isAllowed(host) {
		http.Error(w, "domain not allowed", http.StatusForbidden)
		return
	}

	// Build outbound request.
	outReq, err := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), r.Body)
	if err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	outReq.Header = r.Header.Clone()

	resp, err := http.DefaultTransport.RoundTrip(outReq)
	if err != nil {
		http.Error(w, "upstream error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers and body.
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// isAllowed checks whether the host is in the allowlist.
func (nf *NetworkFilter) isAllowed(host string) bool {
	return nf.allowedDomains[strings.ToLower(host)]
}

// extractHost strips the port from a host:port string.
func extractHost(hostport string) string {
	h, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport // No port present.
	}
	return h
}

// transfer copies data between two connections and closes dst when done.
func transfer(dst, src net.Conn) {
	defer dst.Close()
	_, _ = io.Copy(dst, src)
}
