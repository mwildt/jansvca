// Package gateway implements a configurable reverse proxy that fronts external
// services. Each configured upstream is exposed under a path prefix; the BFF
// injects the session's OAuth2 access token as a Bearer Authorization header on
// every proxied request, so downstream services receive an authenticated call
// without ever seeing the browser's session cookie.
//
// This turns the app into both an application host and an API gateway.
package gateway

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"

	"github.com/mwildt/jansvca/app/internal/session"
)

// Upstream maps a path prefix to a target service URL.
type Upstream struct {
	// Prefix is the path prefix this upstream is mounted under, e.g. "/proxy/svc".
	Prefix string
	// Target is the upstream base URL, e.g. "https://api.example.com".
	Target string
	// StripPrefix controls whether the prefix is removed before forwarding.
	StripPrefix bool
}

// Gateway holds the compiled reverse proxies for all configured upstreams.
type Gateway struct {
	mu      sync.RWMutex
	routes  []route
	manager *session.Manager
}

type route struct {
	prefix string
	proxy  *httputil.ReverseProxy
	strip  bool
}

// New builds a gateway from the given upstreams and session manager. Upstreams
// with invalid target URLs are skipped.
func New(upstreams []Upstream, mgr *session.Manager) (*Gateway, error) {
	if mgr == nil {
		return nil, errors.New("gateway: nil session manager")
	}
	g := &Gateway{manager: mgr}
	for _, u := range upstreams {
		target, err := url.Parse(u.Target)
		if err != nil {
			return nil, err
		}
		proxy := httputil.NewSingleHostReverseProxy(target)
		orig := proxy.Director
		proxy.Director = func(req *http.Request) {
			orig(req)
			if sess := mgr.FromRequest(req); sess != nil && sess.Token != "" {
				req.Header.Set("Authorization", "Bearer "+sess.Token)
			}
		}
		g.routes = append(g.routes, route{
			prefix: u.Prefix,
			proxy:  proxy,
			strip:  u.StripPrefix,
		})
	}
	return g, nil
}

// ServeHTTP dispatches to the first matching route. It returns false (without
// writing to w) if no route matches, so callers can fall through to other
// handlers.
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	for _, rt := range g.routes {
		if !strings.HasPrefix(r.URL.Path, rt.prefix) {
			continue
		}
		out := r.Clone(r.Context())
		if rt.strip {
			out.URL.Path = strings.TrimPrefix(out.URL.Path, rt.prefix)
			if !strings.HasPrefix(out.URL.Path, "/") {
				out.URL.Path = "/" + out.URL.Path
			}
			out.URL.RawPath = ""
		}
		rt.proxy.ServeHTTP(w, out)
		return true
	}
	return false
}
