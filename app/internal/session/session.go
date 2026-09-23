// Package session implements a server-side, cookie-backed session store. It
// is the BFF (backend-for-frontend) core: the OAuth2 access token issued by the
// identity provider is held server-side and never reaches the browser. The
// browser only holds an opaque, random session id in a signed cookie.
//
// The store is in-memory by default and therefore suitable for a single
// instance. It is safe for concurrent use.
package session

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"sync"
	"time"
)

// Session is the server-side data bound to a session id.
type Session struct {
	ID        string
	Subject   string
	Name      string
	Token     string
	TokenType string
	// RefreshToken is the OAuth2 refresh token used to renew an expired
	// access token without user interaction.
	RefreshToken string
	ExpiresAt    time.Time
	CreatedAt    time.Time
	// State holds an ephemeral OAuth2 state value during the authorization
	// code flow; empty once the session is authenticated.
	State string
	// CodeVerifier holds the ephemeral PKCE code verifier during the
	// authorization code flow; empty once the session is authenticated.
	CodeVerifier string
}

// IsAuthenticated reports whether the session holds a valid, non-expired token.
func (s Session) IsAuthenticated() bool {
	return s.Token != "" && (s.ExpiresAt.IsZero() || time.Now().Before(s.ExpiresAt))
}

// Store persists sessions keyed by their random id. The interface allows
// alternative backends (e.g. Redis) for multi-instance deployments; the
// default in-memory implementation is only suitable for a single instance.
type Store interface {
	Create() *Session
	Get(id string) *Session
	Save(sess *Session)
	Delete(id string)
}

// MemoryStore keeps sessions in process memory. It is safe for concurrent use.
type MemoryStore struct {
	mu       sync.Mutex
	sessions map[string]*Session
}

// NewStore creates an empty in-memory session store.
func NewStore() *MemoryStore {
	return &MemoryStore{sessions: map[string]*Session{}}
}

// Create starts a new session and returns it.
func (s *MemoryStore) Create() *Session {
	id := randomID()
	now := time.Now()
	sess := &Session{
		ID:        id,
		CreatedAt: now,
	}
	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()
	return sess
}

// maxSessionAge bounds how long any session may live server-side, even when
// it never obtains a token expiry (e.g. abandoned pending logins).
const maxSessionAge = 24 * time.Hour

// Get returns the session for the given id, or nil if it does not exist or has
// expired.
func (s *MemoryStore) Get(id string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok {
		return nil
	}
	if !sess.ExpiresAt.IsZero() && !time.Now().Before(sess.ExpiresAt) {
		delete(s.sessions, id)
		return nil
	}
	if !sess.CreatedAt.IsZero() && time.Since(sess.CreatedAt) > maxSessionAge {
		delete(s.sessions, id)
		return nil
	}
	cp := *sess
	return &cp
}

// Save persists updates to a session (e.g. after storing a token).
func (s *MemoryStore) Save(sess *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sess.ID] = sess
}

// Delete removes a session, effectively logging the user out.
func (s *MemoryStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

// CookieName is the name of the session cookie.
const CookieName = "jansvca_sid"

// ErrNoSession is returned when a request carries no valid session.
var ErrNoSession = errors.New("session: no session")

// Manager owns a store plus cookie configuration and provides request-scoped
// helpers to read, set and clear the session.
type Manager struct {
	Store     Store
	CookieCfg CookieConfig
}

// CookieConfig configures the session cookie attributes.
type CookieConfig struct {
	Path     string
	Domain   string
	Secure   bool
	HttpOnly bool
	SameSite http.SameSite
	MaxAge   time.Duration
}

// DefaultCookieConfig returns sensible cookie defaults.
func DefaultCookieConfig(secure bool) CookieConfig {
	return CookieConfig{
		Path:     "/",
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   12 * time.Hour,
	}
}

// NewManager creates a manager with a fresh in-memory store and the given
// cookie config.
func NewManager(cfg CookieConfig) *Manager {
	return &Manager{Store: NewStore(), CookieCfg: cfg}
}

// NewManagerWithStore creates a manager backed by the given store.
func NewManagerWithStore(store Store, cfg CookieConfig) *Manager {
	return &Manager{Store: store, CookieCfg: cfg}
}

// FromRequest reads the session bound to the request, if any.
func (m *Manager) FromRequest(r *http.Request) *Session {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return nil
	}
	return m.Store.Get(c.Value)
}

// SetCookie writes the session id cookie on the response.
func (m *Manager) SetCookie(w http.ResponseWriter, sess *Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    sess.ID,
		Path:     m.CookieCfg.Path,
		Domain:   m.CookieCfg.Domain,
		Secure:   m.CookieCfg.Secure,
		HttpOnly: m.CookieCfg.HttpOnly,
		SameSite: m.CookieCfg.SameSite,
		MaxAge:   int(m.CookieCfg.MaxAge.Seconds()),
	})
}

// ClearCookie removes the session cookie and drops the server-side session.
func (m *Manager) ClearCookie(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(CookieName); err == nil {
		m.Store.Delete(c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     m.CookieCfg.Path,
		Domain:   m.CookieCfg.Domain,
		Secure:   m.CookieCfg.Secure,
		HttpOnly: m.CookieCfg.HttpOnly,
		SameSite: m.CookieCfg.SameSite,
		MaxAge:   -1,
	})
}

func randomID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// A predictable fallback id would allow session guessing; fail hard.
		panic("session: crypto/rand failed: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
