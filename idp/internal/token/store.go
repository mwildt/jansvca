// Package token holds the in-memory authorization-code and access-token store
// for the IdP. Codes and tokens are random, short-lived, opaque strings.
package token

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"
)

// Code is an issued authorization code (RFC 6749 §4.1).
type Code struct {
	ClientID      string
	Subject       string
	Name          string
	RedirectURI   string
	Scope         string
	CodeChallenge string
	ExpiresAt     time.Time
}

// AccessToken is an issued access token with its principal.
type AccessToken struct {
	Token     string
	ClientID  string
	Subject   string
	Name      string
	Scope     string
	ExpiresAt time.Time
}

// Store keeps codes and tokens in memory. It is safe for concurrent use.
type Store struct {
	mu      sync.Mutex
	codes   map[string]*Code
	tokens  map[string]*AccessToken
	refresh map[string]string
}

// NewStore creates an empty store.
func NewStore() *Store {
	return &Store{codes: map[string]*Code{}, tokens: map[string]*AccessToken{}, refresh: map[string]string{}}
}

const (
	codeTTL  = 2 * time.Minute
	tokenTTL = 30 * time.Minute
)

// ErrNotFound is returned for unknown or expired codes/tokens.
var ErrNotFound = errors.New("token: not found")

// IssueCode creates, stores and returns a new single-use authorization code.
// codeChallenge is the RFC 7636 PKCE challenge (with method S256) and may be
// empty when the client does not use PKCE.
func (s *Store) IssueCode(clientID, subject, name, redirectURI, scope, codeChallenge string) (string, error) {
	id, err := randomID()
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	s.codes[id] = &Code{
		ClientID:      clientID,
		Subject:       subject,
		Name:          name,
		RedirectURI:   redirectURI,
		Scope:         scope,
		CodeChallenge: codeChallenge,
		ExpiresAt:     time.Now().Add(codeTTL),
	}
	s.mu.Unlock()
	return id, nil
}

// ConsumeCode returns and invalidates an authorization code (single-use).
func (s *Store) ConsumeCode(id string) (*Code, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.codes[id]
	if !ok || time.Now().After(c.ExpiresAt) {
		delete(s.codes, id)
		return nil, ErrNotFound
	}
	delete(s.codes, id)
	cp := *c
	return &cp, nil
}

// IssueToken creates, stores and returns a new access token plus refresh
// token. The refresh token is single-use: consuming it rotates both tokens.
// Both tokens are bound to the client that exchanged the code (clientID).
func (s *Store) IssueToken(clientID, subject, name, scope string) (*AccessToken, string, error) {
	id, err := randomID()
	if err != nil {
		return nil, "", err
	}
	refresh, err := randomID()
	if err != nil {
		return nil, "", err
	}
	t := &AccessToken{
		Token:     id,
		ClientID:  clientID,
		Subject:   subject,
		Name:      name,
		Scope:     scope,
		ExpiresAt: time.Now().Add(tokenTTL),
	}
	s.mu.Lock()
	s.tokens[id] = t
	s.refresh[refresh] = t.Token
	s.mu.Unlock()
	return t, refresh, nil
}

// Token returns the access token if present and not expired.
func (s *Store) Token(id string) (*AccessToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tokens[id]
	if !ok || time.Now().After(t.ExpiresAt) {
		delete(s.tokens, id)
		return nil, ErrNotFound
	}
	cp := *t
	return &cp, nil
}

// Revoke removes the access token and any refresh token pointing at it, but
// only when the token was issued to the given client (RFC 7009 §2.1: the
// authorization server validates whether the token was issued to the client
// making the request). Revoking a token of another client is reported as a
// no-op success. It reports whether a token was actually removed.
func (s *Store) Revoke(accessToken, clientID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tokens[accessToken]
	if !ok || t.ClientID != clientID {
		return false
	}
	delete(s.tokens, accessToken)
	for refresh, access := range s.refresh {
		if access == accessToken {
			delete(s.refresh, refresh)
		}
	}
	return true
}

// ConsumeRefreshToken validates a refresh token for the given client, removes
// it (single-use) and returns the principal it was issued for. The refresh
// token is rejected when it was issued to a different client. The associated
// access token stays valid until its own expiry.
func (s *Store) ConsumeRefreshToken(refresh, clientID string) (*AccessToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	access, ok := s.refresh[refresh]
	if !ok {
		return nil, ErrNotFound
	}
	t, ok := s.tokens[access]
	if !ok {
		delete(s.refresh, refresh)
		return nil, ErrNotFound
	}
	if t.ClientID != clientID {
		return nil, ErrNotFound
	}
	delete(s.refresh, refresh)
	cp := *t
	return &cp, nil
}

// cleanup removes expired codes and tokens. It is called by the janitor
// goroutine started via StartJanitor and can be called directly in tests.
func (s *Store) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for id, c := range s.codes {
		if now.After(c.ExpiresAt) {
			delete(s.codes, id)
		}
	}
	for id, t := range s.tokens {
		if now.After(t.ExpiresAt) {
			delete(s.tokens, id)
			for refresh, access := range s.refresh {
				if access == id {
					delete(s.refresh, refresh)
				}
			}
		}
	}
}

// StartJanitor starts a background goroutine that periodically removes
// expired codes and tokens (and refresh tokens pointing at them) so the
// in-memory store does not grow without bound. It stops when stop is closed.
func (s *Store) StartJanitor(interval time.Duration, stop <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.cleanup()
			case <-stop:
				return
			}
		}
	}()
}

func randomID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
