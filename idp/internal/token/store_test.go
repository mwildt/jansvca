package token

import (
	"testing"
	"time"
)

func TestIssueAndConsumeCode(t *testing.T) {
	s := NewStore()
	code, err := s.IssueCode("app", "alice", "Alice", "http://localhost/cb", "openid")
	if err != nil {
		t.Fatalf("IssueCode: %v", err)
	}
	if code == "" {
		t.Fatal("empty code")
	}
	c, err := s.ConsumeCode(code)
	if err != nil {
		t.Fatalf("ConsumeCode: %v", err)
	}
	if c.ClientID != "app" || c.Subject != "alice" || c.RedirectURI != "http://localhost/cb" {
		t.Fatalf("unexpected code %+v", c)
	}
	// single-use
	if _, err := s.ConsumeCode(code); err == nil {
		t.Fatal("code should be single-use")
	}
}

func TestConsumeCodeExpired(t *testing.T) {
	s := NewStore()
	code, _ := s.IssueCode("app", "alice", "Alice", "http://localhost/cb", "")
	// force expiry by overwriting the entry directly.
	s.mu.Lock()
	s.codes[code].ExpiresAt = time.Now().Add(-time.Minute)
	s.mu.Unlock()
	if _, err := s.ConsumeCode(code); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestIssueAndReadToken(t *testing.T) {
	s := NewStore()
	tok, err := s.IssueToken("alice", "Alice", "openid profile")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	if tok.Token == "" || tok.Subject != "alice" || tok.Scope != "openid profile" {
		t.Fatalf("unexpected token %+v", tok)
	}
	got, err := s.Token(tok.Token)
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if got.Subject != "alice" || got.Name != "Alice" {
		t.Fatalf("unexpected stored token %+v", got)
	}
}

func TestTokenExpired(t *testing.T) {
	s := NewStore()
	tok, _ := s.IssueToken("alice", "Alice", "")
	s.mu.Lock()
	s.tokens[tok.Token].ExpiresAt = time.Now().Add(-time.Minute)
	s.mu.Unlock()
	if _, err := s.Token(tok.Token); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
