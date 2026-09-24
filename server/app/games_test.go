package app_test

import (
	"net/http"
	"testing"

	"github.com/sbondCo/Watcharr/internal/testutil/testserver"
)

// Games were removed (movies and tv only).
func TestGamesAreRejected(t *testing.T) {
	s := testserver.New(t)
	token := s.SeedAdmin()
	cases := []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, "/api/search?query=zelda&type=game", nil},
		{http.MethodGet, "/api/discover?type=game&filter=trending", nil},
		{http.MethodPost, "/api/watched", map[string]any{"contentType": "game", "igdbId": 1, "status": "FINISHED"}},
	}
	for _, c := range cases {
		if rec := s.Do(c.method, c.path, c.body, token); rec.Code != http.StatusBadRequest {
			t.Errorf("%s %s: expected 400, got %d: %s", c.method, c.path, rec.Code, rec.Body.String())
		}
	}
}

func TestFollowsTableIsDropped(t *testing.T) {
	s := testserver.New(t)
	if s.DB.Migrator().HasTable("follows") {
		t.Fatal("follows table should be dropped by migration")
	}
}
