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

	// GET: render the login form.
	s.renderLogin(w, r, clientID, redirectURI, state, scope, "")
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
		s.renderLogin(w, r, client.ID, redirectURI, state, scope, "Anmeldung fehlgeschlagen.")
		return
	}
	code, err := s.tokens.IssueCode(client.ID, user.Subject, user.Name, redirectURI, scope)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
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

func (s *Server) renderLogin(w http.ResponseWriter, r *http.Request, clientID, redirectURI, state, scope, errMsg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	esc := html.EscapeString
	body := `<!doctype html><html lang="de"><head><meta charset="utf-8">
<title>jansvca IdP – Anmeldung</title>
<style>
body{font-family:system-ui,sans-serif;background:#f1f5f9;margin:0;display:flex;min-height:100vh;align-items:center;justify-content:center}
.card{background:#fff;border:1px solid #e2e8f0;border-radius:10px;padding:24px;max-width:340px;width:100%}
h1{font-size:1.1rem;margin:0 0 12px}
label{display:block;font-size:.8rem;color:#64748b;margin-bottom:4px}
input{width:100%;padding:8px;border:1px solid #e2e8f0;border-radius:6px;font-size:.95rem;box-sizing:border-box;margin-bottom:12px}
button{width:100%;padding:8px;border:0;border-radius:6px;background:#1d4ed8;color:#fff;font-size:.95rem;cursor:pointer}
.err{color:#dc2626;font-size:.85rem;margin-bottom:12px}
</style></head><body>
<form class="card" method="post">
<h1>Anmeldung – jansvca IdP</h1>`
	if errMsg != "" {
		body += `<div class="err">` + esc(errMsg) + `</div>`
	}
	body += `<label>Benutzername (subject)</label>
<input name="username" autofocus required>
<label>Passwort</label>
<input name="password" type="password" required>
<button type="submit">Anmelden</button>
</form></body></html>`
	_, _ = w.Write([]byte(body))
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
