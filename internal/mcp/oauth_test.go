// ABOUTME: Tests for OAuth 2.0 authorization code flow with PKCE
// ABOUTME: Covers code verifier/challenge generation, full exchange, refresh, and timeout

package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestGenerateCodeVerifier(t *testing.T) {
	v := generateCodeVerifier()

	// RFC 7636: code_verifier must be 43-128 characters.
	if len(v) < 43 || len(v) > 128 {
		t.Errorf("code verifier length %d; want 43-128", len(v))
	}

	// Must only contain unreserved characters (base64url alphabet).
	validChars := regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	if !validChars.MatchString(v) {
		t.Errorf("code verifier contains invalid characters: %q", v)
	}

	// Two calls should produce different values (crypto/rand).
	v2 := generateCodeVerifier()
	if v == v2 {
		t.Error("two calls to generateCodeVerifier returned the same value")
	}
}

func TestGenerateCodeChallenge(t *testing.T) {
	// RFC 7636 Appendix B test vector:
	// verifier: "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	// expected S256 challenge: "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	expected := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	got := generateCodeChallenge(verifier)
	if got != expected {
		t.Errorf("generateCodeChallenge(%q) = %q; want %q", verifier, got, expected)
	}
}

func TestGenerateState(t *testing.T) {
	s := generateState()

	// 16 bytes hex encoded = 32 characters.
	if len(s) != 32 {
		t.Errorf("state length %d; want 32", len(s))
	}

	// Must be valid hex.
	validHex := regexp.MustCompile(`^[0-9a-f]+$`)
	if !validHex.MatchString(s) {
		t.Errorf("state contains non-hex characters: %q", s)
	}

	// Two calls should produce different values.
	s2 := generateState()
	if s == s2 {
		t.Error("two calls to generateState returned the same value")
	}
}

func TestOAuthFlow_FullExchange(t *testing.T) {
	// Track what the token endpoint receives.
	var receivedGrantType string
	var receivedCode string
	var receivedCodeVerifier string
	var receivedRedirectURI string
	var receivedClientID string

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/x-www-form-urlencoded") {
			t.Errorf("token request Content-Type = %q; want application/x-www-form-urlencoded", ct)
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}

		receivedGrantType = r.FormValue("grant_type")
		receivedCode = r.FormValue("code")
		receivedCodeVerifier = r.FormValue("code_verifier")
		receivedRedirectURI = r.FormValue("redirect_uri")
		receivedClientID = r.FormValue("client_id")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "test-access-token",
			"refresh_token": "test-refresh-token",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	}))
	defer tokenServer.Close()

	cfg := OAuthConfig{
		ClientID: "test-client-id",
		AuthURL:  "http://example.com/authorize", // Not actually used; we simulate the callback.
		TokenURL: tokenServer.URL,
		Scopes:   []string{"read", "write"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Override openBrowser to capture the auth URL and simulate the callback.
	var capturedAuthURL string
	origOpenBrowser := openBrowserFunc
	openBrowserFunc = func(u string) error {
		capturedAuthURL = u
		// Parse the auth URL to extract state and redirect_uri.
		parsed, err := url.Parse(u)
		if err != nil {
			return err
		}
		q := parsed.Query()
		state := q.Get("state")
		redirectURI := q.Get("redirect_uri")

		// Simulate the browser callback with the auth code.
		callbackURL := redirectURI + "?code=test-auth-code&state=" + state
		resp, err := http.Get(callbackURL) //nolint:gosec // test code
		if err != nil {
			return err
		}
		resp.Body.Close()
		return nil
	}
	defer func() { openBrowserFunc = origOpenBrowser }()

	token, err := OAuthFlow(ctx, cfg)
	if err != nil {
		t.Fatalf("OAuthFlow: %v", err)
	}

	// Verify the token was returned correctly.
	if token.AccessToken != "test-access-token" {
		t.Errorf("AccessToken = %q; want %q", token.AccessToken, "test-access-token")
	}
	if token.RefreshToken != "test-refresh-token" {
		t.Errorf("RefreshToken = %q; want %q", token.RefreshToken, "test-refresh-token")
	}
	if token.TokenType != "Bearer" {
		t.Errorf("TokenType = %q; want %q", token.TokenType, "Bearer")
	}
	if token.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should not be zero")
	}

	// Verify the auth URL was constructed correctly.
	parsed, err := url.Parse(capturedAuthURL)
	if err != nil {
		t.Fatalf("parsing captured auth URL: %v", err)
	}
	q := parsed.Query()
	if q.Get("client_id") != "test-client-id" {
		t.Errorf("auth URL client_id = %q; want %q", q.Get("client_id"), "test-client-id")
	}
	if q.Get("response_type") != "code" {
		t.Errorf("auth URL response_type = %q; want %q", q.Get("response_type"), "code")
	}
	if q.Get("scope") != "read write" {
		t.Errorf("auth URL scope = %q; want %q", q.Get("scope"), "read write")
	}
	if q.Get("code_challenge_method") != "S256" {
		t.Errorf("auth URL code_challenge_method = %q; want %q", q.Get("code_challenge_method"), "S256")
	}
	if q.Get("code_challenge") == "" {
		t.Error("auth URL missing code_challenge")
	}
	if q.Get("state") == "" {
		t.Error("auth URL missing state")
	}

	// Verify the code_challenge is valid base64url.
	challenge := q.Get("code_challenge")
	if _, err := base64.RawURLEncoding.DecodeString(challenge); err != nil {
		t.Errorf("code_challenge is not valid base64url: %v", err)
	}

	// Verify token endpoint received correct parameters.
	if receivedGrantType != "authorization_code" {
		t.Errorf("token grant_type = %q; want %q", receivedGrantType, "authorization_code")
	}
	if receivedCode != "test-auth-code" {
		t.Errorf("token code = %q; want %q", receivedCode, "test-auth-code")
	}
	if receivedCodeVerifier == "" {
		t.Error("token request missing code_verifier")
	}
	if receivedRedirectURI == "" {
		t.Error("token request missing redirect_uri")
	}
	if receivedClientID != "test-client-id" {
		t.Errorf("token client_id = %q; want %q", receivedClientID, "test-client-id")
	}
}

func TestRefreshToken(t *testing.T) {
	var receivedGrantType string
	var receivedRefreshToken string
	var receivedClientID string

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}

		receivedGrantType = r.FormValue("grant_type")
		receivedRefreshToken = r.FormValue("refresh_token")
		receivedClientID = r.FormValue("client_id")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "new-access-token",
			"refresh_token": "new-refresh-token",
			"token_type":    "Bearer",
			"expires_in":    7200,
		})
	}))
	defer tokenServer.Close()

	cfg := OAuthConfig{
		ClientID: "test-client-id",
		TokenURL: tokenServer.URL,
	}

	ctx := context.Background()
	token, err := RefreshToken(ctx, cfg, "old-refresh-token")
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}

	if token.AccessToken != "new-access-token" {
		t.Errorf("AccessToken = %q; want %q", token.AccessToken, "new-access-token")
	}
	if token.RefreshToken != "new-refresh-token" {
		t.Errorf("RefreshToken = %q; want %q", token.RefreshToken, "new-refresh-token")
	}
	if token.TokenType != "Bearer" {
		t.Errorf("TokenType = %q; want %q", token.TokenType, "Bearer")
	}

	if receivedGrantType != "refresh_token" {
		t.Errorf("grant_type = %q; want %q", receivedGrantType, "refresh_token")
	}
	if receivedRefreshToken != "old-refresh-token" {
		t.Errorf("refresh_token = %q; want %q", receivedRefreshToken, "old-refresh-token")
	}
	if receivedClientID != "test-client-id" {
		t.Errorf("client_id = %q; want %q", receivedClientID, "test-client-id")
	}
}

func TestOAuthFlow_Timeout(t *testing.T) {
	cfg := OAuthConfig{
		ClientID: "test-client-id",
		AuthURL:  "http://example.com/authorize",
		TokenURL: "http://example.com/token",
		Scopes:   []string{"read"},
	}

	// Use an already-cancelled context.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Override openBrowser to do nothing (no callback will come).
	origOpenBrowser := openBrowserFunc
	openBrowserFunc = func(_ string) error { return nil }
	defer func() { openBrowserFunc = origOpenBrowser }()

	_, err := OAuthFlow(ctx, cfg)
	if err == nil {
		t.Fatal("OAuthFlow with cancelled context should return error")
	}
	if !strings.Contains(err.Error(), "context") {
		t.Errorf("error should mention context; got %q", err)
	}
}

func TestOAuthFlow_CallbackStateMismatch(t *testing.T) {
	cfg := OAuthConfig{
		ClientID: "test-client-id",
		AuthURL:  "http://example.com/authorize",
		TokenURL: "http://example.com/token",
		Scopes:   []string{"read"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	origOpenBrowser := openBrowserFunc
	openBrowserFunc = func(u string) error {
		parsed, err := url.Parse(u)
		if err != nil {
			return err
		}
		q := parsed.Query()
		redirectURI := q.Get("redirect_uri")

		// Send callback with wrong state to trigger state mismatch.
		callbackURL := redirectURI + "?code=test-code&state=wrong-state"
		resp, err := http.Get(callbackURL) //nolint:gosec // test code
		if err != nil {
			return err
		}
		resp.Body.Close()
		return nil
	}
	defer func() { openBrowserFunc = origOpenBrowser }()

	_, err := OAuthFlow(ctx, cfg)
	if err == nil {
		t.Fatal("OAuthFlow should return error on state mismatch")
	}
	if !strings.Contains(err.Error(), "state mismatch") {
		t.Errorf("error should mention state mismatch; got %q", err)
	}
}

func TestOAuthFlow_CallbackAuthError(t *testing.T) {
	cfg := OAuthConfig{
		ClientID: "test-client-id",
		AuthURL:  "http://example.com/authorize",
		TokenURL: "http://example.com/token",
		Scopes:   []string{"read"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	origOpenBrowser := openBrowserFunc
	openBrowserFunc = func(u string) error {
		parsed, err := url.Parse(u)
		if err != nil {
			return err
		}
		q := parsed.Query()
		state := q.Get("state")
		redirectURI := q.Get("redirect_uri")

		// Simulate authorization server returning an error (user denied).
		callbackURL := redirectURI + "?error=access_denied&error_description=user+denied&state=" + state
		resp, err := http.Get(callbackURL) //nolint:gosec // test code
		if err != nil {
			return err
		}
		resp.Body.Close()
		return nil
	}
	defer func() { openBrowserFunc = origOpenBrowser }()

	_, err := OAuthFlow(ctx, cfg)
	if err == nil {
		t.Fatal("OAuthFlow should return error on access denied")
	}
	if !strings.Contains(err.Error(), "access_denied") {
		t.Errorf("error should mention access_denied; got %q", err)
	}
}

func TestOAuthFlow_CallbackMissingCode(t *testing.T) {
	cfg := OAuthConfig{
		ClientID: "test-client-id",
		AuthURL:  "http://example.com/authorize",
		TokenURL: "http://example.com/token",
		Scopes:   []string{"read"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	origOpenBrowser := openBrowserFunc
	openBrowserFunc = func(u string) error {
		parsed, err := url.Parse(u)
		if err != nil {
			return err
		}
		q := parsed.Query()
		state := q.Get("state")
		redirectURI := q.Get("redirect_uri")

		// Callback with correct state but no code parameter.
		callbackURL := redirectURI + "?state=" + state
		resp, err := http.Get(callbackURL) //nolint:gosec // test code
		if err != nil {
			return err
		}
		resp.Body.Close()
		return nil
	}
	defer func() { openBrowserFunc = origOpenBrowser }()

	_, err := OAuthFlow(ctx, cfg)
	if err == nil {
		t.Fatal("OAuthFlow should return error when code is missing")
	}
	if !strings.Contains(err.Error(), "missing code") {
		t.Errorf("error should mention missing code; got %q", err)
	}
}

func TestRefreshToken_ServerError(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_grant",
			"error_description": "refresh token expired",
		})
	}))
	defer tokenServer.Close()

	cfg := OAuthConfig{
		ClientID: "test-client-id",
		TokenURL: tokenServer.URL,
	}

	_, err := RefreshToken(context.Background(), cfg, "expired-token")
	if err == nil {
		t.Fatal("RefreshToken should return error on server error response")
	}
}
