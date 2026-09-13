package gateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mwildt/jansvca/app/internal/session"
)

// startBackend returns a test server that echoes the Authorization header.
func startBackend(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.Header.Get("Authorization"))
	}))
}

func newAuthManager(token string) *session.Manager {
	m := session.NewManager(session.DefaultCookieConfig(false))
	sess := m.Store.Create()
	sess.Token = token
	sess.ExpiresAt = time.Now().Add(time.Hour)
	m.Store.Save(sess)
	return m
}

// withSessionCookie copies the manager's single session cookie onto req.
func withSessionCookie(t *testing.T, mgr *session.Manager, token string, req *http.Request) {
	t.Helper()
	sess := mgr.Store.Create()
	sess.Token = token
	sess.ExpiresAt = time.Now().Add(time.Hour)
	mgr.Store.Save(sess)
	rec := httptest.NewRecorder()
	mgr.SetCookie(rec, sess)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}
}

func TestGatewayForwardsBearerToken(t *testing.T) {
	backend := startBackend(t)
	defer backend.Close()

	mgr := newAuthManager("test-token")
	gw, err := New([]Upstream{{Prefix: "/proxy/svc", Target: backend.URL}}, mgr)
	if err != nil {
		t.Fatalf("gateway: %v", err)
	}

	// Simulate a request carrying the session cookie.
	req := httptest.NewRequest(http.MethodGet, "/proxy/svc/x", nil)
	withSessionCookie(t, mgr, "test-token", req)
	w := httptest.NewRecorder()
	if !gw.ServeHTTP(w, req) {
		t.Fatal("expected gateway to handle the request")
	}
	if got := w.Body.String(); got != "Bearer test-token" {
		t.Fatalf("expected Bearer test-token, got %q", got)
	}
}

func TestGatewayNoMatchFallsThrough(t *testing.T) {
	backend := startBackend(t)
	defer backend.Close()
	mgr := newAuthManager("test-token")
	gw, _ := New([]Upstream{{Prefix: "/proxy/svc", Target: backend.URL}}, mgr)
	req := httptest.NewRequest(http.MethodGet, "/other", nil)
	w := httptest.NewRecorder()
	if gw.ServeHTTP(w, req) {
		t.Fatal("expected no match for /other")
	}
}

func TestGatewayStripPrefix(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.URL.Path)
	}))
	defer backend.Close()
	mgr := newAuthManager("test-token")
	gw, _ := New([]Upstream{{Prefix: "/proxy/svc", Target: backend.URL, StripPrefix: true}}, mgr)
	req := httptest.NewRequest(http.MethodGet, "/proxy/svc/items", nil)
	withSessionCookie(t, mgr, "test-token", req)
	w := httptest.NewRecorder()
	if !gw.ServeHTTP(w, req) {
		t.Fatal("expected gateway to handle the request")
	}
	if got := w.Body.String(); got != "/items" {
		t.Fatalf("expected /items, got %q", got)
	}
}

func TestNewNilManager(t *testing.T) {
	if _, err := New(nil, nil); err == nil {
		t.Fatal("expected error for nil manager")
	}
}
