package app_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/sbondCo/Watcharr/app"
	"github.com/sbondCo/Watcharr/internal/testutil/testserver"
	"github.com/sbondCo/Watcharr/util/safefetch"
)

func newResolveServer(t *testing.T) (*testserver.Server, string) {
	t.Helper()
	sites := safefetch.NewFakeSites(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Host + r.URL.Path {
		case "letterboxd.com/film/fight-club/":
			w.Write([]byte(`<body data-tmdb-type="movie" data-tmdb-id="550"></body>`))
		case "letterboxd.com/film/blocked/":
			w.WriteHeader(http.StatusForbidden)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(sites.Close)
	s := testserver.NewWithOptions(t, app.Options{FetchTransport: sites.Transport()})
	return s, s.SeedAdmin()
}

type candidates struct {
	Candidates []struct {
		TmdbID    int    `json:"tmdbId"`
		MediaType string `json:"mediaType"`
		Title     string `json:"title"`
		Year      int    `json:"year"`
	} `json:"candidates"`
}

func TestResolveEndpoint(t *testing.T) {
	s, token := newResolveServer(t)
	cases := []struct {
		url    string
		status int
		tmdbID int
	}{
		{"https://www.imdb.com/title/tt0137523/", http.StatusOK, 550},
		{"https://www.themoviedb.org/movie/550", http.StatusOK, 550},
		{"https://letterboxd.com/film/fight-club/", http.StatusOK, 550},
		{"https://letterboxd.com/film/blocked/", http.StatusUnprocessableEntity, 0},
		{"https://fakeimdb.com/title/tt0137523/", http.StatusBadRequest, 0},
		{"https://example.com/", http.StatusBadRequest, 0},
		{"https://www.imdb.com/title/tt9999999/", http.StatusNotFound, 0},
	}
	for _, c := range cases {
		rec := s.Do(http.MethodPost, "/api/import/resolve", map[string]string{"url": c.url}, token)
		if rec.Code != c.status {
			t.Errorf("%s: status %d, want %d (%s)", c.url, rec.Code, c.status, rec.Body.String())
			continue
		}
		if c.status != http.StatusOK {
			// Errors carry a message for the admin.
			if msg := testserver.DecodeJSON[struct{ Error string }](t, rec).Error; msg == "" {
				t.Errorf("%s: no error message", c.url)
			}
			continue
		}
		got := testserver.DecodeJSON[candidates](t, rec)
		if len(got.Candidates) != 1 || got.Candidates[0].TmdbID != c.tmdbID {
			t.Errorf("%s: candidates %+v", c.url, got.Candidates)
		}
	}
	if rec := s.Do(http.MethodPost, "/api/import/resolve", map[string]string{}, token); rec.Code != http.StatusBadRequest {
		t.Errorf("missing url: %d", rec.Code)
	}
	// Admin only.
	if rec := s.Do(http.MethodPost, "/api/import/resolve", map[string]string{"url": "https://www.imdb.com/title/tt0137523/"}, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous resolve: %d", rec.Code)
	}
}

// A failed import row fixed by pasting a url: re-importing with the resolved
// tmdbId keeps the row's status, dates, review and numeric rating, and the
// tier stays empty (D4).
func TestFixFailedImportByURL(t *testing.T) {
	s, token := newResolveServer(t)
	row := map[string]any{
		"name":         "Some Title The Search Can't Find",
		"status":       "WATCHING",
		"rating":       8,
		"thoughts":     "Imported review.",
		"datesWatched": []string{"2020-05-01T00:00:00Z"},
	}
	rec := s.Do(http.MethodPost, "/api/import", row, token)
	if rec.Code != http.StatusOK || testserver.DecodeJSON[struct{ Type string }](t, rec).Type != "IMPORT_NOTFOUND" {
		t.Fatalf("first import should be NOTFOUND: %d %s", rec.Code, rec.Body.String())
	}

	rec = s.Do(http.MethodPost, "/api/import/resolve", map[string]string{"url": "https://www.imdb.com/title/tt0084787/"}, token)
	cands := testserver.DecodeJSON[candidates](t, rec)
	if len(cands.Candidates) != 1 || cands.Candidates[0].TmdbID != 1091 || cands.Candidates[0].Title != "The Thing" || cands.Candidates[0].Year != 1982 {
		t.Fatalf("resolve: %s", rec.Body.String())
	}
	// Nothing is imported by resolving.
	if _, ok := allWatched(t, s, token)[1091]; ok {
		t.Fatal("resolve imported something")
	}

	row["tmdbId"] = cands.Candidates[0].TmdbID
	row["type"] = cands.Candidates[0].MediaType
	rec = s.Do(http.MethodPost, "/api/import", row, token)
	if rec.Code != http.StatusOK || testserver.DecodeJSON[struct{ Type string }](t, rec).Type != "IMPORT_SUCCESS" {
		t.Fatalf("fixed import: %d %s", rec.Code, rec.Body.String())
	}

	rec = s.Do(http.MethodGet, "/api/watched", nil, token)
	for _, w := range testserver.DecodeJSON[[]struct {
		Status    string    `json:"status"`
		Rating    float64   `json:"rating"`
		Thoughts  string    `json:"thoughts"`
		Tier      *string   `json:"tier"`
		Hidden    bool      `json:"hidden"`
		CreatedAt time.Time `json:"createdAt"`
		Content   struct {
			TmdbID int `json:"tmdbId"`
		} `json:"content"`
	}](t, rec) {
		if w.Content.TmdbID != 1091 {
			continue
		}
		if w.Status != "WATCHING" || w.Rating != 8 || w.Thoughts != "Imported review." || w.Tier != nil || w.Hidden {
			t.Fatalf("fixed row lost data: %+v", w)
		}
		if !w.CreatedAt.Equal(time.Date(2020, 5, 1, 0, 0, 0, 0, time.UTC)) {
			t.Fatalf("watched date not kept: %v", w.CreatedAt)
		}
		return
	}
	t.Fatal("fixed title not on the list")
}

// The search bar understands the same urls.
func TestSearchByURL(t *testing.T) {
	s, token := newResolveServer(t)
	for _, u := range []string{
		"https://www.themoviedb.org/movie/550", // slug-less TMDB url
		"https://letterboxd.com/film/fight-club/",
		"https://m.imdb.com/title/tt0137523/",
	} {
		rec := s.Do(http.MethodGet, "/api/search?query="+u, nil, token)
		if rec.Code != http.StatusOK {
			t.Errorf("%s: %d %s", u, rec.Code, rec.Body.String())
			continue
		}
		res := testserver.DecodeJSON[struct {
			Results []struct {
				IDs struct {
					TMDB int `json:"tmdb"`
				} `json:"ids"`
			} `json:"results"`
		}](t, rec)
		if len(res.Results) != 1 || res.Results[0].IDs.TMDB != 550 {
			t.Errorf("%s: results %s", u, rec.Body.String())
		}
	}
}
