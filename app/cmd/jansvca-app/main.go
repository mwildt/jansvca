// Command jansvca-app is the web application (BFF). It serves the SPA, the auth
// endpoints, proxies /api/* to the backend service and optionally fronts
// configured upstream services as an authenticated gateway.
//
// The backend service is run as a separate process (or sidecar); the BFF talks
// to it over HTTP. In the simplest local setup both run on the same host.
package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/mwildt/jansvca/app/internal/gateway"
	"github.com/mwildt/jansvca/app/internal/oauth"
	"github.com/mwildt/jansvca/app/internal/web"
)

func main() {
	cfg := web.Config{
		Addr:          envOr("JANSVCA_ADDR", ":8080"),
		BackendURL:    os.Getenv("JANSVCA_BACKEND_URL"),
		SPADir:        envOr("JANSVCA_SPA_DIR", "./app/dist"),
		SecureCookies: envBool("JANSVCA_SECURE_COOKIES", false),
	}

	cfg.OAuth = oauth.Config{
		AuthorizationURL: os.Getenv("JANSVCA_OAUTH2_AUTHORIZATION_URL"),
		TokenURL:         os.Getenv("JANSVCA_OAUTH2_TOKEN_URL"),
		IntrospectionURL: os.Getenv("JANSVCA_OAUTH2_INTROSPECTION_URL"),
		ClientID:         os.Getenv("JANSVCA_OAUTH2_CLIENT_ID"),
		ClientSecret:     os.Getenv("JANSVCA_OAUTH2_CLIENT_SECRET"),
		RedirectURL:      envOr("JANSVCA_OAUTH2_REDIRECT_URL", "http://localhost:8080/api/auth/callback"),
		Scope:            envOr("JANSVCA_OAUTH2_SCOPE", "openid profile"),
	}

	upstreams, err := parseUpstreams(os.Getenv("JANSVCA_GATEWAY_UPSTREAMS"))
	if err != nil {
		log.Fatalf("gateway upstreams: %v", err)
	}
	cfg.Upstreams = upstreams

	app, err := web.New(cfg)
	if err != nil {
		log.Fatalf("app: %v", err)
	}

	log.Printf("jansvca-app listening on %s (backend=%s spa=%s auth=%v)",
		cfg.Addr, orDash(cfg.BackendURL), cfg.SPADir, cfg.OAuth.AuthorizationURL != "")
	if err := http.ListenAndServe(cfg.Addr, app.Handler()); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// parseUpstreams parses a comma-separated list of "prefix=target" entries,
// e.g. "/proxy/svc=https://api.example.com,/proxy/other=https://other.example.com".
// A trailing "/strip" removes the prefix before forwarding.
func parseUpstreams(raw string) ([]gateway.Upstream, error) {
	if raw == "" {
		return nil, nil
	}
	var out []gateway.Upstream
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		strip := false
		if strings.HasSuffix(part, "/strip") {
			strip = true
			part = strings.TrimSuffix(part, "/strip")
		}
		idx := strings.Index(part, "=")
		if idx <= 0 {
			return nil, fmt.Errorf("invalid upstream %q: expected prefix=target", part)
		}
		prefix := strings.TrimSpace(part[:idx])
		target := strings.TrimSpace(part[idx+1:])
		if prefix == "" || target == "" {
			return nil, errors.New("upstream prefix and target are required")
		}
		out = append(out, gateway.Upstream{Prefix: prefix, Target: target, StripPrefix: strip})
	}
	return out, nil
}
