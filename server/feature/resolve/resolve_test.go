package resolve

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"testing"

	"github.com/sbondCo/Watcharr/internal/testutil/tmdbstub"
	"github.com/sbondCo/Watcharr/media/tmdb"
	"github.com/sbondCo/Watcharr/util/safefetch"
)

// fakeSites serves the saved pages, no network.
func fakeSites(t *testing.T) *safefetch.FakeSites {
	t.Helper()
	pages := map[string]string{
		"letterboxd.com/film/fight-club/":        "letterboxd_fight_club.html",
		"letterboxd.com/film/breaking-bad/":      "letterboxd_link_only.html",
		"letterboxd.com/film/challenge/":         "letterboxd_no_id.html",
		"www.rottentomatoes.com/m/the_thing":     "rt_the_thing.html",
		"www.rottentomatoes.com/tv/breaking_bad": "rt_breaking_bad_og.html",
	}
	s := safefetch.NewFakeSites(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Host + r.URL.Path
		switch key {
		case "boxd.it/29Kg":
			http.Redirect(w, r, "https://letterboxd.com/film/fight-club/", http.StatusFound)
			return
		case "boxd.it/evil":
			http.Redirect(w, r, "https://evil.example/", http.StatusFound)
			return
		case "letterboxd.com/film/blocked/", "www.rottentomatoes.com/m/the_thing_1982":
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if f, ok := pages[key]; ok {
			w.Write([]byte(fixture(t, f)))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(s.Close)
	return s
}

func newResolver(t *testing.T) *Resolver {
	t.Helper()
	stub := tmdbstub.NewServer(t)
	client := tmdb.NewTMDB("test-key", stub.URL, stub.URL+"/t/p")
	return New(client, safefetch.New(FetchDomains, fakeSites(t).Transport()))
}

func TestResolveExactMatches(t *testing.T) {
	r := newResolver(t)
	cases := []struct {
		url  string
		want Candidate
	}{
		{"https://www.themoviedb.org/movie/550-fight-club", Candidate{550, "movie", "Fight Club", 1999, "/fight-club.jpg"}},
		{"https://www.themoviedb.org/tv/1396", Candidate{1396, "tv", "Breaking Bad", 2008, "/breaking-bad.jpg"}},
		{"https://m.imdb.com/title/tt0137523/?ref_=x", Candidate{550, "movie", "Fight Club", 1999, "/fight-club.jpg"}},
		{"https://www.imdb.com/title/tt0903747/", Candidate{1396, "tv", "Breaking Bad", 2008, "/breaking-bad.jpg"}},
		{"https://letterboxd.com/film/fight-club/", Candidate{550, "movie", "Fight Club", 1999, "/fight-club.jpg"}},
		// Diary path is stripped down to the film.
		{"https://letterboxd.com/someuser/film/fight-club/2/", Candidate{550, "movie", "Fight Club", 1999, "/fight-club.jpg"}},
		// Short link, followed through an allowed redirect.
		{"https://boxd.it/29Kg", Candidate{550, "movie", "Fight Club", 1999, "/fight-club.jpg"}},
		// Page with only a TMDB link.
		{"https://letterboxd.com/film/breaking-bad/", Candidate{1396, "tv", "Breaking Bad", 2008, "/breaking-bad.jpg"}},
	}
	for _, c := range cases {
		res, err := r.Resolve(context.Background(), Request{URL: c.url})
		if err != nil {
			t.Errorf("%s: %v", c.url, err)
			continue
		}
		if len(res.Candidates) != 1 || res.Candidates[0] != c.want {
			t.Errorf("%s: got %+v, want exactly %+v", c.url, res.Candidates, c.want)
		}
	}
}

func TestResolveRottenTomatoesGivesCandidates(t *testing.T) {
	r := newResolver(t)
	// JSON-LD title + year, TMDB search, capped at 5.
	res, err := r.Resolve(context.Background(), Request{URL: "https://www.rottentomatoes.com/m/the_thing"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Candidates) != maxCandidates || res.Candidates[0].TmdbID != 1091 || res.Candidates[0].Year != 1982 {
		t.Fatalf("rt candidates: %+v", res.Candidates)
	}
	// Page blocked, falls back to the slug ("the_thing_1982").
	res, err = r.Resolve(context.Background(), Request{URL: "https://www.rottentomatoes.com/m/the_thing_1982"})
	if err != nil || len(res.Candidates) == 0 || res.Candidates[0].TmdbID != 1091 {
		t.Fatalf("rt slug fallback: %+v %v", res.Candidates, err)
	}
	// TV, title from og:title.
	res, err = r.Resolve(context.Background(), Request{URL: "https://www.rottentomatoes.com/tv/breaking_bad"})
	if err != nil || len(res.Candidates) != 1 || res.Candidates[0].TmdbID != 1396 || res.Candidates[0].MediaType != "tv" {
		t.Fatalf("rt tv: %+v %v", res.Candidates, err)
	}
}

func TestResolveErrors(t *testing.T) {
	r := newResolver(t)
	cases := []struct {
		url  string
		want error
	}{
		{"https://example.com/film/x", ErrUnsupportedURL},
		{"https://fakeimdb.com/title/tt0137523/", ErrUnsupportedURL},
		{"not a url at all", ErrUnsupportedURL},
		// Letterboxd refused the request / page without an id.
		{"https://letterboxd.com/film/blocked/", ErrFetchBlocked},
		{"https://letterboxd.com/film/challenge/", ErrFetchBlocked},
		// Short link redirecting off the allowlist.
		{"https://boxd.it/evil", ErrFetchBlocked},
		// Letterboxd 404, IMDb/TMDB ids TMDB doesn't know.
		{"https://letterboxd.com/film/does-not-exist/", ErrNoMatch},
		{"https://www.imdb.com/title/tt9999999/", ErrNoMatch},
		{"https://www.themoviedb.org/movie/99999999", ErrNoMatch},
	}
	for _, c := range cases {
		if _, err := r.Resolve(context.Background(), Request{URL: c.url}); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", c.url, err, c.want)
		}
	}
}

func TestResolveHintPicksMediaType(t *testing.T) {
	r := newResolver(t)
	cands, err := r.byIMDbID("tt0137523", "tv")
	// Only a movie result exists, the hint doesn't drop everything.
	if err != nil || len(cands) != 1 || cands[0].MediaType != "movie" {
		t.Fatalf("hint with no match: %+v %v", cands, err)
	}
	if !slices.Contains(FetchDomains, "letterboxd.com") || slices.Contains(FetchDomains, "imdb.com") {
		t.Fatalf("fetch allowlist: %v", FetchDomains)
	}
}
