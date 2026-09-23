// Package web wires the BFF: it mounts the auth endpoints, the session-protected
// API proxy to the backend service, the optional gateway upstreams and serves
// the built SPA. The browser talks only to the BFF; the OAuth2 access token
// stays server-side.
package web

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/mwildt/jansvca/app/internal/gateway"
	"github.com/mwildt/jansvca/app/internal/oauth"
	"github.com/mwildt/jansvca/app/internal/session"
)


// Config configures the BFF web layer.
type Config struct {
	// Addr is the listen address, e.g. ":8080".
	Addr string
	// BackendURL is the base URL of the backend service (the /service HTTP API),
	// e.g. "http://localhost:8081". The BFF proxies /api/* to it.
	BackendURL string
	// SPADir is the directory holding the built frontend assets.
	SPADir string
	// OAuth is the OAuth2 provider config. If empty, auth is disabled and the
	// BFF proxies the backend without adding a Bearer token.
	OAuth oauth.Config
	// Upstreams configure the gateway reverse-proxy routes.
	Upstreams []gateway.Upstream
	// SecureCookies controls the Secure flag on session cookies.
	SecureCookies bool
}

// App is the assembled BFF HTTP handler.
type App struct {
	mgr      *session.Manager
	provider *oauth.Provider
	cfg      Config
	gateway  *gateway.Gateway
}

// New builds the BFF handler from config.
func New(cfg Config) (*App, error) {
	mgr := session.NewManager(session.DefaultCookieConfig(cfg.SecureCookies))
	app := &App{
		mgr:      mgr,
		provider: oauth.NewProvider(cfg.OAuth),
		cfg:      cfg,
	}
	if len(cfg.Upstreams) > 0 {
		gw, err := gateway.New(cfg.Upstreams, mgr)
		if err != nil {
			return nil, err
		}
		app.gateway = gw
	}
	return app, nil
}

// Handler returns the composed HTTP handler.
func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()

	a.registerAuth(mux)
	a.registerAPI(mux)
	a.registerGateway(mux)
	a.registerSPA(mux)

	return mux
}

// --- Auth endpoints -------------------------------------------------------

const (
	authLoginPath    = "/api/auth/login"
	authCallbackPath = "/api/auth/callback"
	authLogoutPath   = "/api/auth/logout"
	authUserPath     = "/api/auth/user"
)

func (a *App) registerAuth(mux *http.ServeMux) {
	mux.HandleFunc(authLoginPath, a.handleLogin)
	mux.HandleFunc(authCallbackPath, a.handleCallback)
	mux.HandleFunc(authLogoutPath, a.handleLogout)
	mux.HandleFunc(authUserPath, a.handleUser)
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	if a.cfg.OAuth.AuthorizationURL == "" {
		http.Error(w, "auth disabled", http.StatusNotImplemented)
		return
	}
	state := randomState()
	verifier, err := oauth.NewCodeVerifier()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	sess := a.mgr.Store.Create()
	sess.State = state
	sess.CodeVerifier = verifier
	a.mgr.Store.Save(sess)
	a.mgr.SetCookie(w, sess)

	http.Redirect(w, r, a.provider.AuthURL(state, oauth.CodeChallengeS256(verifier)), http.StatusFound)
}

func (a *App) handleCallback(w http.ResponseWriter, r *http.Request) {
	sess := a.mgr.FromRequest(r)
	if sess == nil {
		http.Error(w, "session expired", http.StatusBadRequest)
		return
	}
	expected := sess.State
	if expected == "" {
		http.Error(w, "no pending login", http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("state") != expected {
		http.Error(w, "state mismatch", http.StatusBadRequest)
		return
	}
	if errStr := r.URL.Query().Get("error"); errStr != "" {
		http.Error(w, "auth error: "+errStr, http.StatusBadRequest)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}
	tok, err := a.provider.Exchange(r.Context(), code, sess.CodeVerifier)
	if err != nil {
		http.Error(w, "token exchange failed", http.StatusBadGateway)
		return
	}
	sess.Token = tok.AccessToken
	sess.TokenType = tok.TokenType
	sess.RefreshToken = tok.RefreshToken
	sess.ExpiresAt = oauth.ExpiresAt(time.Now(), tok.ExpiresIn)
	sess.State = ""
	sess.CodeVerifier = ""
	// Enrich and validate from introspection if available. A token the
	// provider reports as inactive must never yield an authenticated session,
	// and introspection errors are surfaced instead of silently ignored.
	if a.cfg.OAuth.IntrospectionURL != "" {
		ir, err := a.provider.Introspect(r.Context(), tok.AccessToken)
		if err != nil {
			log.Printf("auth: introspection failed: %v", err)
			http.Error(w, "token validation failed", http.StatusBadGateway)
			return
		}
		if !ir.Active {
			http.Error(w, "token not active", http.StatusUnauthorized)
			return
		}
		sess.Subject = ir.Sub
		sess.Name = ir.Username
		if ir.Exp > 0 {
			sess.ExpiresAt = time.Unix(ir.Exp, 0)
		}
	}
	a.mgr.Store.Save(sess)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	a.mgr.ClearCookie(w, r)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (a *App) handleUser(w http.ResponseWriter, r *http.Request) {
	sess := a.mgr.FromRequest(r)
	if sess == nil || !sess.IsAuthenticated() {
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"subject":       sess.Subject,
		"name":          sess.Name,
	})
}

// --- Backend API proxy ----------------------------------------------------

func (a *App) registerAPI(mux *http.ServeMux) {
	target, err := parseBackend(a.cfg.BackendURL)
	if err != nil {
		// Without a backend configured, /api returns 502.
		mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "backend not configured", http.StatusBadGateway)
		})
		return
	}
	mux.Handle("/api/", a.requireAuth(a.backendProxy(target)))
}

// backendProxy builds the reverse proxy to the backend service. The session
// token is injected as a Bearer header so the backend's existing OAuth2
// introspection middleware keeps working.
func (a *App) backendProxy(target string) http.Handler {
	proxy := newBackendProxy(target)
	orig := proxy.Director
	proxy.Director = func(req *http.Request) {
		orig(req)
		if sess := a.mgr.FromRequest(req); sess != nil && sess.Token != "" {
			req.Header.Set("Authorization", "Bearer "+sess.Token)
		}
	}
	return proxy
}

// requireAuth gates access behind an authenticated session. When OAuth is
// disabled (no AuthorizationURL) the gate is open so local/dev setups work
// without a provider.
func (a *App) requireAuth(next http.Handler) http.Handler {
	if a.cfg.OAuth.AuthorizationURL == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := a.mgr.FromRequest(r)
		if sess == nil || !sess.IsAuthenticated() {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		sess = a.ensureFreshToken(w, r, sess)
		if sess == nil {
			return
		}
		next.ServeHTTP(w, r)
	})
}

// tokenRefreshSkew refreshes an access token slightly before it actually
// expires, so a proxied request never carries an already-expired token.
const tokenRefreshSkew = 30 * time.Second

// ensureFreshToken refreshes the session's access token via the provider's
// refresh-token grant when it is about to expire. On failure the session is
// dropped and the caller receives a 401, forcing a fresh login.
func (a *App) ensureFreshToken(w http.ResponseWriter, r *http.Request, sess *session.Session) *session.Session {
	if sess.Token == "" || sess.ExpiresAt.IsZero() || time.Until(sess.ExpiresAt) > tokenRefreshSkew {
		return sess
	}
	if sess.RefreshToken == "" {
		a.mgr.Store.Delete(sess.ID)
		a.mgr.ClearCookie(w, r)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return nil
	}
	tok, err := a.provider.Refresh(r.Context(), sess.RefreshToken)
	if err != nil {
		log.Printf("auth: token refresh failed: %v", err)
		a.mgr.Store.Delete(sess.ID)
		a.mgr.ClearCookie(w, r)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return nil
	}
	sess.Token = tok.AccessToken
	sess.TokenType = tok.TokenType
	sess.RefreshToken = tok.RefreshToken
	sess.ExpiresAt = oauth.ExpiresAt(time.Now(), tok.ExpiresIn)
	a.mgr.Store.Save(sess)
	return sess
}

// --- Gateway ---------------------------------------------------------------

func (a *App) registerGateway(mux *http.ServeMux) {
	if a.gateway == nil {
		return
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if a.gateway.Matches(r.URL.Path) {
			a.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				a.gateway.ServeHTTP(w, r)
			})).ServeHTTP(w, r)
			return
		}
		a.serveSPA(w, r)
	})
}

// --- SPA -------------------------------------------------------------------

func (a *App) registerSPA(mux *http.ServeMux) {
	if a.gateway != nil {
		// Gateway already owns "/" and falls through to serveSPA.
		return
	}
	mux.HandleFunc("/", a.serveSPA)
}

func (a *App) serveSPA(w http.ResponseWriter, r *http.Request) {
	if a.cfg.SPADir == "" {
		http.Error(w, "spa not configured", http.StatusNotFound)
		return
	}
	fs := http.FileServer(http.Dir(a.cfg.SPADir))
	clean := path.Clean(r.URL.Path)
	full := path.Join(a.cfg.SPADir, clean)
	if !strings.HasPrefix(full, a.cfg.SPADir) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if !fileExists(full) && !isDir(full) {
		// SPA fallback: unknown client routes serve index.html.
		serveFile(w, r, path.Join(a.cfg.SPADir, "index.html"))
		return
	}
	fs.ServeHTTP(w, r)
}

// --- helpers ---------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func randomState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func parseBackend(raw string) (string, error) {
	if raw == "" {
		return "", errors.New("no backend url")
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		return "", errors.New("backend url must start with http(s)://")
	}
	return strings.TrimRight(raw, "/"), nil
}

func fileExists(p string) bool {
	// declared in spa.go
	return existsFile(p)
}

func isDir(p string) bool {
	return existsDir(p)
}
