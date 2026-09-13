package session

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStoreCreateGetDelete(t *testing.T) {
	s := NewStore()
	sess := s.Create()
	if sess.ID == "" {
		t.Fatal("expected non-empty session id")
	}
	if got := s.Get(sess.ID); got == nil || got.ID != sess.ID {
		t.Fatalf("expected to retrieve session %q", sess.ID)
	}
	s.Delete(sess.ID)
	if got := s.Get(sess.ID); got != nil {
		t.Fatal("expected session to be deleted")
	}
}

func TestStoreExpiry(t *testing.T) {
	s := NewStore()
	sess := s.Create()
	sess.ExpiresAt = time.Now().Add(-time.Second)
	s.Save(sess)
	if got := s.Get(sess.ID); got != nil {
		t.Fatal("expected expired session to be evicted")
	}
}

func TestSessionIsAuthenticated(t *testing.T) {
	var s Session
	if s.IsAuthenticated() {
		t.Fatal("empty session should not be authenticated")
	}
	s.Token = "abc"
	s.ExpiresAt = time.Now().Add(-time.Second)
	if s.IsAuthenticated() {
		t.Fatal("expired token should not be authenticated")
	}
	s.ExpiresAt = time.Now().Add(time.Hour)
	if !s.IsAuthenticated() {
		t.Fatal("valid token should be authenticated")
	}
	s.ExpiresAt = time.Time{}
	if !s.IsAuthenticated() {
		t.Fatal("token without expiry should be authenticated")
	}
}

func TestManagerFromRequestNoCookie(t *testing.T) {
	m := NewManager(DefaultCookieConfig(false))
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if s := m.FromRequest(r); s != nil {
		t.Fatalf("expected nil session, got %v", s)
	}
}

func TestManagerSetAndClearCookie(t *testing.T) {
	m := NewManager(DefaultCookieConfig(false))
	sess := m.Store.Create()
	rec := httptest.NewRecorder()
	m.SetCookie(rec, sess)
	if cookies := rec.Result().Cookies(); len(cookies) == 0 {
		t.Fatal("expected Set-Cookie")
	}
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range rec.Result().Cookies() {
		r.AddCookie(c)
	}
	if got := m.FromRequest(r); got == nil || got.ID != sess.ID {
		t.Fatalf("expected to read session back, got %v", got)
	}
	rec2 := httptest.NewRecorder()
	m.ClearCookie(rec2, r)
	if len(rec2.Result().Cookies()) == 0 {
		t.Fatal("expected Clear-Cookie")
	}
	if got := m.FromRequest(httptest.NewRequest(http.MethodGet, "/", nil)); got != nil {
		t.Fatal("expected session cleared")
	}
}
