// Package tmdbstub serves canned TMDB API responses from JSON fixtures, so
// tests (Go and e2e) never touch the real TMDB API or the network.
//
// Fixture lookup for a request to `/movie/550`:
//   - `<dir>/movie/550.json`
//
// For requests with a `query` param (search), `<dir>/<path>/<query>.json` is
// tried first, where query is lowercased with non [a-z0-9] runs replaced by
// `-`. Paths under `/t/p/` (images) return a tiny placeholder png.
package tmdbstub

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// 1x1 transparent png.
var placeholderPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00,
	0x0d, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49,
	0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}

const notFoundBody = `{"success":false,"status_code":34,"status_message":"The resource you requested could not be found."}`

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

// FixturesDir returns the absolute path to `server/testdata/tmdb`.
func FixturesDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "testdata", "tmdb")
}

// Handler serves fixtures from dir.
func Handler(dir string) http.Handler {
	root := filepath.Clean(dir)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/t/p/") {
			w.Header().Set("Content-Type", "image/png")
			w.Write(placeholderPNG)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("api_key") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"success":false,"status_code":7,"status_message":"Invalid API key."}`))
			return
		}
		candidates := []string{}
		if q := r.URL.Query().Get("query"); q != "" {
			slug := strings.Trim(nonSlugChars.ReplaceAllString(strings.ToLower(q), "-"), "-")
			candidates = append(candidates, filepath.Join(root, r.URL.Path, slug+".json"))
		}
		candidates = append(candidates, filepath.Join(root, r.URL.Path+".json"))
		for _, f := range candidates {
			// Never serve anything outside of root.
			if !strings.HasPrefix(f, root+string(filepath.Separator)) {
				continue
			}
			if b, err := os.ReadFile(f); err == nil {
				w.Write(b)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(notFoundBody))
	})
}

// NewServer starts a stub TMDB server serving FixturesDir, closed when the
// test finishes.
func NewServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(Handler(FixturesDir()))
	t.Cleanup(srv.Close)
	return srv
}
