package search

import "testing"

func TestGetExtProviderFromURLHostMatching(t *testing.T) {
	s := &Service{}
	cases := []struct {
		name         string
		url          string
		wantProvider string
		wantID       string
	}{
		// Exact hosts.
		{"imdb exact", "https://imdb.com/title/tt0137523/", "imdb", "tt0137523"},
		{"tmdb exact", "https://themoviedb.org/movie/550-fight-club", "movie", "550"},
		// Subdomains.
		{"imdb www", "https://www.imdb.com/title/tt0137523/", "imdb", "tt0137523"},
		{"imdb mobile", "https://m.imdb.com/title/tt0137523/", "imdb", "tt0137523"},
		{"tmdb www", "https://www.themoviedb.org/tv/1396-breaking-bad", "tv", "1396"},
		// Uppercase host (the query is lowercased by the caller, but the
		// helper must not depend on that).
		{"uppercase host", "https://WWW.IMDB.COM/title/tt0137523/", "imdb", "tt0137523"},
		// Hosts with ports.
		{"imdb port", "https://www.imdb.com:443/title/tt0137523/", "imdb", "tt0137523"},
		{"tmdb port", "https://themoviedb.org:8443/movie/550-fight-club", "movie", "550"},
		// Lookalike prefixes.
		{"fake imdb", "https://fakeimdb.com/title/tt0137523/", "", ""},
		{"fake tmdb", "https://notthemoviedb.org/movie/550-fight-club", "", ""},
		{"fake imdb port", "https://fakeimdb.com:443/title/tt0137523/", "", ""},
		// Suffixed hosts.
		{"suffixed imdb", "https://imdb.com.evil.net/title/tt0137523/", "", ""},
		{"suffixed tmdb", "https://www.themoviedb.org.evil.net/movie/550-fight-club", "", ""},
		// Other / not urls.
		{"other site", "https://example.com/title/tt0137523/", "", ""},
		{"not a url", "fight club", "", ""},
	}
	for _, c := range cases {
		p, id := s.getExtProviderFromURL(c.url)
		if p != c.wantProvider || id != c.wantID {
			t.Errorf("%s: getExtProviderFromURL(%q) = (%q, %q), want (%q, %q)",
				c.name, c.url, p, id, c.wantProvider, c.wantID)
		}
	}
}

// The alias cases used to be empty (Go has no fallthrough), so i: and wd:
// never matched.
func TestGetExtProviderFromQueryAliases(t *testing.T) {
	s := &Service{}
	for q, want := range map[string]string{
		"i:tt0137523":    "imdb",
		"imd:tt0137523":  "imdb",
		"imdb:tt0137523": "imdb",
		"wd:q190050":     "wikidata",
		"wdt:q190050":    "wikidata",
		"series:1396":    "tv",
		"game:1":         "",
		"igdb:1":         "",
	} {
		if p, _ := s.getExtProviderFromQuery(q); p != want {
			t.Errorf("getExtProviderFromQuery(%q) provider = %q, want %q", q, p, want)
		}
	}
}
