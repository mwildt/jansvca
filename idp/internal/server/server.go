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
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"time"

	"github.com/mwildt/jansvca/idp/internal/config"
	"github.com/mwildt/jansvca/idp/internal/token"
)

//go:embed login.html
var loginHTML []byte

// loginTmpl is the parsed login form template, parsed once at package init.
var loginTmpl = template.Must(template.New("login").Parse(string(loginHTML)))

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
	mux.HandleFunc("/revoke", s.handleRevoke)
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
	codeChallenge := q.Get("code_challenge")
	codeChallengeMethod := q.Get("code_challenge_method")

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
	if codeChallengeMethod != "" && codeChallengeMethod != "S256" {
		http.Error(w, "unsupported code_challenge_method", http.StatusBadRequest)
		return
	}
	if codeChallenge != "" && codeChallengeMethod == "" {
		http.Error(w, "code_challenge requires code_challenge_method", http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodPost {
		s.handleAuthorizeSubmit(w, r, client, redirectURI, state, scope, codeChallenge)
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

func (s *Server) handleAuthorizeSubmit(w http.ResponseWriter, r *http.Request, client config.Client, redirectURI, state, scope, codeChallenge string) {
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
	code, err := s.tokens.IssueCode(client.ID, user.Subject, user.Name, redirectURI, scope, codeChallenge)
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
	data := struct {
		ErrMsg        string
		LastUser      string
		FocusUser     bool
		FocusPassword bool
	}{
		ErrMsg:        errMsg,
		LastUser:      lastUser,
		FocusUser:     lastUser == "",
		FocusPassword: lastUser != "",
	}
	_ = loginTmpl.Execute(w, data)
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
	if grantType != "authorization_code" && grantType != "refresh_token" {
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
	if grantType == "refresh_token" {
		s.handleRefreshGrant(w, client, r)
		return
	}
	code := r.FormValue("code")
	c, err := s.tokens.ConsumeCode(code)
	if err != nil || c.ClientID != clientID {
		tokenError(w, http.StatusBadRequest, "invalid_grant")
		return
	}
	if c.CodeChallenge != "" {
		verifier := r.FormValue("code_verifier")
		if verifier == "" || CodeChallengeS256(verifier) != c.CodeChallenge {
			tokenError(w, http.StatusBadRequest, "invalid_grant")
			return
		}
	}
	redirectURI := r.FormValue("redirect_uri")
	if redirectURI != "" && redirectURI != c.RedirectURI {
		tokenError(w, http.StatusBadRequest, "invalid_grant")
		return
	}
	t, refresh, err := s.tokens.IssueToken(c.Subject, c.Name, c.Scope)
	if err != nil {
		tokenError(w, http.StatusInternalServerError, "server_error")
		return
	}
	writeTokenResponse(w, t, refresh)
}

// handleRefreshGrant exchanges a refresh token for a new access token
// (RFC 6749 §6). Refresh tokens are single-use; both tokens are rotated.
func (s *Server) handleRefreshGrant(w http.ResponseWriter, client config.Client, r *http.Request) {
	refreshToken := r.FormValue("refresh_token")
	old, err := s.tokens.ConsumeRefreshToken(refreshToken)
	if err != nil {
		tokenError(w, http.StatusBadRequest, "invalid_grant")
		return
	}
	t, refresh, err := s.tokens.IssueToken(old.Subject, old.Name, old.Scope)
	if err != nil {
		tokenError(w, http.StatusInternalServerError, "server_error")
		return
	}
	writeTokenResponse(w, t, refresh)
}

// CodeChallengeS256 derives the RFC 7636 S256 code challenge for a verifier.
func CodeChallengeS256(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func writeTokenResponse(w http.ResponseWriter, t *token.AccessToken, refresh string) {
	resp := map[string]any{
		"access_token": t.Token,
		"token_type":   "Bearer",
		"expires_in":   int(time.Until(t.ExpiresAt).Seconds()),
		"scope":        t.Scope,
	}
	if refresh != "" {
		resp["refresh_token"] = refresh
	}
	writeJSON(w, http.StatusOK, resp)
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

// --- /revoke --------------------------------------------------------------

// handleRevoke implements RFC 7009 token revocation for access tokens (the
// token_hint carries the access token; its refresh tokens are revoked with
// it). The response is always 200, per RFC 7009 §2.2.
func (s *Server) handleRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
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
	s.tokens.Revoke(r.FormValue("token"))
	writeJSON(w, http.StatusOK, map[string]any{})
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
