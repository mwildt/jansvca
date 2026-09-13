package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
