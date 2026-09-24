package public_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/internal/testutil"
	"github.com/sbondCo/Watcharr/internal/testutil/testserver"
)

func TestMain(m *testing.M) {
	testutil.SetupLogging()
	m.Run()
}

type fixture struct {
	t     *testing.T
	s     *testserver.Server
	token string
	ids   map[string]uint // title -> watched id
	tagID uint
	// Tag that only has hidden/admin only items.
	privateTagID uint
}

type item struct {
	title, contentType string
	tmdbID             int
	status             string
	rating             float64
	thoughts           string
	tagged             bool
	// Grade set when adding ("" = ungraded).
	grade string
}

var items = []item{
	{"Fight Club", "movie", 550, "FINISHED", 9, "Review of Fight Club.", true, "A"},
	{"The Matrix", "movie", 603, "FINISHED", 8, "Hidden draft review.", false, "F"}, // hidden below
	{"Pulp Fiction", "movie", 680, "WATCHING", 0, "", false, ""},
	{"Forrest Gump", "movie", 13, "PLANNED", 0, "", false, "B"}, // planned, grade must not show
	{"Inception", "movie", 27205, "HOLD", 7, "On hold.", true, "D"},
	{"Interstellar", "movie", 157336, "DROPPED", 3, "Dropped.", false, ""},
	{"Breaking Bad", "tv", 1396, "FINISHED", 10, "Review of Breaking Bad.", true, "S"},
	{"Game of Thrones", "tv", 1399, "WATCHING", 0, "", false, "C"},
	{"Stranger Things", "tv", 66732, "PLANNED", 0, "", false, ""},
}

// The titles visitors should see from `items`.
var visibleTitles = []string{
	"Breaking Bad", "Fight Club", "Forrest Gump", "Game of Thrones", "Pulp Fiction", "Stranger Things",
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	s := testserver.New(t)
	f := &fixture{t: t, s: s, token: s.SeedAdmin(), ids: map[string]uint{}}
	tag := func(name string) uint {
		rec := s.Do(http.MethodPost, "/api/tag", map[string]string{"name": name}, f.token)
		if rec.Code != http.StatusOK {
			t.Fatalf("create tag: %d %s", rec.Code, rec.Body.String())
		}
		return testserver.DecodeJSON[struct {
			ID uint `json:"id"`
		}](t, rec).ID
	}
	f.tagID = tag("Favourites")
	f.privateTagID = tag("Drafts")
	for _, it := range items {
		body := map[string]any{
			"contentType": it.contentType, "tmdbId": it.tmdbID,
			"status": it.status, "rating": it.rating, "thoughts": it.thoughts,
		}
		if it.grade != "" {
			body["grade"] = it.grade
		}
		id := s.AddWatched(f.token, body)
		f.ids[it.title] = id
		if it.tagged {
			f.do(http.MethodPost, fmt.Sprintf("/api/watched/%d/tag/%d", id, f.tagID), nil)
		}
	}
	// Hide The Matrix (a FINISHED item), tag it and a HOLD item as drafts.
	f.do(http.MethodPut, fmt.Sprintf("/api/watched/%d", f.ids["The Matrix"]), map[string]any{"hidden": true})
	f.do(http.MethodPost, fmt.Sprintf("/api/watched/%d/tag/%d", f.ids["The Matrix"], f.privateTagID), nil)
	f.do(http.MethodPost, fmt.Sprintf("/api/watched/%d/tag/%d", f.ids["Inception"], f.privateTagID), nil)

	// Another (non admin) user's entry for the same title must never show.
	var other entity.User
	s.UserToken("someone-else")
	s.DB.Where("username = ?", "someone-else").Take(&other)
	var content entity.Content
	s.DB.Where("tmdb_id = ?", 550).Take(&content)
	s.DB.Create(&entity.Watched{UserID: other.ID, ContentID: &content.ID, Status: entity.FINISHED, Thoughts: "not the owner"})
	return f
}

func (f *fixture) do(method, path string, body any) {
	f.t.Helper()
	if rec := f.s.Do(method, path, body, f.token); rec.Code != http.StatusOK && rec.Code != http.StatusNoContent {
		f.t.Fatalf("%s %s: %d %s", method, path, rec.Code, rec.Body.String())
	}
}

// get performs an anonymous request.
func (f *fixture) get(path string) (int, []byte) {
	rec := f.s.Do(http.MethodGet, path, nil, "")
	return rec.Code, rec.Body.Bytes()
}

type listResp struct {
	Page         int   `json:"page"`
	Limit        int   `json:"limit"`
	TotalPages   int   `json:"totalPages"`
	TotalResults int64 `json:"totalResults"`
	Results      []struct {
		Title  string `json:"title"`
		Status string `json:"status"`
		Review string `json:"review"`
	} `json:"results"`
}

func (f *fixture) list(t *testing.T, path string) listResp {
	t.Helper()
	code, body := f.get(path)
	if code != http.StatusOK {
		t.Fatalf("GET %s: %d %s", path, code, body)
	}
	var r listResp
	if err := json.Unmarshal(body, &r); err != nil {
		t.Fatal(err)
	}
	return r
}

func titles(r listResp) []string {
	out := []string{}
	for _, v := range r.Results {
		out = append(out, v.Title)
	}
	return out
}

func sorted(s []string) []string {
	c := slices.Clone(s)
	sort.Strings(c)
	return c
}

func TestListOnlyReturnsVisibleItems(t *testing.T) {
	f := newFixture(t)
	r := f.list(t, "/api/public/watched")
	if got := sorted(titles(r)); !slices.Equal(got, visibleTitles) {
		t.Fatalf("visible items:\n got  %v\n want %v", got, visibleTitles)
	}
	if r.TotalResults != int64(len(visibleTitles)) {
		t.Fatalf("totalResults %d, want %d", r.TotalResults, len(visibleTitles))
	}
	for _, v := range r.Results {
		if v.Review == "not the owner" {
			t.Fatal("another user's entry leaked into the public list")
		}
	}

	// Deleting an item removes it (soft delete).
	f.do(http.MethodDelete, fmt.Sprintf("/api/watched/%d", f.ids["Pulp Fiction"]), nil)
	if slices.Contains(titles(f.list(t, "/api/public/watched")), "Pulp Fiction") {
		t.Fatal("deleted item still listed")
	}

	// Unhiding makes it visible.
	f.do(http.MethodPut, fmt.Sprintf("/api/watched/%d", f.ids["The Matrix"]), map[string]any{"hidden": false})
	if !slices.Contains(titles(f.list(t, "/api/public/watched")), "The Matrix") {
		t.Fatal("unhidden item not listed")
	}
}

func TestSingleItemEndpoints(t *testing.T) {
	f := newFixture(t)
	ok := []string{"/movie/550", "/movie/680", "/movie/13", "/tv/1396", "/tv/66732"}
	notFound := []string{
		"/movie/603",    // hidden
		"/movie/27205",  // HOLD
		"/movie/157336", // DROPPED
		"/movie/999999", // not in the list
		"/tv/550",       // wrong type
		"/game/550",     // games aren't public
		"/movie/abc",
	}
	for _, base := range []string{"/api/public/watched", "/api/public/content"} {
		for _, p := range ok {
			if code, body := f.get(base + p); code != http.StatusOK {
				t.Errorf("GET %s%s: expected 200, got %d %s", base, p, code, body)
			}
		}
		for _, p := range notFound {
			if code, _ := f.get(base + p); code != http.StatusNotFound {
				t.Errorf("GET %s%s: expected 404, got %d", base, p, code)
			}
		}
	}
	_, body := f.get("/api/public/watched/movie/550")
	if !strings.Contains(string(body), `"review":"Review of Fight Club."`) {
		t.Fatalf("review missing: %s", body)
	}
	_, body = f.get("/api/public/content/movie/550")
	if !strings.Contains(string(body), `"title":"Fight Club"`) {
		t.Fatalf("content details missing: %s", body)
	}
}

func TestListFiltersSortAndPagination(t *testing.T) {
	f := newFixture(t)
	cases := []struct {
		query string
		want  []string
	}{
		{"status=planned", []string{"Forrest Gump", "Stranger Things"}},
		{"status=finished,watching", []string{"Breaking Bad", "Fight Club", "Game of Thrones", "Pulp Fiction"}},
		// Admin only statuses are ignored, not revealed.
		{"status=hold,dropped", visibleTitles},
		{"type=tv", []string{"Breaking Bad", "Game of Thrones", "Stranger Things"}},
		{fmt.Sprintf("tag=%d", f.tagID), []string{"Breaking Bad", "Fight Club"}},
		{"q=fight", []string{"Fight Club"}},
		{"q=matrix", []string{}},
	}
	for _, c := range cases {
		if got := sorted(titles(f.list(t, "/api/public/watched?"+c.query))); !slices.Equal(got, sorted(c.want)) {
			t.Errorf("?%s:\n got  %v\n want %v", c.query, got, c.want)
		}
	}

	if got := titles(f.list(t, "/api/public/watched?sort=ALPHA&sortDir=asc")); !slices.Equal(got, visibleTitles) {
		t.Errorf("alpha asc order: %v", got)
	}

	r := f.list(t, "/api/public/watched?limit=2&page=3&sort=ALPHA&sortDir=asc")
	if r.TotalPages != 3 || len(r.Results) != 2 || r.Results[0].Title != "Pulp Fiction" {
		t.Errorf("pagination: %+v", r)
	}
	if r := f.list(t, "/api/public/watched?limit=100000"); r.Limit != 100 {
		t.Errorf("limit not clamped: %d", r.Limit)
	}
}

func TestTags(t *testing.T) {
	f := newFixture(t)
	r := f.list(t, fmt.Sprintf("/api/public/tag/%d/watched", f.tagID))
	if got := sorted(titles(r)); !slices.Equal(got, []string{"Breaking Bad", "Fight Club"}) {
		t.Errorf("tag items: %v", got)
	}
	// A tag with only hidden/admin only items returns an empty list.
	if r := f.list(t, fmt.Sprintf("/api/public/tag/%d/watched", f.privateTagID)); len(r.Results) != 0 || r.TotalResults != 0 {
		t.Errorf("private tag leaked items: %v", titles(r))
	}
	if code, _ := f.get("/api/public/tag/9999/watched"); code != http.StatusNotFound {
		t.Errorf("unknown tag: expected 404, got %d", code)
	}
	code, body := f.get("/api/public/tags")
	if code != http.StatusOK || !strings.Contains(string(body), `"name":"Favourites"`) {
		t.Errorf("tags: %d %s", code, body)
	}
}

func TestOwner(t *testing.T) {
	f := newFixture(t)
	code, body := f.get("/api/public/owner")
	if code != http.StatusOK || string(body) != `{"username":"admin","bio":"","avatar":null}` {
		t.Fatalf("owner: %d %s", code, body)
	}
}

func TestEmptySiteHasEmptyPublicLists(t *testing.T) {
	s := testserver.New(t)
	rec := s.Do(http.MethodGet, "/api/public/watched", nil, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"results":[]`) {
		t.Fatalf("empty site list: %d %s", rec.Code, rec.Body.String())
	}
	if rec := s.Do(http.MethodGet, "/api/public/owner", nil, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("owner before setup: expected 404, got %d", rec.Code)
	}
}

// keysOf returns the sorted keys of a json object.
func keysOf(t *testing.T, raw json.RawMessage) []string {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("not an object: %s", raw)
	}
	out := []string{}
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Public responses must have exactly these keys, so nothing private (numeric
// ratings, user ids, activity, hidden flag, settings) can slip in.
func TestPublicResponseKeysAreExact(t *testing.T) {
	f := newFixture(t)
	watchedKeys := []string{"createdAt", "grade", "mediaType", "posterPath", "releaseDate", "review", "status", "tags", "title", "tmdbId", "updatedAt"}
	tagKeys := []string{"bgColor", "color", "id", "name"}
	pageKeys := []string{"limit", "page", "results", "totalPages", "totalResults"}
	contentKeys := []string{"backdropPath", "genres", "homepage", "mediaType", "overview", "posterPath", "providers", "providersFullListLink", "releaseDate", "releaseDateLast", "runtime", "seasons", "status", "title", "tmdbId", "videos"}

	check := func(name string, raw json.RawMessage, want []string) {
		if got := keysOf(t, raw); !slices.Equal(got, want) {
			t.Errorf("%s keys:\n got  %v\n want %v", name, got, want)
		}
	}
	_, body := f.get("/api/public/owner")
	check("owner", body, []string{"avatar", "bio", "username"})

	for _, p := range []string{"/api/public/watched", fmt.Sprintf("/api/public/tag/%d/watched", f.tagID)} {
		_, body = f.get(p)
		check(p, body, pageKeys)
		var page struct{ Results []json.RawMessage }
		json.Unmarshal(body, &page)
		for _, r := range page.Results {
			check(p+" item", r, watchedKeys)
			var w struct{ Tags []json.RawMessage }
			json.Unmarshal(r, &w)
			for _, tg := range w.Tags {
				check(p+" item tag", tg, tagKeys)
			}
		}
	}
	_, body = f.get("/api/public/watched/movie/550")
	check("watched item", body, watchedKeys)
	_, body = f.get("/api/public/content/movie/550")
	check("movie content", body, contentKeys)
	_, body = f.get("/api/public/content/tv/1396")
	check("tv content", body, contentKeys)
	_, body = f.get(fmt.Sprintf("/api/public/tag/%d", f.tagID))
	check("tag", body, tagKeys)
	_, body = f.get("/api/public/tags")
	var tags []json.RawMessage
	json.Unmarshal(body, &tags)
	for _, tg := range tags {
		check("tags item", tg, tagKeys)
	}
}

// Belt and braces: no public response ever contains these keys, anywhere.
func TestPublicResponsesNeverContainPrivateKeys(t *testing.T) {
	f := newFixture(t)
	paths := []string{
		"/api/public/owner", "/api/public/watched", "/api/public/watched/movie/550", "/api/public/stats",
		"/api/public/content/movie/550", "/api/public/content/tv/1396", "/api/public/tags",
		fmt.Sprintf("/api/public/tag/%d", f.tagID), fmt.Sprintf("/api/public/tag/%d/watched", f.tagID),
	}
	forbidden := []string{`"rating"`, `"ratingCount"`, `"voteAverage"`, `"vote_average"`, `"userId"`, `"user_id"`, `"activity"`, `"hidden"`, `"password"`, `"permissions"`, `"thoughts"`, `"pinned"`, `"similar"`, `"watched"`}
	for _, p := range paths {
		_, body := f.get(p)
		for _, k := range forbidden {
			if strings.Contains(string(body), k) {
				t.Errorf("%s contains forbidden key %s: %s", p, k, body)
			}
		}
	}
}

func TestPublicWritesAreRejected(t *testing.T) {
	f := newFixture(t)
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		for _, p := range []string{"/api/public/watched", "/api/public/watched/movie/550", "/api/public/tags"} {
			if rec := f.s.Do(m, p, nil, ""); rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s %s: expected 404/405, got %d", m, p, rec.Code)
			}
		}
	}
}
