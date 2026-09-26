// Command idp is a minimal OAuth2 provider (authorization-code flow) for local
// development of the jansvca BFF. Users and clients are loaded from a YAML
// config file; codes and tokens are kept in memory. Passwords may be stored as
// bcrypt hashes or as plaintext (hashed on load for dev convenience).
//
// The IdP serves two listeners: the main port (IDP_ADDR) exposes the browser-
// and client-facing endpoints (/authorize, /token, /revoke); the introspection
// port (IDP_INTROSPECT_ADDR) serves only /introspect, so resource servers can
// be wired to a dedicated, optionally TLS/mTLS-terminated endpoint. When
// IDP_INTROSPECT_ADDR is empty, introspection stays on the main port.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/mwildt/jansvca/idp/internal/config"
	"github.com/mwildt/jansvca/idp/internal/server"
)

func main() {
	addr := envOr("IDP_ADDR", ":8080")
	introspectAddr := os.Getenv("IDP_INTROSPECT_ADDR")
	configPath := envOr("IDP_CONFIG", "./idp-config.yaml")

	store, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("idp: load config: %v", err)
	}
	srv := server.New(store)
	log.Printf("jansvca-idp listening on %s (config=%s users=%d clients=%d)",
		addr, configPath, len(store.Users()), len(store.Clients()))

	mainHandler := srv.FullHandler()
	if introspectAddr != "" {
		log.Printf("jansvca-idp introspection listening on %s", introspectAddr)
		go func() {
			if err := http.ListenAndServe(introspectAddr, srv.IntrospectionHandler()); err != nil {
				log.Fatalf("idp: introspection server: %v", err)
			}
		}()
		mainHandler = srv.Handler()
	}
	if err := http.ListenAndServe(addr, mainHandler); err != nil {
		log.Fatalf("idp: server: %v", err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
