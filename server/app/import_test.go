package app_test

import (
	"net/http"
	"testing"

	"github.com/sbondCo/Watcharr/internal/testutil/testserver"
)

func importType(t *testing.T, s *testserver.Server, token string, body map[string]any) string {
	t.Helper()
	rec := s.Do(http.MethodPost, "/api/import", body, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("import %v: %d %s", body, rec.Code, rec.Body.String())
	}
	return testserver.DecodeJSON[struct {
		Type string `json:"type"`
	}](t, rec).Type
}

// Movie csv rows: an imdb link picks that exact title (not one by name) and
// the row's tier is kept.
func TestImportByImdbIdWithTier(t *testing.T) {
	s := testserver.New(t)
	token := s.SeedAdmin()
	// The name would match a different title, the imdb id wins.
	got := importType(t, s, token, map[string]any{
		"name": "Fight Club", "type": "movie", "imdbId": "tt0084787", "imdbStrict": true, "tier": "A",
	})
	if got != "IMPORT_SUCCESS" {
		t.Fatalf("import type = %s, want IMPORT_SUCCESS", got)
	}
	all := allWatched(t, s, token)
	if _, ok := all[550]; ok {
		t.Errorf("imported Fight Club by name, want the imdb id's title")
	}
	w, ok := all[1091]
	if !ok {
		t.Fatalf("The Thing (1091) not imported: %+v", all)
	}
	if tierStr(w.Tier) != "A" {
		t.Errorf("tier = %s, want A", tierStr(w.Tier))
	}
}

// An unknown imdb id with imdbStrict fails (so the admin can fix it) instead
// of guessing by name. Without it the old name fallback still happens.
func TestImportImdbStrictNoNameFallback(t *testing.T) {
	s := testserver.New(t)
	token := s.SeedAdmin()
	body := map[string]any{"name": "The Thing", "year": 1982, "type": "movie", "imdbId": "tt9999999"}

	body["imdbStrict"] = true
	if got := importType(t, s, token, body); got != "IMPORT_NOTFOUND" {
		t.Fatalf("strict: import type = %s, want IMPORT_NOTFOUND", got)
	}
	if n := len(allWatched(t, s, token)); n != 0 {
		t.Fatalf("strict: %d watched added, want 0", n)
	}

	body["imdbStrict"] = false
	if got := importType(t, s, token, body); got != "IMPORT_SUCCESS" {
		t.Fatalf("not strict: import type = %s, want IMPORT_SUCCESS by name", got)
	}
	if _, ok := allWatched(t, s, token)[1091]; !ok {
		t.Fatalf("not strict: The Thing (1091) not imported by name")
	}
}
