package token

import (
	"testing"
	"time"
)

func TestIssueAndConsumeCode(t *testing.T) {
	s := NewStore()
	code, err := s.IssueCode("app", "alice", "Alice", "http://localhost/cb", "openid", "")
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
	code, _ := s.IssueCode("app", "alice", "Alice", "http://localhost/cb", "", "")
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
	tok, refresh, err := s.IssueToken("app", "alice", "Alice", "openid profile")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	if refresh == "" {
		t.Fatal("expected non-empty refresh token")
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
	tok, _, _ := s.IssueToken("app", "alice", "Alice", "")
	s.mu.Lock()
	s.tokens[tok.Token].ExpiresAt = time.Now().Add(-time.Minute)
	s.mu.Unlock()
	if _, err := s.Token(tok.Token); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestRefreshTokenRotation(t *testing.T) {
	s := NewStore()
	tok, refresh, err := s.IssueToken("app", "alice", "Alice", "openid")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	old, err := s.ConsumeRefreshToken(refresh, "app")
	if err != nil {
		t.Fatalf("ConsumeRefreshToken: %v", err)
	}
	if old.Subject != "alice" || old.Token != tok.Token {
		t.Fatalf("unexpected token for refresh: %+v", old)
	}
	// refresh token is single-use
	if _, err := s.ConsumeRefreshToken(refresh, "app"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound on reuse, got %v", err)
	}
}

func TestRevoke(t *testing.T) {
	s := NewStore()
	tok, refresh, _ := s.IssueToken("app", "alice", "Alice", "openid")
	if !s.Revoke(tok.Token, "app") {
		t.Fatal("expected revoke to remove the token")
	}
	if _, err := s.Token(tok.Token); err != ErrNotFound {
		t.Fatalf("expected token gone, got %v", err)
	}
	// the refresh token pointing at it is gone too
	if _, err := s.ConsumeRefreshToken(refresh, "app"); err != ErrNotFound {
		t.Fatalf("expected refresh token gone, got %v", err)
	}
	// revoking an unknown token is a no-op success
	if s.Revoke("unknown", "app") {
		t.Fatal("expected false for unknown token")
	}
}

func TestRefreshTokenClientBinding(t *testing.T) {
	s := NewStore()
	tok, refresh, _ := s.IssueToken("app", "alice", "Alice", "openid")
	if tok.ClientID != "app" {
		t.Fatalf("token ClientID=%q, want app", tok.ClientID)
	}
	// a different client must not be able to use the refresh token.
	if _, err := s.ConsumeRefreshToken(refresh, "other-app"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound for other client, got %v", err)
	}
	// the token itself must not be revocable by another client.
	if s.Revoke(tok.Token, "other-app") {
		t.Fatal("revoke by other client must be a no-op")
	}
	// the rightful client still can.
	if _, err := s.ConsumeRefreshToken(refresh, "app"); err != nil {
		t.Fatalf("rightful client refresh: %v", err)
	}
}

func TestRevokeClientBinding(t *testing.T) {
	s := NewStore()
	tok, _, _ := s.IssueToken("app", "alice", "Alice", "openid")
	if s.Revoke(tok.Token, "other-app") {
		t.Fatal("revoke by other client must be a no-op")
	}
	if _, err := s.Token(tok.Token); err != nil {
		t.Fatalf("token must still be valid after foreign revoke attempt: %v", err)
	}
	if !s.Revoke(tok.Token, "app") {
		t.Fatal("expected revoke by owning client to remove the token")
	}
	if _, err := s.Token(tok.Token); err != ErrNotFound {
		t.Fatalf("expected token gone, got %v", err)
	}
}

func TestCleanupRemovesExpired(t *testing.T) {
	s := NewStore()
	code, _ := s.IssueCode("app", "alice", "Alice", "http://localhost/cb", "", "")
	tok, refresh, _ := s.IssueToken("app", "alice", "Alice", "")

	// force expiry.
	s.mu.Lock()
	s.codes[code].ExpiresAt = time.Now().Add(-time.Minute)
	s.tokens[tok.Token].ExpiresAt = time.Now().Add(-time.Minute)
	s.mu.Unlock()

	s.cleanup()

	if _, err := s.ConsumeCode(code); err != ErrNotFound {
		t.Fatalf("expected expired code removed, got %v", err)
	}
	if _, err := s.Token(tok.Token); err != ErrNotFound {
		t.Fatalf("expected expired token removed, got %v", err)
	}
	if _, err := s.ConsumeRefreshToken(refresh, "app"); err != ErrNotFound {
		t.Fatalf("expected refresh token of expired token removed, got %v", err)
	}
	// a refresh token is removed with its access token even if the refresh
	// map still referenced the deleted access token.
	if len(s.refresh) != 0 {
		t.Fatalf("refresh map not empty: %d", len(s.refresh))
	}
}

func TestStartJanitorStopsOnClose(t *testing.T) {
	s := NewStore()
	stop := make(chan struct{})
	s.StartJanitor(time.Millisecond, stop)
	// let it tick at least once without hanging the test on failure.
	time.Sleep(5 * time.Millisecond)
	close(stop)
	// must not panic or leak; a second close would panic, so just return.
}
