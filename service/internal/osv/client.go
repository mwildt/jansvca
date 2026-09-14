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

// FetchModifiedSince lists the modified_id.csv index and returns the entries
// strictly newer than since.
func (c *Client) FetchModifiedSince(ctx context.Context, since time.Time) ([]ModifiedEntry, error) {
	rc, err := c.fetch(ctx, "/modified_id.csv")
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return ReadModifiedSince(rc, since)
}

// FetchRecord downloads a single record JSON by its index path (as returned by
// FetchModifiedSince, e.g. "PyPI/PYSEC-2021-123").
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
