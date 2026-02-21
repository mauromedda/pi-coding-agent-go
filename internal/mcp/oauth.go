// ABOUTME: OAuth 2.0 authorization code flow with PKCE for MCP HTTP servers
// ABOUTME: Handles browser-based auth, token exchange, and token refresh

package mcp

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// OAuthConfig holds the configuration for an OAuth 2.0 authorization flow.
type OAuthConfig struct {
	ClientID    string
	AuthURL     string // Authorization endpoint.
	TokenURL    string // Token endpoint.
	RedirectURL string // Auto-set to localhost callback if empty.
	Scopes      []string
}

// OAuthToken represents the tokens returned by the authorization server.
type OAuthToken struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	TokenType    string // Usually "Bearer".
}

// openBrowserFunc is the function used to open URLs in the browser.
// It is a package-level variable so tests can override it.
var openBrowserFunc = openBrowser

// OAuthFlow performs the OAuth 2.0 authorization code flow with PKCE.
// It starts a local HTTP server for the callback, opens the browser to the
// authorization URL, waits for the callback with the auth code, and exchanges
// it for a token.
func OAuthFlow(ctx context.Context, cfg OAuthConfig) (*OAuthToken, error) {
	verifier := generateCodeVerifier()
	challenge := generateCodeChallenge(verifier)
	state := generateState()

	// Start a local HTTP server on a random port for the callback.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("starting callback listener: %w", err)
	}

	addr := listener.Addr().String()
	redirectURI := "http://" + addr + "/callback"

	type callbackResult struct {
		code string
		err  error
	}
	resultCh := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		// Validate state to prevent CSRF.
		if q.Get("state") != state {
			http.Error(w, "invalid state parameter", http.StatusBadRequest)
			resultCh <- callbackResult{err: fmt.Errorf("state mismatch: got %q", q.Get("state"))}
			return
		}

		if errParam := q.Get("error"); errParam != "" {
			desc := q.Get("error_description")
			http.Error(w, "Authorization failed: "+errParam, http.StatusBadRequest)
			resultCh <- callbackResult{err: fmt.Errorf("authorization error: %s: %s", errParam, desc)}
			return
		}

		code := q.Get("code")
		if code == "" {
			http.Error(w, "missing code parameter", http.StatusBadRequest)
			resultCh <- callbackResult{err: fmt.Errorf("callback missing code parameter")}
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body><h1>Authorization successful</h1><p>You can close this window.</p></body></html>")
		resultCh <- callbackResult{code: code}
	})

	srv := &http.Server{Handler: mux}
	go func() {
		if serveErr := srv.Serve(listener); serveErr != nil && serveErr != http.ErrServerClosed {
			resultCh <- callbackResult{err: fmt.Errorf("callback server: %w", serveErr)}
		}
	}()
	defer srv.Close()

	// Build the authorization URL.
	authURL, err := buildAuthURL(cfg, redirectURI, state, challenge)
	if err != nil {
		return nil, fmt.Errorf("building auth URL: %w", err)
	}

	// Open the browser.
	if err := openBrowserFunc(authURL); err != nil {
		return nil, fmt.Errorf("opening browser: %w", err)
	}

	// Wait for the callback or context cancellation.
	var code string
	select {
	case result := <-resultCh:
		if result.err != nil {
			return nil, result.err
		}
		code = result.code
	case <-ctx.Done():
		return nil, fmt.Errorf("waiting for authorization callback: %w", ctx.Err())
	}

	// Exchange the authorization code for a token.
	return exchangeCode(ctx, cfg, code, redirectURI, verifier)
}

// RefreshToken exchanges a refresh token for a new access token.
func RefreshToken(ctx context.Context, cfg OAuthConfig, refreshToken string) (*OAuthToken, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {cfg.ClientID},
	}

	return postTokenRequest(ctx, cfg.TokenURL, data)
}

// buildAuthURL constructs the full authorization URL with PKCE parameters.
func buildAuthURL(cfg OAuthConfig, redirectURI, state, challenge string) (string, error) {
	u, err := url.Parse(cfg.AuthURL)
	if err != nil {
		return "", fmt.Errorf("parsing auth URL %q: %w", cfg.AuthURL, err)
	}

	q := u.Query()
	q.Set("client_id", cfg.ClientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(cfg.Scopes, " "))
	q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// exchangeCode exchanges an authorization code for tokens.
func exchangeCode(ctx context.Context, cfg OAuthConfig, code, redirectURI, verifier string) (*OAuthToken, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {cfg.ClientID},
		"code_verifier": {verifier},
	}

	return postTokenRequest(ctx, cfg.TokenURL, data)
}

// tokenResponse is the JSON structure returned by the token endpoint.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// postTokenRequest sends a POST request to the token endpoint and parses the response.
func postTokenRequest(ctx context.Context, tokenURL string, data url.Values) (*OAuthToken, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, fmt.Errorf("decoding token response: %w", err)
	}

	if tr.Error != "" {
		return nil, fmt.Errorf("token error: %s: %s", tr.Error, tr.ErrorDesc)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned status %d", resp.StatusCode)
	}

	token := &OAuthToken{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		TokenType:    tr.TokenType,
	}
	if tr.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	}

	return token, nil
}

// generateCodeVerifier generates a PKCE code verifier using crypto/rand.
// It produces a 43-character base64url-encoded string (32 random bytes).
func generateCodeVerifier() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// generateCodeChallenge creates a PKCE S256 code challenge from a verifier.
// challenge = BASE64URL(SHA256(verifier))
func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// generateState generates a random state parameter for CSRF protection.
// It produces a 32-character hex string (16 random bytes).
func generateState() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// openBrowser opens a URL in the system's default browser.
func openBrowser(u string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", u).Start()
	case "linux":
		return exec.Command("xdg-open", u).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start()
	default:
		return fmt.Errorf("unsupported platform %q for opening browser", runtime.GOOS)
	}
}
