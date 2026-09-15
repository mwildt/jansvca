package osv

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client fetches OSV data from the osv.dev data dump bucket. It is configured
// with a base URL (default the public GCS bucket) so tests can point it at an
// httptest server.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// DefaultBaseURL is the public osv.dev GCS bucket serving the aggregated data
// dumps and per-record JSON files.
const DefaultBaseURL = "https://storage.googleapis.com/osv-vulnerabilities"

// NewClient returns an OSV client using the public data dump bucket and a
// reasonable HTTP timeout.
func NewClient() *Client {
	return &Client{
		BaseURL:    DefaultBaseURL,
		HTTPClient: &http.Client{Timeout: 5 * time.Minute},
	}
}

func (c *Client) baseURL() string {
	if c.BaseURL == "" {
		return DefaultBaseURL
	}
	return c.BaseURL
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient == nil {
		return http.DefaultClient
	}
	return c.HTTPClient
}

// FetchAllZip downloads the all.zip archive containing every OSV record and
// returns the parsed records.
func (c *Client) FetchAllZip(ctx context.Context) ([]Record, error) {
	rc, err := c.fetch(ctx, "/all.zip")
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return ParseRecordsFromZip(rc)
}

// FetchEcosystemZip downloads the all.zip archive for a single ecosystem
// (e.g. "Go", "PyPI") and returns the parsed records. The OSV bucket exposes
// gs://osv-vulnerabilities/<ECOSYSTEM>/all.zip with only that ecosystem's
// records, so a sync configured for a small set of ecosystems avoids
// downloading the full global archive.
func (c *Client) FetchEcosystemZip(ctx context.Context, ecosystem string) ([]Record, error) {
	rc, err := c.fetch(ctx, "/"+ecosystem+"/all.zip")
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return ParseRecordsFromZip(rc)
}

// FetchModifiedSince lists the top-level modified_id.csv index and returns the
// entries strictly newer than since. Paths in the top-level index include
// the ecosystem prefix (e.g. "PyPI/PYSEC-2021-123").
func (c *Client) FetchModifiedSince(ctx context.Context, since time.Time) ([]ModifiedEntry, error) {
	rc, err := c.fetch(ctx, "/modified_id.csv")
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return ReadModifiedSince(rc, since)
}

// FetchEcosystemModifiedSince lists the per-ecosystem modified_id.csv index
// and returns the entries strictly newer than since. The per-ecosystem CSV
// omits the ecosystem prefix in the path column (e.g. "PYSEC-2021-123" rather
// than "PyPI/PYSEC-2021-123"); callers must prepend the ecosystem when
// fetching individual records.
func (c *Client) FetchEcosystemModifiedSince(ctx context.Context, ecosystem string, since time.Time) ([]ModifiedEntry, error) {
	rc, err := c.fetch(ctx, "/"+ecosystem+"/modified_id.csv")
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return ReadModifiedSince(rc, since)
}

// FetchRecord downloads a single record JSON by its index path (as returned
// by FetchModifiedSince, e.g. "PyPI/PYSEC-2021-123"). When the path has no
// ecosystem prefix (as returned by FetchEcosystemModifiedSince), pass the
// ecosystem via FetchEcosystemRecord instead.
func (c *Client) FetchRecord(ctx context.Context, path string) (Record, error) {
	rc, err := c.fetch(ctx, "/"+path+".json")
	if err != nil {
		return Record{}, err
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return Record{}, fmt.Errorf("osv: read record %s: %w", path, err)
	}
	return ParseRecord(data)
}

// FetchEcosystemRecord downloads a single record JSON for a known ecosystem.
// id is the bare record id without ecosystem prefix (e.g. "PYSEC-2021-123");
// the client prepends the ecosystem directory.
func (c *Client) FetchEcosystemRecord(ctx context.Context, ecosystem, id string) (Record, error) {
	return c.FetchRecord(ctx, ecosystem+"/"+id)
}

func (c *Client) fetch(ctx context.Context, path string) (io.ReadCloser, error) {
	url := c.baseURL() + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("osv: build request: %w", err)
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("osv: fetch %s: %w", path, err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("osv: fetch %s: status %d", path, resp.StatusCode)
	}
	return resp.Body, nil
}
