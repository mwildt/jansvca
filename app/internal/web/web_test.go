package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mwildt/jansvca/app/internal/gateway"
	"github.com/mwildt/jansvca/app/internal/oauth"
	"github.com/mwildt/jansvca/app/internal/session"
)

// TestSPA serves index.html for unknown routes and static files for known ones.
func TestSPAServing(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>SPA</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "favicon.svg"), []byte("<svg/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	app, err := New(Config{SPADir: dir})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/some/unknown/route")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "SPA") {
		t.Fatalf("expected index.html fallback, got %q", body)
	}

	resp2, err := http.Get(srv.URL + "/favicon.svg")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for favicon, got %d", resp2.StatusCode)
	}
}

// TestBackendProxy forwards /api/* to the backend and injects no token when
// OAuth is disabled.
func TestBackendProxyNoAuth(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.URL.Path)
	}))
	defer backend.Close()

	app, err := New(Config{BackendURL: backend.URL})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/projects")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "/api/projects" {
		t.Fatalf("expected /api/projects proxied verbatim, got %q", body)
	}
}

// TestBackendProxyMissingBackend returns 502 when no backend is configured.
func TestBackendProxyMissingBackend(t *testing.T) {
	app, err := New(Config{})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/projects")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", resp.StatusCode)
	}
}

// TestAuthUserEndpoint reports unauthenticated when no session exists.
func TestAuthUserEndpoint(t *testing.T) {
	app, err := New(Config{})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/auth/user")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "\"authenticated\":false") {
		t.Fatalf("expected unauthenticated user, got %q", body)
	}
}

// TestGatewayRequiresAuth verifies that gateway routes are not reachable
// without an authenticated session when OAuth is enabled.
func TestGatewayRequiresAuth(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "upstream")
	}))
	defer upstream.Close()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>SPA</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	app, err := New(Config{
		SPADir: dir,
		OAuth: oauth.Config{
			AuthorizationURL: "https://idp.example.com/authorize",
			TokenURL:         "https://idp.example.com/token",
			ClientID:         "jansvca-app",
			ClientSecret:     "secret",
			RedirectURL:      "http://localhost:8080/api/auth/callback",
		},
		Upstreams: []gateway.Upstream{{Prefix: "/proxy/svc", Target: upstream.URL}},
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/proxy/svc/data")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated gateway request, got %d", resp.StatusCode)
	}

	resp2, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	body, _ := io.ReadAll(resp2.Body)
	if !strings.Contains(string(body), "SPA") {
		t.Fatalf("expected SPA to stay reachable unauthenticated, got %q", body)
	}
}

// TestGatewayDisabledAuthProxies verifies that gateway routes stay open (dev
// mode) when OAuth is not configured.
func TestGatewayDisabledAuthProxies(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "upstream")
	}))
	defer upstream.Close()

	app, err := New(Config{
		Upstreams: []gateway.Upstream{{Prefix: "/proxy/svc", Target: upstream.URL}},
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/proxy/svc/data")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "upstream" {
		t.Fatalf("expected proxied response, got %q", body)
	}
}

// TestLogoutPostOnly verifies that logout is rejected via GET (CSRF guard)
// and clears the session via POST.
func TestLogoutPostOnly(t *testing.T) {
	revoked := ""
	idp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/revoke" {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		_ = r.ParseForm()
		revoked = r.FormValue("token")
		w.WriteHeader(http.StatusOK)
	}))
	defer idp.Close()

	app, err := New(Config{
		OAuth: oauth.Config{
			AuthorizationURL: "https://idp.example.com/authorize",
			TokenURL:         "https://idp.example.com/token",
			RevocationURL:    idp.URL + "/revoke",
			ClientID:         "jansvca-app",
			ClientSecret:     "secret",
			RedirectURL:      "http://localhost:8080/api/auth/callback",
		},
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	ts := httptest.NewServer(app.Handler())
	defer ts.Close()

	// GET logout -> 405.
	resp, err := client.Get(ts.URL + "/api/auth/logout")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for GET logout, got %d", resp.StatusCode)
	}

	// Seed an authenticated session directly in the store.
	sess := app.mgr.Store.Create()
	sess.Token = "tok-123"
	sess.ExpiresAt = time.Now().Add(time.Hour)
	app.mgr.Store.Save(sess)

	// POST logout with the session cookie -> 302 and revocation called.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: sess.ID})
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302 for POST logout, got %d", resp.StatusCode)
	}
	if revoked != "tok-123" {
		t.Fatalf("expected token to be revoked, got %q", revoked)
	}
	if s := app.mgr.Store.Get(sess.ID); s != nil {
		t.Fatal("expected session to be deleted after logout")
	}
}
