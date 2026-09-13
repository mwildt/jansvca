package web

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// newBackendProxy builds a reverse proxy forwarding to the backend service base
// URL. The BFF mounts it under /api/ and forwards the full path verbatim.
func newBackendProxy(target string) *httputil.ReverseProxy {
	u, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(u)
	orig := proxy.Director
	proxy.Director = func(req *http.Request) {
		orig(req)
		// Keep the original Host so the backend sees its own vhost.
		req.Host = u.Host
	}
	return proxy
}

// existsFile reports whether p is a regular file.
func existsFile(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// existsDir reports whether p is a directory.
func existsDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// serveFile writes the file at p to w.
func serveFile(w http.ResponseWriter, r *http.Request, p string) {
	clean := filepath.Clean(p)
	if !strings.HasPrefix(clean, filepath.Clean(filepath.Dir(p))) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	http.ServeFile(w, r, clean)
}
