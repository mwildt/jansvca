// Package server implements the OAuth2 endpoints of the simple IdP:
//
//	GET/POST /authorize  – authorization-code flow with a built-in login form
//	POST      /token      – code exchange (grant_type=authorization_code)
//	POST      /introspect – RFC 7662 token introspection
//
// The IdP is intentionally minimal: a single client, users from a YAML file,
// in-memory code/token store. It is meant for local development of the BFF.
package server

import (
	"encoding/json"
	"html"
	"net/http"
	"net/url"
	"time"

	"github.com/mwildt/jansvca/idp/internal/config"
	"github.com/mwildt/jansvca/idp/internal/token"
)

// lastUserCookie is a long-lived, non-session cookie that only remembers the
// last username used to sign in, so the login form can prefill it. It carries
// no authentication value.
const lastUserCookie = "jansvca_lastuser"

// lastUserMaxAge is how long the last-user cookie is kept (1 year).
const lastUserMaxAge = 3600 * 24 * 365

// Server is the IdP HTTP handler.
type Server struct {
	store  *config.Store
	tokens *token.Store
}

// New creates an IdP server backed by the given user/client store.
func New(store *config.Store) *Server {
	return &Server{store: store, tokens: token.NewStore()}
}

// Handler returns the mux serving all IdP endpoints.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/authorize", s.handleAuthorize)
	mux.HandleFunc("/token", s.handleToken)
	mux.HandleFunc("/introspect", s.handleIntrospect)
	return mux
}

// --- /authorize ----------------------------------------------------------

func (s *Server) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	state := q.Get("state")
	scope := q.Get("scope")
	responseType := q.Get("response_type")

	client, ok := s.store.Client(clientID)
	if !ok {
		http.Error(w, "invalid client_id", http.StatusBadRequest)
		return
	}
	if responseType != "code" {
		http.Error(w, "unsupported response_type", http.StatusBadRequest)
		return
	}
	if !redirectAllowed(client, redirectURI) {
		http.Error(w, "redirect_uri not allowed", http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodPost {
		s.handleAuthorizeSubmit(w, r, client, redirectURI, state, scope)
		return
	}

	// GET: render the login form. Prefill the username from the last-user
	// cookie if present (non-authenticating convenience only).
	lastUser := ""
	if c, err := r.Cookie(lastUserCookie); err == nil && c.Value != "" {
		lastUser = c.Value
	}
	s.renderLogin(w, r, clientID, redirectURI, state, scope, "", lastUser)
}

func (s *Server) handleAuthorizeSubmit(w http.ResponseWriter, r *http.Request, client config.Client, redirectURI, state, scope string) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	subject := r.FormValue("username")
	password := r.FormValue("password")
	user, err := s.store.VerifyPassword(subject, password)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		s.renderLogin(w, r, client.ID, redirectURI, state, scope, "Anmeldung fehlgeschlagen.", subject)
		return
	}
	code, err := s.tokens.IssueCode(client.ID, user.Subject, user.Name, redirectURI, scope)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     lastUserCookie,
		Value:    user.Subject,
		Path:     "/",
		MaxAge:   lastUserMaxAge,
		Expires:  time.Now().Add(lastUserMaxAge * time.Second),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	u, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, "bad redirect_uri", http.StatusBadRequest)
		return
	}
	qu := u.Query()
	qu.Set("code", code)
	if state != "" {
		qu.Set("state", state)
	}
	u.RawQuery = qu.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func (s *Server) renderLogin(w http.ResponseWriter, r *http.Request, clientID, redirectURI, state, scope, errMsg, lastUser string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	esc := html.EscapeString
	body := `<!doctype html><html lang="de"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>jansvca IdP – Anmeldung</title>
<style>
*{box-sizing:border-box}
:root{
--bg:#0b1120;--surface:#111827;--surface-alt:#1f2937;--border:#273244;--border-strong:#3b4862;
--text:#e5e7eb;--muted:#94a3b8;--primary:#6366f1;--primary-hover:#4f46e5;--danger:#ef4444;
--radius:18px;--radius-sm:10px;
--font:Inter,system-ui,-apple-system,Segoe UI,Roboto,Helvetica,Arial,sans-serif;
}
body{font-family:var(--font);background:var(--bg);margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;padding:24px;color:var(--text);
background-image:
radial-gradient(1200px 600px at 100% -10%,rgba(99,102,241,.14),transparent 60%),
radial-gradient(900px 500px at -10% 0%,rgba(34,197,94,.06),transparent 55%);
}
.card{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);padding:36px 32px;width:100%;max-width:380px;
box-shadow:0 20px 50px rgba(0,0,0,.55);}
.brand{display:flex;align-items:center;gap:10px;justify-content:center;margin-bottom:24px}
.brand .mark{width:30px;height:30px;border-radius:9px;background:linear-gradient(135deg,var(--primary),#8b5cf6);box-shadow:0 6px 16px rgba(99,102,241,.45)}
.brand span{font-weight:700;font-size:1.05rem;letter-spacing:-.02em}
h1{font-size:1.25rem;margin:0 0 6px;text-align:center;font-weight:700}
.subtitle{color:var(--muted);font-size:.85rem;text-align:center;margin:0 0 24px}
label{display:block;font-size:.8rem;color:var(--muted);margin-bottom:6px;font-weight:500}
.field{margin-bottom:18px}
input{width:100%;padding:11px 13px;border:1px solid var(--border);border-radius:var(--radius-sm);background:var(--surface-alt);color:var(--text);font-size:.95rem;font-family:inherit;transition:border-color .14s ease,box-shadow .14s ease}
input:focus{outline:none;border-color:var(--primary);box-shadow:0 0 0 3px rgba(99,102,241,.18)}
input::placeholder{color:#5b6477}
button{width:100%;padding:12px;border:0;border-radius:var(--radius-sm);background:var(--primary);color:#fff;font-size:.95rem;font-weight:600;font-family:inherit;cursor:pointer;transition:background .14s ease,transform .06s ease;box-shadow:0 6px 16px rgba(99,102,241,.35)}
button:hover{background:var(--primary-hover)}
button:active{transform:translateY(1px)}
.err{color:var(--danger);font-size:.85rem;margin-bottom:18px;text-align:center;background:rgba(239,68,68,.12);border:1px solid rgba(239,68,68,.3);padding:8px 12px;border-radius:var(--radius-sm)}
.hint{color:var(--muted);font-size:.75rem;text-align:center;margin-top:18px}
</style></head><body>
<form class="card" method="post">
<div class="brand"><span class="mark"></span><span>jansvca</span></div>
<h1>Anmeldung</h1>
<p class="subtitle">Melde dich über den OAuth2-Provider an.</p>`
	if errMsg != "" {
		body += `<div class="err">` + esc(errMsg) + `</div>`
	}
	focusUser := lastUser == ""
	body += `<div class="field"><label for="username">Benutzername (subject)</label>
<input id="username" name="username" value="` + esc(lastUser) + `"` + autofocusAttr(focusUser) + ` required></div>`
	body += `<div class="field"><label for="password">Passwort</label>
<input id="password" name="password" type="password"` + autofocusAttr(!focusUser) + ` required></div>`
	body += `<button type="submit">Anmelden</button>
</form>
<p class="hint">jansvca IdP</p>
</body></html>`
	_, _ = w.Write([]byte(body))
}

// autofocusAttr returns the HTML autofocus attribute when the field should
// receive focus, otherwise an empty string. Only one field may be autofocused.
func autofocusAttr(focus bool) string {
	if focus {
		return " autofocus"
	}
	return ""
}

func redirectAllowed(client config.Client, uri string) bool {
	if uri == "" {
		return false
	}
	for _, allowed := range client.RedirectURIs {
		if allowed == uri {
			return true
		}
	}
	return false
}

// --- /token --------------------------------------------------------------

func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		tokenError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	grantType := r.FormValue("grant_type")
	if grantType != "authorization_code" {
		tokenError(w, http.StatusBadRequest, "unsupported_grant_type")
		return
	}
	clientID, clientSecret, ok := clientAuth(r)
	if !ok {
		tokenError(w, http.StatusUnauthorized, "invalid_client")
		return
	}
	client, ok := s.store.Client(clientID)
	if !ok || (client.Secret != "" && client.Secret != clientSecret) {
		tokenError(w, http.StatusUnauthorized, "invalid_client")
		return
	}
	code := r.FormValue("code")
	c, err := s.tokens.ConsumeCode(code)
	if err != nil || c.ClientID != clientID {
		tokenError(w, http.StatusBadRequest, "invalid_grant")
		return
	}
	redirectURI := r.FormValue("redirect_uri")
	if redirectURI != "" && redirectURI != c.RedirectURI {
		tokenError(w, http.StatusBadRequest, "invalid_grant")
		return
	}
	t, err := s.tokens.IssueToken(c.Subject, c.Name, c.Scope)
	if err != nil {
		tokenError(w, http.StatusInternalServerError, "server_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": t.Token,
		"token_type":   "Bearer",
		"expires_in":   int(time.Until(t.ExpiresAt).Seconds()),
		"scope":        t.Scope,
	})
}

// --- /introspect ---------------------------------------------------------

func (s *Server) handleIntrospect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"active": false})
		return
	}
	// Client authentication (basic) is optional for the local IdP but accepted.
	clientID, _, _ := clientAuth(r)
	if clientID != "" {
		if _, ok := s.store.Client(clientID); !ok {
			writeJSON(w, http.StatusOK, map[string]any{"active": false})
			return
		}
	}
	tok := r.FormValue("token")
	t, err := s.tokens.Token(tok)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"active": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"active":     true,
		"sub":        t.Subject,
		"username":   t.Name,
		"scope":      t.Scope,
		"exp":        t.ExpiresAt.Unix(),
		"token_type": "Bearer",
	})
}

// --- helpers -------------------------------------------------------------

func clientAuth(r *http.Request) (string, string, bool) {
	if id, sec, ok := r.BasicAuth(); ok {
		return id, sec, true
	}
	id := r.FormValue("client_id")
	sec := r.FormValue("client_secret")
	if id != "" {
		return id, sec, true
	}
	return "", "", false
}

func tokenError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
