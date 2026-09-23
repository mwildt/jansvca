package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mwildt/jansvca/app/internal/gateway"
	"github.com/mwildt/jansvca/app/internal/oauth"
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
