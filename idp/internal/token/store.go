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
func (s *Store) IssueToken(subject, name, scope string) (*AccessToken, string, error) {
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

// ConsumeRefreshToken validates a refresh token, removes it (single-use) and
// returns the principal it was issued for. The associated access token stays
// valid until its own expiry.
func (s *Store) ConsumeRefreshToken(refresh string) (*AccessToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	access, ok := s.refresh[refresh]
	if !ok {
		return nil, ErrNotFound
	}
	delete(s.refresh, refresh)
	t, ok := s.tokens[access]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *t
	return &cp, nil
}

func randomID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
