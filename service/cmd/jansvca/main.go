// Command jansvca starts the Schwachstellen-Tracking server. It wires the
// hexagonal modules: projekte uses a WAL event store with an in-memory event
// bus feeding the project read-model projection; schwachstellen persists
// vulnerabilities as JSON files with a Bleve search index (file-based store),
// replacing the former WAL. OAuth2 authentication guards the HTTP API.
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
	"github.com/mwildt/jansvca/service/internal/schwachstellen/migration"
	vulnstore "github.com/mwildt/jansvca/service/internal/schwachstellen/store"
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
	projRead := projapp.NewProjectProjection()
	for _, env := range projStore.All() {
		projRead.Apply(env)
	}
	bus.Subscribe(matchProjekte, func(env eventstore.Envelope) {
		projRead.Apply(env)
	})
	projects := projapp.NewCommandHandler(projStore, bus)

	// Schwachstellen: file-based store (JSON files + Bleve index). If a legacy
	// WAL exists, migrate it once into the store, then the WAL is no longer
	// used for vulnerabilities.
	vulnStore, err := vulnstore.New(dataDir)
	if err != nil {
		log.Fatalf("open schwachstellen store: %v", err)
		return
	}
	if walPath := filepath.Join(dataDir, "schwachstellen.wal"); fileExists(walPath) {
		if legacy, err := eventstore.New(walPath); err == nil {
			if n, err := migration.FromWAL(legacy, vulnStore); err != nil {
				log.Printf("osv migration: %v", err)
			} else if n > 0 {
				log.Printf("osv migration: %d records imported from legacy WAL", n)
			}
			_ = legacy.Close()
			_ = os.Rename(walPath, walPath+".migrated")
		}
	}
	vulnerabilities := vulnapp.NewCommandHandler(vulnStore)
	vulnRead := vulnapp.NewQueryService(vulnStore, projRead)

	handler := server.New(server.Deps{
		Projects:        projects,
		Vulnerabilities: vulnerabilities,
		ProjectRead:     projRead,
		VulnRead:        vulnRead,
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
