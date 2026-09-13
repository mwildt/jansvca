// Package server wires the hexagonal modules into an HTTP server. It is the
// driving adapter (REST) for both the projekte and schwachstellen modules.
package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/mwildt/jansvca/service/internal/eventstore"
	projapp "github.com/mwildt/jansvca/service/internal/projekte/application"
	"github.com/mwildt/jansvca/service/internal/sbom"
	vulnapp "github.com/mwildt/jansvca/service/internal/schwachstellen/application"
)

// Deps bundles the collaborators required by the HTTP server.
type Deps struct {
	Projects        *projapp.CommandHandler
	Vulnerabilities *vulnapp.CommandHandler
	ProjectRead     *projapp.ProjectProjection
	MatchingRead    *vulnapp.MatchingProjection
}

// New builds and returns the mux serving all module endpoints.
func New(d Deps) http.Handler {
	mux := http.NewServeMux()

	registerProjects(mux, d)
	registerVulnerabilities(mux, d)
	registerMatching(mux, d)

	return mux
}

func registerProjects(mux *http.ServeMux, d Deps) {
	mux.HandleFunc("GET /api/projects", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, d.ProjectRead.All())
	})
	mux.HandleFunc("POST /api/projects", func(w http.ResponseWriter, r *http.Request) {
		var b struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		if err := d.Projects.CreateProject(b.ID, b.Name, b.Description); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, d.ProjectRead.Get(b.ID))
	})
	mux.HandleFunc("GET /api/projects/{id}", func(w http.ResponseWriter, r *http.Request) {
		v := d.ProjectRead.Get(r.PathValue("id"))
		if v == nil {
			writeErr(w, http.StatusNotFound, errors.New("project not found"))
			return
		}
		writeJSON(w, http.StatusOK, v)
	})
	mux.HandleFunc("DELETE /api/projects/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := d.Projects.Delete(r.PathValue("id")); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("PATCH /api/projects/{id}", func(w http.ResponseWriter, r *http.Request) {
		var b struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		id := r.PathValue("id")
		if b.Name != "" {
			if err := d.Projects.Rename(id, b.Name); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
		}
		if strings.TrimSpace(b.Description) != "" || r.URL.Query().Get("desc") != "" {
			if err := d.Projects.ChangeDescription(id, b.Description); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
		}
		writeJSON(w, http.StatusOK, d.ProjectRead.Get(id))
	})
	mux.HandleFunc("POST /api/projects/{id}/components", func(w http.ResponseWriter, r *http.Request) {
		var b struct {
			Component string `json:"component"`
			Version   string `json:"version"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		id := r.PathValue("id")
		if err := d.Projects.AddComponent(id, b.Component, b.Version); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, d.ProjectRead.Get(id))
	})
	mux.HandleFunc("DELETE /api/projects/{id}/components/{component}", func(w http.ResponseWriter, r *http.Request) {
		if err := d.Projects.RemoveComponent(r.PathValue("id"), r.PathValue("component")); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("PUT /api/projects/{id}/components/{component}", func(w http.ResponseWriter, r *http.Request) {
		var b struct {
			Version string `json:"version"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		id := r.PathValue("id")
		if err := d.Projects.UpdateComponentVersion(id, r.PathValue("component"), b.Version); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, d.ProjectRead.Get(id))
	})
	mux.HandleFunc("POST /api/projects/{id}/sbom", func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil || r.ContentLength == 0 {
			writeErr(w, http.StatusBadRequest, errors.New("empty request body"))
			return
		}
		body, err := readBody(w, r)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		inputs, err := sbom.Parse(body)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		id := r.PathValue("id")
		n, err := d.Projects.ImportComponents(id, inputs)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"imported":   n,
			"project":    d.ProjectRead.Get(id),
			"components": len(inputs),
		})
	})
}

func registerVulnerabilities(mux *http.ServeMux, d Deps) {
	mux.HandleFunc("GET /api/vulnerabilities", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, d.MatchingRead.AllVulnerabilities())
	})
	mux.HandleFunc("POST /api/vulnerabilities", func(w http.ResponseWriter, r *http.Request) {
		var b struct {
			ID          string  `json:"id"`
			Identifier  string  `json:"identifier"`
			Title       string  `json:"title"`
			Description string  `json:"description"`
			CVSS        float64 `json:"cvss"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		if err := d.Vulnerabilities.Create(b.ID, b.Identifier, b.Title, b.Description, b.CVSS); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, d.MatchingRead.AllVulnerabilities())
	})
	mux.HandleFunc("DELETE /api/vulnerabilities/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := d.Vulnerabilities.Delete(r.PathValue("id")); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /api/vulnerabilities/{id}/affected-ranges", func(w http.ResponseWriter, r *http.Request) {
		var b struct {
			Component    string `json:"component"`
			VersionRange string `json:"version_range"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		id := r.PathValue("id")
		if err := d.Vulnerabilities.AddAffectedRange(id, b.Component, b.VersionRange); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, d.MatchingRead.AllVulnerabilities())
	})
	mux.HandleFunc("DELETE /api/vulnerabilities/{id}/affected-ranges/{component}", func(w http.ResponseWriter, r *http.Request) {
		if err := d.Vulnerabilities.RemoveAffectedRange(r.PathValue("id"), r.PathValue("component")); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func registerMatching(mux *http.ServeMux, d Deps) {
	mux.HandleFunc("GET /api/projects/{id}/matches", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, d.MatchingRead.Matches(r.PathValue("id")))
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func decodeJSON(r *http.Request, v any) error {
	if r.Body == nil || r.ContentLength == 0 {
		return errors.New("empty request body")
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func readBody(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	return io.ReadAll(http.MaxBytesReader(w, r.Body, 10<<20))
}

// Ensure eventstore import is used (envelope types referenced transitively).
var _ eventstore.Envelope
