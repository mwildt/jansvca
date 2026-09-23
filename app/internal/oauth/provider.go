// Package oauth implements the OAuth2 authorization-code flow used by the BFF.
// The browser is redirected to the provider's authorization endpoint; after the
// user consents, the provider calls back to the BFF with a code, which is
// exchanged for an access token stored server-side in the session.
//
// Only standard OAuth2 (RFC 6749) endpoints are used: authorization, token and
// introspection. The provider is fully configurable via Config.
package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Config describes the OAuth2 provider endpoints and client credentials.
type Config struct {
	// AuthorizationURL is the provider's authorization endpoint.
	AuthorizationURL string
	// TokenURL is the provider's token endpoint (code exchange).
	TokenURL string
	// IntrospectionURL is the provider's token introspection endpoint.
	IntrospectionURL string
	// RevocationURL is the provider's token revocation endpoint (RFC 7009).
	RevocationURL string
	// ClientID and ClientSecret identify this BFF at the provider.
	ClientID     string
	ClientSecret string
	// RedirectURL is the BFF's own callback URL, e.g. https://app/api/auth/callback.
	RedirectURL string
	// Scope requested from the provider (space separated).
	Scope string
}

// Provider wraps a Config and an HTTP client and performs the OAuth2 flows.
type Provider struct {
	cfg        Config
	httpClient *http.Client
}

// NewProvider creates a provider for the given config.
func NewProvider(cfg Config) *Provider {
	return &Provider{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// AuthURL builds the authorization-endpoint URL the browser should be sent to.
// state must be a random value the caller validates on callback. codeChallenge
// is the RFC 7636 PKCE challenge derived from the caller's verifier.
func (p *Provider) AuthURL(state, codeChallenge string) string {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", p.cfg.ClientID)
	q.Set("redirect_uri", p.cfg.RedirectURL)
	if p.cfg.Scope != "" {
		q.Set("scope", p.cfg.Scope)
	}
	q.Set("state", state)
	if codeChallenge != "" {
		q.Set("code_challenge", codeChallenge)
		q.Set("code_challenge_method", "S256")
	}
	return p.cfg.AuthorizationURL + "?" + q.Encode()
}

// NewCodeVerifier creates a random RFC 7636 PKCE code verifier.
func NewCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// CodeChallengeS256 derives the S256 code challenge for a verifier.
func CodeChallengeS256(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// Token is the result of the token exchange.
type Token struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in,omitempty"`
}

// Exchange performs the authorization-code -> access-token exchange.
// codeVerifier is the RFC 7636 PKCE verifier matching the challenge sent in
// the authorization request.
func (p *Provider) Exchange(ctx context.Context, code, codeVerifier string) (Token, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", p.cfg.RedirectURL)
	if codeVerifier != "" {
		form.Set("code_verifier", codeVerifier)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return Token{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(p.cfg.ClientID, p.cfg.ClientSecret)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return Token{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Token{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Token{}, &TokenError{StatusCode: resp.StatusCode, Body: string(body)}
	}
	var tok Token
	if err := json.Unmarshal(body, &tok); err != nil {
		return Token{}, err
	}
	if tok.AccessToken == "" {
		return Token{}, errors.New("oauth: token response missing access_token")
	}
	if tok.TokenType == "" {
		tok.TokenType = "Bearer"
	}
	return tok, nil
}

// TokenError describes a non-200 token-exchange response.
type TokenError struct {
	StatusCode int
	Body       string
}

func (e *TokenError) Error() string {
	return "oauth: token exchange failed: " + e.Body
}

// IntrospectionResult is the relevant subset of an introspection response.
type IntrospectionResult struct {
	Active   bool   `json:"active"`
	Sub      string `json:"sub"`
	Username string `json:"username"`
	Exp      int64  `json:"exp"`
}

// Introspect calls the introspection endpoint to validate and enrich a token.
func (p *Provider) Introspect(ctx context.Context, token string) (IntrospectionResult, error) {
	if p.cfg.IntrospectionURL == "" {
		return IntrospectionResult{}, errors.New("oauth: no introspection url configured")
	}
	form := url.Values{}
	form.Set("token", token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.IntrospectionURL, strings.NewReader(form.Encode()))
	if err != nil {
		return IntrospectionResult{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(p.cfg.ClientID, p.cfg.ClientSecret)
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return IntrospectionResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return IntrospectionResult{}, errors.New("oauth: introspection failed")
	}
	var ir IntrospectionResult
	if err := json.NewDecoder(resp.Body).Decode(&ir); err != nil {
		return IntrospectionResult{}, err
	}
	return ir, nil
}

// Refresh exchanges a refresh token for a new access token (RFC 6749 §6).
func (p *Provider) Refresh(ctx context.Context, refreshToken string) (Token, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return Token{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(p.cfg.ClientID, p.cfg.ClientSecret)
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return Token{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Token{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Token{}, &TokenError{StatusCode: resp.StatusCode, Body: string(body)}
	}
	var tok Token
	if err := json.Unmarshal(body, &tok); err != nil {
		return Token{}, err
	}
	if tok.AccessToken == "" {
		return Token{}, errors.New("oauth: refresh response missing access_token")
	}
	if tok.TokenType == "" {
		tok.TokenType = "Bearer"
	}
	if tok.RefreshToken == "" {
		tok.RefreshToken = refreshToken
	}
	return tok, nil
}

// Revoke invalidates an access token at the provider's revocation endpoint
// (RFC 7009). Revoking an already-invalid token is treated as success, per
// the RFC. Only transport-level failures are returned as errors.
func (p *Provider) Revoke(ctx context.Context, accessToken string) error {
	if p.cfg.RevocationURL == "" {
		return nil
	}
	form := url.Values{}
	form.Set("token", accessToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.RevocationURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(p.cfg.ClientID, p.cfg.ClientSecret)
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &TokenError{StatusCode: resp.StatusCode, Body: string(body)}
	}
	return nil
}

// ExpiresAt converts an expires_in seconds value into an absolute time.
func ExpiresAt(now time.Time, expiresIn int64) time.Time {
	if expiresIn <= 0 {
		return time.Time{}
	}
	return now.Add(time.Duration(expiresIn) * time.Second)
}
