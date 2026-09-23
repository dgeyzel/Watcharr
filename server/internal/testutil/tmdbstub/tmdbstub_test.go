package tmdbstub

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func get(t *testing.T, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	Handler(FixturesDir()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func TestServesFixture(t *testing.T) {
	rec := get(t, "/movie/550?api_key=k&append_to_response=videos")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"title": "Fight Club"`) {
		t.Fatalf("unexpected response %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRequiresAPIKey(t *testing.T) {
	if rec := get(t, "/movie/550"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without api_key, got %d", rec.Code)
	}
}

func TestMissingFixtureIs404(t *testing.T) {
	if rec := get(t, "/movie/999999999?api_key=k"); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestNoPathTraversal(t *testing.T) {
	for _, p := range []string{"/../../go.mod?api_key=k", "/%2e%2e/%2e%2e/go?api_key=k"} {
		if rec := get(t, p); rec.Code != http.StatusNotFound {
			t.Fatalf("%s: expected 404, got %d: %s", p, rec.Code, rec.Body.String())
		}
	}
}

func TestServesPlaceholderImage(t *testing.T) {
	rec := get(t, "/t/p/w500/fight-club.jpg")
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("unexpected image response %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
}
