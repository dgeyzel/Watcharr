package resolve

import (
	"errors"
	"testing"
)

func TestParseURL(t *testing.T) {
	cases := []struct {
		in   string
		want ParsedURL
	}{
		// TMDB (with and without slug, www, language prefix, query, trailing).
		{"https://www.themoviedb.org/movie/550-fight-club", ParsedURL{SourceTMDB, "movie", "550", ""}},
		{"https://www.themoviedb.org/movie/550", ParsedURL{SourceTMDB, "movie", "550", ""}},
		{"https://themoviedb.org/tv/1396-breaking-bad/", ParsedURL{SourceTMDB, "tv", "1396", ""}},
		{"https://www.themoviedb.org/movie/550-fight-club?language=en-US#cast", ParsedURL{SourceTMDB, "movie", "550", ""}},
		{"https://www.themoviedb.org/movie/550-fight-club/cast", ParsedURL{SourceTMDB, "movie", "550", ""}},
		{"themoviedb.org/movie/550", ParsedURL{SourceTMDB, "movie", "550", ""}},
		{"https://WWW.THEMOVIEDB.ORG/movie/550", ParsedURL{SourceTMDB, "movie", "550", ""}},
		// IMDb (www, m., trailing paths, query strings, uppercase).
		{"https://www.imdb.com/title/tt0137523/", ParsedURL{SourceIMDb, "", "tt0137523", ""}},
		{"https://imdb.com/title/tt0137523", ParsedURL{SourceIMDb, "", "tt0137523", ""}},
		{"https://m.imdb.com/title/tt0137523/?ref_=nv_sr_srsg_0", ParsedURL{SourceIMDb, "", "tt0137523", ""}},
		{"https://www.imdb.com/title/tt0903747/episodes/?season=1", ParsedURL{SourceIMDb, "", "tt0903747", ""}},
		{"http://www.imdb.com/title/TT0137523/maindetails", ParsedURL{SourceIMDb, "", "tt0137523", ""}},
		{"www.imdb.com/title/tt0137523", ParsedURL{SourceIMDb, "", "tt0137523", ""}},
		// Letterboxd (film pages, sub pages, diary/review paths).
		{"https://letterboxd.com/film/fight-club/", ParsedURL{SourceLetterboxd, "", "", "fight-club"}},
		{"https://letterboxd.com/film/fight-club", ParsedURL{SourceLetterboxd, "", "", "fight-club"}},
		{"https://www.letterboxd.com/film/fight-club/reviews/by/activity/", ParsedURL{SourceLetterboxd, "", "", "fight-club"}},
		{"https://letterboxd.com/someuser/film/fight-club/", ParsedURL{SourceLetterboxd, "", "", "fight-club"}},
		{"https://letterboxd.com/someuser/film/fight-club/1/", ParsedURL{SourceLetterboxd, "", "", "fight-club"}},
		{"https://letterboxd.com/film/the-thing/?utm_source=x", ParsedURL{SourceLetterboxd, "", "", "the-thing"}},
		// boxd.it short links (case sensitive codes).
		{"https://boxd.it/29Kg", ParsedURL{SourceLetterboxdLink, "", "", "29Kg"}},
		{"boxd.it/29Kg", ParsedURL{SourceLetterboxdLink, "", "", "29Kg"}},
		// Rotten Tomatoes.
		{"https://www.rottentomatoes.com/m/the_thing", ParsedURL{SourceRottenTomatoes, "movie", "", "the_thing"}},
		{"https://www.rottentomatoes.com/m/the_thing_1982/", ParsedURL{SourceRottenTomatoes, "movie", "", "the_thing_1982"}},
		{"https://rottentomatoes.com/tv/breaking_bad/s01", ParsedURL{SourceRottenTomatoes, "tv", "", "breaking_bad"}},
	}
	for _, c := range cases {
		got, err := ParseURL(c.in)
		if err != nil || got != c.want {
			t.Errorf("ParseURL(%q) = %+v, %v; want %+v", c.in, got, err, c.want)
		}
	}
}

func TestParseURLRejects(t *testing.T) {
	for _, in := range []string{
		"",
		"fight club",
		"https://example.com/movie/550",
		// Lookalike and suffixed hosts.
		"https://fakeimdb.com/title/tt0137523/",
		"https://imdb.com.evil.net/title/tt0137523/",
		"https://notletterboxd.com/film/fight-club/",
		"https://letterboxd.com.evil.net/film/fight-club/",
		"https://myrottentomatoes.com/m/the_thing",
		"https://xboxd.it/29Kg",
		// Supported sites, unsupported pages.
		"https://www.imdb.com/name/nm0000093/",
		"https://www.imdb.com/title/abc/",
		"https://www.themoviedb.org/person/287",
		"https://www.themoviedb.org/movie/abc",
		"https://letterboxd.com/someuser/",
		"https://letterboxd.com/films/popular/",
		"https://boxd.it/",
		"https://boxd.it/a/b",
		"https://www.rottentomatoes.com/celebrity/brad_pitt",
		// Other schemes and userinfo.
		"ftp://imdb.com/title/tt0137523/",
		"javascript:alert(1)",
		"https://user:pw@www.imdb.com/title/tt0137523/",
	} {
		if got, err := ParseURL(in); !errors.Is(err, ErrUnsupportedURL) {
			t.Errorf("ParseURL(%q) = %+v, %v; want ErrUnsupportedURL", in, got, err)
		}
	}
}
