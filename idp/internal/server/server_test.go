package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mwildt/jansvca/idp/internal/config"
)

func newTestStore(t *testing.T) *config.Store {
	t.Helper()
	s, err := config.LoadString(`
clients:
  - id: app
    secret: s3cret
    redirect_uris:
      - http://localhost:8080/api/auth/callback
users:
  - subject: admin
    name: Administrator
    password: admin
`)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return s
}

func TestAuthorizeRendersLoginForm(t *testing.T) {
	srv := New(newTestStore(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/authorize?response_type=code&client_id=app&redirect_uri=http://localhost:8080/api/auth/callback&state=xyz", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Anmeldung") {
		t.Fatalf("login form missing: %s", rec.Body.String())
	}
}

func TestAuthorizeInvalidClient(t *testing.T) {
	srv := New(newTestStore(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/authorize?response_type=code&client_id=nope&redirect_uri=http://localhost:8080/api/auth/callback", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAuthorizeBadRedirectURI(t *testing.T) {
	srv := New(newTestStore(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/authorize?response_type=code&client_id=app&redirect_uri=http://evil.example/cb", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestFullCodeFlow(t *testing.T) {
	srv := New(newTestStore(t))
	h := srv.Handler()

	// 1) POST login form -> expect redirect with code.
	form := url.Values{}
	form.Set("username", "admin")
	form.Set("password", "admin")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/authorize?response_type=code&client_id=app&redirect_uri=http://localhost:8080/api/auth/callback&state=xyz&scope=openid",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse location: %v", err)
	}
	code := u.Query().Get("code")
	if code == "" {
		t.Fatal("missing code in redirect")
	}
	if u.Query().Get("state") != "xyz" {
		t.Fatalf("state mismatch: %q", u.Query().Get("state"))
	}
	if u.String()[:len("http://localhost:8080/api/auth/callback")] != "http://localhost:8080/api/auth/callback" {
		t.Fatalf("unexpected redirect host: %s", u.String())
	}

	// 2) Exchange code for token.
	form = url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", "http://localhost:8080/api/auth/callback")
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("app", "s3cret")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("token exchange: %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"access_token"`) {
		t.Fatalf("no access_token: %s", rec.Body.String())
	}

	// extract access_token naively
	parts := strings.Split(rec.Body.String(), "\"access_token\":\"")
	if len(parts) < 2 {
		t.Fatal("cannot extract access_token")
	}
	accessToken := strings.Split(parts[1], "\"")[0]

	// 3) Introspect.
	form = url.Values{}
	form.Set("token", accessToken)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/introspect", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("introspect: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"active":true`) {
		t.Fatalf("expected active token: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"sub":"admin"`) {
		t.Fatalf("expected sub admin: %s", rec.Body.String())
	}

	// 4) Code is single-use.
	form = url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", "http://localhost:8080/api/auth/callback")
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.SetBasicAuth("app", "s3cret")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("reused code should fail: %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTokenInvalidClient(t *testing.T) {
	srv := New(newTestStore(t))
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", "whatever")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.SetBasicAuth("wrong", "creds")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestIntrospectUnknownToken(t *testing.T) {
	srv := New(newTestStore(t))
	form := url.Values{}
	form.Set("token", "unknown")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/introspect", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"active":false`) {
		t.Fatalf("expected inactive: %s", rec.Body.String())
	}
}

func TestTokenFormClientAuth(t *testing.T) {
	// client_id/client_secret in form body (no Basic auth) must also work.
	srv := New(newTestStore(t))
	h := srv.Handler()

	form := url.Values{}
	form.Set("username", "admin")
	form.Set("password", "admin")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/authorize?response_type=code&client_id=app&redirect_uri=http://localhost:8080/api/auth/callback",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	code := getQuery(t, rec.Header().Get("Location"), "code")

	form = url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", "http://localhost:8080/api/auth/callback")
	form.Set("client_id", "app")
	form.Set("client_secret", "s3cret")
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("form client auth: %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAuthorizeSetsLastUserCookie(t *testing.T) {
	srv := New(newTestStore(t))
	h := srv.Handler()

	form := url.Values{}
	form.Set("username", "admin")
	form.Set("password", "admin")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/authorize?response_type=code&client_id=app&redirect_uri=http://localhost:8080/api/auth/callback&state=xyz",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)

	var cookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "jansvca_lastuser" {
			cookie = c
			break
		}
	}
	if cookie == nil {
		t.Fatalf("jansvca_lastuser cookie not set; cookies=%v", rec.Result().Cookies())
	}
	if cookie.Value != "admin" {
		t.Fatalf("cookie value=%q, want admin", cookie.Value)
	}
	if cookie.MaxAge <= 0 {
		t.Fatalf("cookie MaxAge=%d, want long-lived", cookie.MaxAge)
	}
}

func TestAuthorizePrefillsUsernameFromCookie(t *testing.T) {
	srv := New(newTestStore(t))
	h := srv.Handler()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/authorize?response_type=code&client_id=app&redirect_uri=http://localhost:8080/api/auth/callback&state=xyz", nil)
	req.AddCookie(&http.Cookie{Name: "jansvca_lastuser", Value: "admin"})
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `value="admin"`) {
		t.Fatalf("username not prefilled from cookie: %s", rec.Body.String())
	}
}

func TestAuthorizeNoCookieDoesNotPrefill(t *testing.T) {
	srv := New(newTestStore(t))
	h := srv.Handler()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/authorize?response_type=code&client_id=app&redirect_uri=http://localhost:8080/api/auth/callback&state=xyz", nil)
	h.ServeHTTP(rec, req)

	if strings.Contains(rec.Body.String(), `value="admin"`) {
		t.Fatalf("username should not be prefilled without cookie: %s", rec.Body.String())
	}
}

func TestPKCECodeFlow(t *testing.T) {
	srv := New(newTestStore(t))
	h := srv.Handler()

	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := CodeChallengeS256(verifier)

	// 1) login with PKCE challenge.
	form := url.Values{}
	form.Set("username", "admin")
	form.Set("password", "admin")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/authorize?response_type=code&client_id=app&redirect_uri=http://localhost:8080/api/auth/callback&state=xyz&code_challenge="+url.QueryEscape(challenge)+"&code_challenge_method=S256",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	code := getQuery(t, rec.Header().Get("Location"), "code")

	// 2) exchange without verifier must fail.
	form = url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.SetBasicAuth("app", "s3cret")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without code_verifier, got %d", rec.Code)
	}

	// the failed exchange consumed the code; run a fresh login for the success path.
	form = url.Values{}
	form.Set("username", "admin")
	form.Set("password", "admin")
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost,
		"/authorize?response_type=code&client_id=app&redirect_uri=http://localhost:8080/api/auth/callback&state=xyz&code_challenge="+url.QueryEscape(challenge)+"&code_challenge_method=S256",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	code = getQuery(t, rec.Header().Get("Location"), "code")

	// 3) exchange with correct verifier succeeds and returns a refresh token.
	form = url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("code_verifier", verifier)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.SetBasicAuth("app", "s3cret")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid code_verifier, got %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"access_token"`) {
		t.Fatalf("no access_token: %s", body)
	}
	if !strings.Contains(body, `"refresh_token"`) {
		t.Fatalf("no refresh_token: %s", body)
	}
}

func TestRefreshGrant(t *testing.T) {
	srv := New(newTestStore(t))
	h := srv.Handler()

	// obtain a token via code flow.
	form := url.Values{}
	form.Set("username", "admin")
	form.Set("password", "admin")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/authorize?response_type=code&client_id=app&redirect_uri=http://localhost:8080/api/auth/callback",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	code := getQuery(t, rec.Header().Get("Location"), "code")

	form = url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.SetBasicAuth("app", "s3cret")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code exchange: %d body=%s", rec.Code, rec.Body.String())
	}
	parts := strings.Split(rec.Body.String(), `"refresh_token":"`)
	if len(parts) < 2 {
		t.Fatal("cannot extract refresh_token")
	}
	refresh := strings.Split(parts[1], `"`)[0]

	// refresh grant rotates the tokens.
	form = url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refresh)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.SetBasicAuth("app", "s3cret")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh grant: %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"access_token"`) || !strings.Contains(body, `"refresh_token"`) {
		t.Fatalf("expected rotated tokens: %s", body)
	}
	parts = strings.Split(body, `"refresh_token":"`)
	newRefresh := strings.Split(parts[1], `"`)[0]
	if newRefresh == refresh {
		t.Fatal("refresh token should be rotated")
	}

	// the old refresh token is single-use.
	form = url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refresh)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.SetBasicAuth("app", "s3cret")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("reused refresh token should fail: %d", rec.Code)
	}
}

func getQuery(t *testing.T, raw, key string) string {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	v := u.Query().Get(key)
	if v == "" {
		t.Fatalf("missing %q in %q", key, raw)
	}
	return v
}
