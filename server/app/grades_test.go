package app_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/internal/testutil/testserver"
)

type watchedRow struct {
	ID      uint    `json:"id"`
	Status  string  `json:"status"`
	Rating  float64 `json:"rating"`
	Grade   *string `json:"grade"`
	Hidden  bool    `json:"hidden"`
	Content struct {
		Title  string `json:"title"`
		TmdbID int    `json:"tmdbId"`
	} `json:"content"`
}

// allWatched gets the admin's full (unpaginated) list, keyed by tmdb id.
func allWatched(t *testing.T, s *testserver.Server, token string) map[int]watchedRow {
	t.Helper()
	rec := s.Do(http.MethodGet, "/api/watched", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/watched: %d %s", rec.Code, rec.Body.String())
	}
	out := map[int]watchedRow{}
	for _, w := range testserver.DecodeJSON[[]watchedRow](t, rec) {
		out[w.Content.TmdbID] = w
	}
	return out
}

func gradeStr(g *string) string {
	if g == nil {
		return "null"
	}
	return *g
}

func TestAdminSetsChangesAndClearsGrade(t *testing.T) {
	s := testserver.New(t)
	token := s.SeedAdmin()
	id := s.AddWatched(token, map[string]any{"contentType": "movie", "tmdbId": 550, "status": "FINISHED", "rating": 8})
	path := fmt.Sprintf("/api/watched/%d", id)

	// Added without a grade: null (never derived from the rating).
	if g := allWatched(t, s, token)[550].Grade; g != nil {
		t.Fatalf("new item grade = %s, want null", *g)
	}

	steps := []struct {
		body any
		want string
	}{
		{map[string]any{"grade": "A"}, "A"},
		{map[string]any{"grade": "S"}, "S"},
		{map[string]any{"grade": nil}, "null"},
		{map[string]any{"grade": "F"}, "F"},
		{map[string]any{"grade": ""}, "null"},
	}
	for _, st := range steps {
		if rec := s.Do(http.MethodPut, path, st.body, token); rec.Code != http.StatusOK {
			t.Fatalf("PUT %v: %d %s", st.body, rec.Code, rec.Body.String())
		}
		if got := gradeStr(allWatched(t, s, token)[550].Grade); got != st.want {
			t.Fatalf("after PUT %v grade = %s, want %s", st.body, got, st.want)
		}
	}

	// Numeric ratings still work and don't touch the grade.
	s.Do(http.MethodPut, path, map[string]any{"grade": "B"}, token)
	if rec := s.Do(http.MethodPut, path, map[string]any{"rating": 6.5}, token); rec.Code != http.StatusOK {
		t.Fatalf("PUT rating: %d %s", rec.Code, rec.Body.String())
	}
	w := allWatched(t, s, token)[550]
	if w.Rating != 6.5 || gradeStr(w.Grade) != "B" {
		t.Fatalf("after rating change: rating=%v grade=%s, want 6.5 B", w.Rating, gradeStr(w.Grade))
	}
}

func TestInvalidGradesAreRejected(t *testing.T) {
	s := testserver.New(t)
	token := s.SeedAdmin()
	id := s.AddWatched(token, map[string]any{"contentType": "movie", "tmdbId": 550, "status": "FINISHED", "grade": "C"})
	path := fmt.Sprintf("/api/watched/%d", id)
	for _, bad := range []any{"A+", "E", "a", "s", "B-", "SS", 5, true} {
		if rec := s.Do(http.MethodPut, path, map[string]any{"grade": bad}, token); rec.Code != http.StatusBadRequest {
			t.Errorf("PUT grade %v: expected 400, got %d", bad, rec.Code)
		}
		if rec := s.Do(http.MethodPost, "/api/watched", map[string]any{
			"contentType": "movie", "tmdbId": 603, "status": "FINISHED", "grade": bad,
		}, token); rec.Code != http.StatusBadRequest {
			t.Errorf("POST grade %v: expected 400, got %d", bad, rec.Code)
		}
	}
	if got := gradeStr(allWatched(t, s, token)[550].Grade); got != "C" {
		t.Fatalf("grade changed by invalid requests: %s", got)
	}
}

func TestAddWithGradeAndHidden(t *testing.T) {
	s := testserver.New(t)
	token := s.SeedAdmin()
	s.AddWatched(token, map[string]any{"contentType": "movie", "tmdbId": 550, "status": "FINISHED", "grade": "A", "hidden": true})
	s.AddWatched(token, map[string]any{"contentType": "tv", "tmdbId": 1396, "status": "WATCHING"})
	all := allWatched(t, s, token)
	if gradeStr(all[550].Grade) != "A" || !all[550].Hidden {
		t.Errorf("550: %+v", all[550])
	}
	if all[1396].Grade != nil || all[1396].Hidden {
		t.Errorf("1396 should be ungraded and visible: %+v", all[1396])
	}
}

func TestGradeChangeActivity(t *testing.T) {
	s := testserver.New(t)
	token := s.SeedAdmin()
	id := s.AddWatched(token, map[string]any{"contentType": "movie", "tmdbId": 550, "status": "FINISHED"})
	path := fmt.Sprintf("/api/watched/%d", id)
	s.Do(http.MethodPut, path, map[string]any{"grade": "B"}, token)
	s.Do(http.MethodPut, path, map[string]any{"grade": "A"}, token)
	s.Do(http.MethodPut, path, map[string]any{"grade": "A"}, token) // no change, no activity
	s.Do(http.MethodPut, path, map[string]any{"grade": nil}, token)

	rec := s.Do(http.MethodGet, fmt.Sprintf("/api/activity/%d", id), nil, token)
	var acts []struct {
		Type string `json:"type"`
		Data string `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &acts); err != nil {
		t.Fatalf("activity: %v %s", err, rec.Body.String())
	}
	got := []string{}
	for _, a := range acts {
		if a.Type == string(entity.GRADE_CHANGED) {
			got = append(got, a.Data)
		}
	}
	want := []string{`{"new":"B","old":null}`, `{"new":"A","old":"B"}`, `{"new":null,"old":"A"}`}
	if !slices.Equal(got, want) {
		t.Fatalf("grade activity:\n got  %v\n want %v", got, want)
	}
}

// D4: numeric ratings are never mapped to grades, imports arrive ungraded.
func TestImportNeverMapsRatingToGrade(t *testing.T) {
	s := testserver.New(t)
	token := s.SeedAdmin()
	for _, body := range []map[string]any{
		{"tmdbId": 550, "type": "movie", "rating": 10, "status": "FINISHED"},
		{"tmdbId": 1396, "type": "tv", "rating": 9.5, "status": "WATCHING"},
		{"tmdbId": 680, "type": "movie", "rating": 1, "status": "FINISHED"},
	} {
		rec := s.Do(http.MethodPost, "/api/import", body, token)
		if rec.Code != http.StatusOK || !json.Valid(rec.Body.Bytes()) {
			t.Fatalf("import %v: %d %s", body, rec.Code, rec.Body.String())
		}
	}
	all := allWatched(t, s, token)
	for id, rating := range map[int]float64{550: 10, 1396: 9.5, 680: 1} {
		w := all[id]
		if w.Rating != rating {
			t.Errorf("%d rating = %v, want %v (kept as today)", id, w.Rating, rating)
		}
		if w.Grade != nil {
			t.Errorf("%d grade = %s, want null (no mapping)", id, *w.Grade)
		}
		if w.Hidden {
			t.Errorf("%d imported as hidden, want visible", id)
		}
	}
}

// A Watcharr export restore is lossless: grade and hidden come back.
func TestWatcharrImportRestoresGradeAndHidden(t *testing.T) {
	s := testserver.New(t)
	token := s.SeedAdmin()
	s.Do(http.MethodPost, "/api/import", map[string]any{
		"tmdbId": 550, "type": "movie", "rating": 7, "status": "FINISHED", "grade": "S", "hidden": true,
	}, token)
	w := allWatched(t, s, token)[550]
	if gradeStr(w.Grade) != "S" || !w.Hidden || w.Rating != 7 {
		t.Fatalf("restored: %+v", w)
	}
	if rec := s.Do(http.MethodPost, "/api/import", map[string]any{
		"tmdbId": 603, "type": "movie", "status": "FINISHED", "grade": "a",
	}, token); rec.Code != http.StatusBadRequest {
		t.Fatalf("import with invalid grade: expected 400, got %d", rec.Code)
	}
}

func TestAdminGradeSortAndFilter(t *testing.T) {
	s := testserver.New(t)
	token := s.SeedAdmin()
	for _, it := range []map[string]any{
		{"contentType": "movie", "tmdbId": 550, "status": "FINISHED", "grade": "B"},
		{"contentType": "movie", "tmdbId": 603, "status": "FINISHED", "grade": "S"},
		{"contentType": "movie", "tmdbId": 680, "status": "WATCHING"},
		{"contentType": "movie", "tmdbId": 13, "status": "PLANNED", "grade": "S"},
		{"contentType": "tv", "tmdbId": 1396, "status": "FINISHED", "grade": "F"},
	} {
		s.AddWatched(token, it)
	}
	titles := func(path string) []string {
		rec := s.Do(http.MethodGet, path, nil, token)
		var r struct {
			Results []struct {
				Name string `json:"name"`
			} `json:"results"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
			t.Fatalf("%s: %v %s", path, err, rec.Body.String())
		}
		out := []string{}
		for _, x := range r.Results {
			out = append(out, x.Name)
		}
		return out
	}
	want := []string{"The Matrix", "Fight Club", "Breaking Bad", "Pulp Fiction", "Forrest Gump"}
	if got := titles("/api/watched?page=1&sort=GRADE&sortDir=desc"); !slices.Equal(got, want) {
		t.Errorf("admin grade sort:\n got  %v\n want %v", got, want)
	}
	if got := titles("/api/watched?page=1&grade=none"); !slices.Equal(got, []string{"Pulp Fiction"}) {
		t.Errorf("admin grade=none: %v", got)
	}
	// A letter never matches planned items (Forrest Gump has S saved).
	if got := titles("/api/watched?page=1&grade=S"); !slices.Equal(got, []string{"The Matrix"}) {
		t.Errorf("admin grade=S: %v", got)
	}
}

// The grade column is nullable text with no backfill: rows written without a
// grade stay null.
func TestGradeColumnIsNullable(t *testing.T) {
	s := testserver.New(t)
	if !s.DB.Migrator().HasColumn(&entity.Watched{}, "grade") {
		t.Fatal("watcheds.grade column missing")
	}
	token := s.SeedAdmin()
	id := s.AddWatched(token, map[string]any{"contentType": "movie", "tmdbId": 550, "status": "FINISHED", "rating": 9})
	var n int64
	s.DB.Model(&entity.Watched{}).Where("id = ? AND grade IS NULL", id).Count(&n)
	if n != 1 {
		t.Fatal("expected grade to be NULL in the database")
	}
}
