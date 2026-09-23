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

const testAuthorizeQuery = "response_type=code&client_id=app&redirect_uri=http://localhost:8080/api/auth/callback"

// login performs the browser-side login dance: GET the login form to obtain
// the CSRF cookie and the embedded token, then POST credentials with the
// token. It returns the response recorder of the POST.
func login(t *testing.T, h http.Handler, query, username, password string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/authorize?"+query, nil)
	h.ServeHTTP(rec, req)
	var cookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == csrfCookie {
			cookie = c
			break
		}
	}
	if cookie == nil {
		t.Fatalf("%s cookie not set", csrfCookie)
	}
	token := extractCSRFToken(t, rec.Body.String())

	form := url.Values{}
	form.Set("username", username)
	form.Set("password", password)
	form.Set(csrfFieldName, token)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/authorize?"+query, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)
	return rec
}

// extractCSRFToken pulls the csrf_token hidden-field value out of the rendered
// login form HTML.
func extractCSRFToken(t *testing.T, body string) string {
	t.Helper()
	const marker = `name="csrf_token" value="`
	i := strings.Index(body, marker)
	if i < 0 {
		t.Fatalf("csrf_token field missing in login form: %s", body)
	}
	rest := body[i+len(marker):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		t.Fatalf("malformed csrf_token field")
	}
	return rest[:end]
}

// loginCode runs the full login dance and returns the authorization code from
// the redirect Location header.
func loginCode(t *testing.T, h http.Handler, query, username, password string) string {
	t.Helper()
	rec := login(t, h, query, username, password)
	if rec.Code != http.StatusFound {
		t.Fatalf("login: expected 302, got %d body=%s", rec.Code, rec.Body.String())
	}
	return getQuery(t, rec.Header().Get("Location"), "code")
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

	// 1) Login via the browser dance (GET form + CSRF, then POST) -> redirect with code.
	rec := login(t, h, testAuthorizeQuery+"&state=xyz&scope=openid", "admin", "admin")

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
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", "http://localhost:8080/api/auth/callback")
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
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

	code := loginCode(t, h, testAuthorizeQuery, "admin", "admin")

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", "http://localhost:8080/api/auth/callback")
	form.Set("client_id", "app")
	form.Set("client_secret", "s3cret")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("form client auth: %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAuthorizeSetsLastUserCookie(t *testing.T) {
	srv := New(newTestStore(t))
	h := srv.Handler()

	rec := login(t, h, testAuthorizeQuery+"&state=xyz", "admin", "admin")
	if rec.Code != http.StatusFound {
		t.Fatalf("login: expected 302, got %d", rec.Code)
	}

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

	pkceQuery := testAuthorizeQuery + "&state=xyz&code_challenge=" + url.QueryEscape(challenge) + "&code_challenge_method=S256"

	// 1) login with PKCE challenge.
	code := loginCode(t, h, pkceQuery, "admin", "admin")

	// 2) exchange without verifier must fail.
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.SetBasicAuth("app", "s3cret")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without code_verifier, got %d", rec.Code)
	}

	// the failed exchange consumed the code; run a fresh login for the success path.
	code = loginCode(t, h, pkceQuery, "admin", "admin")

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
	code := loginCode(t, h, testAuthorizeQuery, "admin", "admin")

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
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

func TestRevokeEndpoint(t *testing.T) {
	srv := New(newTestStore(t))
	h := srv.Handler()

	// obtain an access token via code flow.
	code := loginCode(t, h, testAuthorizeQuery, "admin", "admin")

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.SetBasicAuth("app", "s3cret")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	parts := strings.Split(rec.Body.String(), `"access_token":"`)
	if len(parts) < 2 {
		t.Fatal("cannot extract access_token")
	}
	accessToken := strings.Split(parts[1], `"`)[0]

	// introspect must report the token active.
	form = url.Values{}
	form.Set("token", accessToken)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/introspect", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), `"active":true`) {
		t.Fatalf("expected active token: %s", rec.Body.String())
	}

	// revoke requires client auth.
	form = url.Values{}
	form.Set("token", accessToken)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/revoke", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without client auth, got %d", rec.Code)
	}

	// revoke with auth succeeds.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/revoke", strings.NewReader(form.Encode()))
	req.SetBasicAuth("app", "s3cret")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for revoke, got %d", rec.Code)
	}

	// introspect must now report inactive.
	form = url.Values{}
	form.Set("token", accessToken)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/introspect", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), `"active":false`) {
		t.Fatalf("expected inactive token after revoke: %s", rec.Body.String())
	}
}

func TestAuthorizeWithoutCSRFTokenRejected(t *testing.T) {
	srv := New(newTestStore(t))
	h := srv.Handler()

	// Prime the CSRF cookie via GET, then POST credentials without the token.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/authorize?"+testAuthorizeQuery, nil)
	h.ServeHTTP(rec, req)
	var cookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == csrfCookie {
			cookie = c
			break
		}
	}
	if cookie == nil {
		t.Fatalf("%s cookie not set", csrfCookie)
	}

	form := url.Values{}
	form.Set("username", "admin")
	form.Set("password", "admin")
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/authorize?"+testAuthorizeQuery, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)

	// The login must be rejected with the form re-rendered, not redirected.
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 re-render, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Sitzung abgelaufen") {
		t.Fatalf("expected CSRF error message: %s", rec.Body.String())
	}
	if strings.Contains(rec.Header().Get("Location"), "code=") {
		t.Fatal("no code must be issued without a CSRF token")
	}
}

func TestAuthorizeWrongCSRFTokenRejected(t *testing.T) {
	srv := New(newTestStore(t))
	h := srv.Handler()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/authorize?"+testAuthorizeQuery, nil)
	h.ServeHTTP(rec, req)
	var cookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == csrfCookie {
			cookie = c
			break
		}
	}
	if cookie == nil {
		t.Fatalf("%s cookie not set", csrfCookie)
	}

	form := url.Values{}
	form.Set("username", "admin")
	form.Set("password", "admin")
	form.Set(csrfFieldName, "forged-token-value")
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/authorize?"+testAuthorizeQuery, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 re-render, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Sitzung abgelaufen") {
		t.Fatalf("expected CSRF error message: %s", rec.Body.String())
	}
}

func TestAuthorizeSetsSecurityHeaders(t *testing.T) {
	srv := New(newTestStore(t))
	h := srv.Handler()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/authorize?"+testAuthorizeQuery, nil)
	h.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "frame-ancestors 'none'") || !strings.Contains(csp, "default-src 'none'") {
		t.Fatalf("unexpected CSP: %q", csp)
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("X-Frame-Options: %q", rec.Header().Get("X-Frame-Options"))
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("X-Content-Type-Options: %q", rec.Header().Get("X-Content-Type-Options"))
	}
	if rec.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("Referrer-Policy: %q", rec.Header().Get("Referrer-Policy"))
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control: %q", rec.Header().Get("Cache-Control"))
	}
}

func TestCSRFCookieIsHttpOnlySessionScoped(t *testing.T) {
	srv := New(newTestStore(t))
	h := srv.Handler()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/authorize?"+testAuthorizeQuery, nil)
	h.ServeHTTP(rec, req)

	var cookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == csrfCookie {
			cookie = c
			break
		}
	}
	if cookie == nil {
		t.Fatalf("%s cookie not set", csrfCookie)
	}
	if !cookie.HttpOnly {
		t.Fatal("CSRF cookie must be HttpOnly")
	}
	if cookie.MaxAge != 0 && cookie.Expires.IsZero() {
		t.Fatalf("CSRF cookie must be session-scoped, got MaxAge=%d", cookie.MaxAge)
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("CSRF cookie SameSite: %v", cookie.SameSite)
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
