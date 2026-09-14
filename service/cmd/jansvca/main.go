// Command jansvca starts the Schwachstellen-Tracking server. It wires the
// hexagonal modules (projekte, schwachstellen) with per-module WAL event
// stores, an in-memory event bus feeding the read-model projections, and
// OAuth2 authentication.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mwildt/jansvca/service/internal/auth"
	"github.com/mwildt/jansvca/service/internal/eventstore"
	"github.com/mwildt/jansvca/service/internal/osv"
	syncpkg "github.com/mwildt/jansvca/service/internal/osv/sync"
	projapp "github.com/mwildt/jansvca/service/internal/projekte/application"
	vulnapp "github.com/mwildt/jansvca/service/internal/schwachstellen/application"
	"github.com/mwildt/jansvca/service/internal/server"
)

func main() {
	dataDir := envOr("JANSVCA_DATA_DIR", "./data")
	addr := envOr("JANSVCA_ADDR", ":8080")
	introspectionURL := os.Getenv("JANSVCA_OAUTH2_INTROSPECTION_URL")
	clientID := os.Getenv("JANSVCA_OAUTH2_CLIENT_ID")
	clientSecret := os.Getenv("JANSVCA_OAUTH2_CLIENT_SECRET")

	bus := eventstore.NewBus()

	projStore, err := eventstore.New(filepath.Join(dataDir, "projekte.wal"))
	if err != nil {
		log.Fatalf("open projekte store: %v", err)
	}
	vulnStore, err := eventstore.New(filepath.Join(dataDir, "schwachstellen.wal"))
	if err != nil {
		log.Fatalf("open schwachstellen store: %v", err)
	}

	projRead := projapp.NewProjectProjection()
	matchRead := vulnapp.NewMatchingProjection()

	// Rebuild read models from existing WALs.
	for _, env := range projStore.All() {
		projRead.Apply(env)
		matchRead.Apply(env)
	}
	for _, env := range vulnStore.All() {
		matchRead.Apply(env)
	}

	// Subscribe projections: projekte events feed both the project read model
	// and the matching projection's component model.
	bus.Subscribe(matchProjekte, func(env eventstore.Envelope) {
		projRead.Apply(env)
		matchRead.Apply(env)
	})
	bus.Subscribe(matchSchwachstellen, func(env eventstore.Envelope) {
		matchRead.Apply(env)
	})

	projects := projapp.NewCommandHandler(projStore, bus)
	vulnerabilities := vulnapp.NewCommandHandler(vulnStore, bus)

	handler := server.New(server.Deps{
		Projects:        projects,
		Vulnerabilities: vulnerabilities,
		ProjectRead:     projRead,
		MatchingRead:    matchRead,
	})

	if osvSync := newOSVSync(vulnerabilities, dataDir); osvSync != nil {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() {
			log.Printf("osv sync: initial import started")
			n, err := osvSync.Once(ctx)
			if err != nil {
				log.Printf("osv sync: initial import failed: %v", err)
			} else {
				log.Printf("osv sync: initial import done (%d records)", n)
			}
			osvSync.Run(ctx)
		}()
	}

	if introspectionURL != "" {
		verifier := auth.NewIntrospectionVerifier(introspectionURL, clientID, clientSecret)
		handler = auth.Middleware(verifier, handler)
	}

	log.Printf("jansvca listening on %s (data=%s)", addr, dataDir)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// newOSVSync builds the osv.dev sync unless disabled via JANSVCA_OSV_SYNC=off.
// The fetch base URL and refresh interval are configurable for development.
func newOSVSync(store *vulnapp.CommandHandler, dataDir string) *syncpkg.Sync {
	if os.Getenv("JANSVCA_OSV_SYNC") == "off" {
		return nil
	}
	client := osv.NewClient()
	if v := os.Getenv("JANSVCA_OSV_BASE_URL"); v != "" {
		client.BaseURL = v
	}
	interval := time.Hour
	if v := os.Getenv("JANSVCA_OSV_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			interval = d
		}
	}
	state := syncpkg.NewStateStore(filepath.Join(dataDir, "osv-sync.json"))
	return syncpkg.New(client, store, state, interval)
}

func matchProjekte(env eventstore.Envelope) bool {
	prefix := eventstore.EventType("projekte.")
	return len(env.Type) >= len(prefix) && env.Type[:len(prefix)] == prefix
}

func matchSchwachstellen(env eventstore.Envelope) bool {
	prefix := "schwachstellen."
	return len(env.Type) >= len(prefix) && env.Type[:len(prefix)] == eventstore.EventType(prefix)
}
