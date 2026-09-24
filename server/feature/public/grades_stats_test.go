package public_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/sbondCo/Watcharr/feature/public"
	"github.com/sbondCo/Watcharr/internal/testutil/testserver"
)

type gradedItem struct {
	Title  string  `json:"title"`
	Status string  `json:"status"`
	Grade  *string `json:"grade"`
}

func (f *fixture) gradedList(t *testing.T, path string) []gradedItem {
	t.Helper()
	code, body := f.get(path)
	if code != http.StatusOK {
		t.Fatalf("GET %s: %d %s", path, code, body)
	}
	var r struct{ Results []gradedItem }
	if err := json.Unmarshal(body, &r); err != nil {
		t.Fatal(err)
	}
	return r.Results
}

func gradeOf(items []gradedItem, title string) string {
	for _, i := range items {
		if i.Title == title {
			if i.Grade == nil {
				return "null"
			}
			return *i.Grade
		}
	}
	return "missing"
}

func TestPublicGrades(t *testing.T) {
	f := newFixture(t)
	items := f.gradedList(t, "/api/public/watched")
	for title, want := range map[string]string{
		"Fight Club":      "A",
		"Breaking Bad":    "S",
		"Game of Thrones": "C",
		"Pulp Fiction":    "null", // watched, not rated yet
		// Planned titles have no grade slot, even with a grade saved.
		"Forrest Gump":    "null",
		"Stranger Things": "null",
	} {
		if got := gradeOf(items, title); got != want {
			t.Errorf("%s grade = %s, want %s", title, got, want)
		}
	}
	_, body := f.get("/api/public/watched/movie/13")
	if !strings.Contains(string(body), `"grade":null`) {
		t.Errorf("planned item leaked its grade: %s", body)
	}
}

func TestPublicGradeSort(t *testing.T) {
	f := newFixture(t)
	titlesOf := func(items []gradedItem) []string {
		out := []string{}
		for _, i := range items {
			out = append(out, i.Title)
		}
		return out
	}
	// S -> F, then watched but not rated, then planned (alphabetical within).
	want := []string{"Breaking Bad", "Fight Club", "Game of Thrones", "Pulp Fiction", "Forrest Gump", "Stranger Things"}
	if got := titlesOf(f.gradedList(t, "/api/public/watched?sort=GRADE&sortDir=desc")); !slices.Equal(got, want) {
		t.Errorf("grade desc:\n got  %v\n want %v", got, want)
	}
	// Default direction is desc (S first).
	if got := titlesOf(f.gradedList(t, "/api/public/watched?sort=GRADE")); !slices.Equal(got, want) {
		t.Errorf("grade default:\n got  %v\n want %v", got, want)
	}
	// Ascending flips the graded ones only, ungraded stay last.
	wantAsc := []string{"Game of Thrones", "Fight Club", "Breaking Bad", "Pulp Fiction", "Forrest Gump", "Stranger Things"}
	if got := titlesOf(f.gradedList(t, "/api/public/watched?sort=GRADE&sortDir=asc")); !slices.Equal(got, wantAsc) {
		t.Errorf("grade asc:\n got  %v\n want %v", got, wantAsc)
	}
}

func TestPublicGradeFilter(t *testing.T) {
	f := newFixture(t)
	cases := map[string][]string{
		"A":      {"Fight Club"},
		"S,A":    {"Breaking Bad", "Fight Club"},
		"B":      {}, // only the planned Forrest Gump has B, never shown
		"none":   {"Pulp Fiction"},
		"C,none": {"Game of Thrones", "Pulp Fiction"},
		// Invalid values are ignored (no filter).
		"a":    visibleTitles,
		"A%2B": visibleTitles, // A+
	}
	for q, want := range cases {
		got := []string{}
		for _, i := range f.gradedList(t, "/api/public/watched?grade="+q) {
			got = append(got, i.Title)
		}
		if !slices.Equal(sorted(got), sorted(want)) {
			t.Errorf("grade=%s:\n got  %v\n want %v", q, sorted(got), sorted(want))
		}
	}
}

func TestPublicStats(t *testing.T) {
	f := newFixture(t)
	code, body := f.get("/api/public/stats")
	if code != http.StatusOK {
		t.Fatalf("stats: %d %s", code, body)
	}
	var st public.StatsResponse
	if err := json.Unmarshal(body, &st); err != nil {
		t.Fatal(err)
	}
	if st.Totals != (public.StatsTotals{Titles: 6, Movies: 3, Shows: 3}) {
		t.Errorf("totals: %+v", st.Totals)
	}
	if st.ByStatus != (public.StatsByStatus{Finished: 2, Watching: 2, Planned: 2}) {
		t.Errorf("byStatus: %+v", st.ByStatus)
	}
	// Planned (Forrest Gump B) is never counted, hidden (Matrix F) and on hold
	// (Inception D) are excluded.
	if st.Grades != (public.StatsGrades{S: 1, A: 1, C: 1, Unrated: 1}) {
		t.Errorf("grades: %+v", st.Grades)
	}
	if len(st.AddedPerMonth) != 12 {
		t.Fatalf("addedPerMonth has %d months", len(st.AddedPerMonth))
	}
	thisMonth := time.Now().UTC().Format("2006-01")
	last := st.AddedPerMonth[11]
	if last.Month != thisMonth || last.Count != 6 {
		t.Errorf("this month: %+v, want %s 6", last, thisMonth)
	}
	for _, m := range st.AddedPerMonth[:11] {
		if m.Count != 0 {
			t.Errorf("month %s should be 0, got %d", m.Month, m.Count)
		}
	}
	wantDecades := []public.DecadeCount{{Decade: 1990, Count: 3}, {Decade: 2000, Count: 1}, {Decade: 2010, Count: 2}}
	if !slices.Equal(st.ByDecade, wantDecades) {
		t.Errorf("byDecade: %+v", st.ByDecade)
	}
	wantGenres := []public.NameCount{
		{Name: "Drama", Count: 4}, {Name: "Crime", Count: 2}, {Name: "Sci-Fi & Fantasy", Count: 2},
		{Name: "Comedy", Count: 1}, {Name: "Mystery", Count: 1}, {Name: "Romance", Count: 1}, {Name: "Thriller", Count: 1},
	}
	if !slices.Equal(st.TopGenres, wantGenres) {
		t.Errorf("topGenres: %+v", st.TopGenres)
	}
	wantTags := []public.TagCount{{ID: f.privateTagID, Name: "Drafts", Count: 0}, {ID: f.tagID, Name: "Favourites", Count: 2}}
	if !slices.Equal(st.Tags, wantTags) {
		t.Errorf("tags: %+v", st.Tags)
	}
	// Only Fight Club is a finished (visible) movie: 139 min.
	if st.FinishedMovieHours != 2.3 {
		t.Errorf("finishedMovieHours = %v, want 2.3", st.FinishedMovieHours)
	}
	for _, k := range []string{`"rating"`, `"average"`, `"voteAverage"`} {
		if strings.Contains(string(body), k) {
			t.Errorf("stats contains %s: %s", k, body)
		}
	}
}

func TestPublicStatsEmptySite(t *testing.T) {
	for _, withAdmin := range []bool{false, true} {
		s := testserver.New(t)
		if withAdmin {
			s.SeedAdmin()
		}
		rec := s.Do(http.MethodGet, "/api/public/stats", nil, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("empty stats (admin=%v): %d %s", withAdmin, rec.Code, rec.Body.String())
		}
		st := testserver.DecodeJSON[public.StatsResponse](t, rec)
		if st.Totals.Titles != 0 || len(st.AddedPerMonth) != 12 || st.ByDecade == nil || st.TopGenres == nil || st.Tags == nil {
			t.Errorf("empty stats (admin=%v): %+v", withAdmin, st)
		}
	}
}

func TestPublicStatsKeys(t *testing.T) {
	f := newFixture(t)
	_, body := f.get("/api/public/stats")
	want := []string{"addedPerMonth", "byDecade", "byStatus", "finishedMovieHours", "grades", "tags", "topGenres", "totals"}
	if got := keysOf(t, body); !slices.Equal(got, want) {
		t.Errorf("stats keys:\n got  %v\n want %v", got, want)
	}
	var st map[string]json.RawMessage
	json.Unmarshal(body, &st)
	if got := keysOf(t, st["grades"]); !slices.Equal(got, []string{"A", "B", "C", "D", "F", "S", "unrated"}) {
		t.Errorf("grades keys: %v", got)
	}
}
