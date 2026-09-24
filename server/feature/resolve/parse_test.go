package resolve

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	b, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "testdata", "resolve", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestParseLetterboxd(t *testing.T) {
	cases := []struct {
		file      string
		mediaType string
		id        int
		ok        bool
	}{
		{"letterboxd_fight_club.html", "movie", 550, true},
		// No data-tmdb-* attributes, falls back to the TMDB link.
		{"letterboxd_link_only.html", "tv", 1396, true},
		// Blocked / changed markup.
		{"letterboxd_no_id.html", "", 0, false},
	}
	for _, c := range cases {
		mt, id, ok := ParseLetterboxd(fixture(t, c.file))
		if mt != c.mediaType || id != c.id || ok != c.ok {
			t.Errorf("%s: got (%q, %d, %v), want (%q, %d, %v)", c.file, mt, id, ok, c.mediaType, c.id, c.ok)
		}
	}
}

func TestParseRottenTomatoes(t *testing.T) {
	cases := []struct {
		file string
		want TitleYear
		ok   bool
	}{
		{"rt_the_thing.html", TitleYear{"The Thing", 1982}, true},
		// No usable JSON-LD, og:title "Breaking Bad (2008 - 2013) | Rotten Tomatoes".
		{"rt_breaking_bad_og.html", TitleYear{"Breaking Bad", 2008}, true},
		{"rt_no_metadata.html", TitleYear{}, false},
	}
	for _, c := range cases {
		got, ok := ParseRottenTomatoes(fixture(t, c.file))
		if got != c.want || ok != c.ok {
			t.Errorf("%s: got %+v %v, want %+v %v", c.file, got, ok, c.want, c.ok)
		}
	}
}

func TestTitleYearFromRTSlug(t *testing.T) {
	for slug, want := range map[string]TitleYear{
		"the_thing_1982":    {"the thing", 1982},
		"the_thing":         {"the thing", 0},
		"breaking_bad":      {"breaking bad", 0},
		"blade_runner_2049": {"blade runner", 2049}, // ambiguous, search still finds it
		"1917_2019":         {"1917", 2019},
	} {
		if got := TitleYearFromRTSlug(slug); got != want {
			t.Errorf("%s: got %+v, want %+v", slug, got, want)
		}
	}
}
