package app_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sbondCo/Watcharr/app"
	"github.com/sbondCo/Watcharr/internal/testutil"
	"github.com/sbondCo/Watcharr/internal/testutil/testserver"
)

func TestMain(m *testing.M) {
	testutil.SetupLogging()
	m.Run()
}

func TestAnonymousRequestIsRejected(t *testing.T) {
	s := testserver.New(t)
	rec := s.Do(http.MethodGet, "/api/watched", nil, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestNonAdminIsRejectedFromAdminRoute(t *testing.T) {
	s := testserver.New(t)
	s.SeedAdmin()
	rec := s.Do(http.MethodGet, "/api/server/config", nil, s.UserToken("someone"))
	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusForbidden {
		t.Fatalf("expected 401 or 403, got %d", rec.Code)
	}
}

func TestUnknownAPIRouteIsJSON404AndNotProxied(t *testing.T) {
	// Proxy host points nowhere, if the request were proxied we'd get a 502.
	s := testserver.NewWithOptions(t, app.Options{UIProxyHost: "127.0.0.1:1"})
	for _, p := range []string{"/api/does-not-exist", "/api", "/api/auth/nope"} {
		rec := s.Do(http.MethodGet, p, nil, "")
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s: expected 404, got %d", p, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Fatalf("%s: expected json content type, got %q", p, ct)
		}
	}
	// Non api routes are proxied. The reverse proxy needs a real
	// ResponseWriter (the recorder isn't a CloseNotifier), so use a server.
	srv := httptest.NewServer(s.Engine)
	defer srv.Close()
	res, err := http.Get(srv.URL + "/some/page")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected non api route to be proxied (502), got %d", res.StatusCode)
	}
}

func TestAdminCanAddMovieFromTMDBStub(t *testing.T) {
	s := testserver.New(t)
	token := s.SeedAdmin()

	rec := s.Do(http.MethodPost, "/api/watched", map[string]any{
		"contentType": "movie",
		"tmdbId":      550,
		"status":      "FINISHED",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("add watched: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = s.Do(http.MethodGet, "/api/watched", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("get watched: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	list := testserver.DecodeJSON[[]struct {
		Status  string `json:"status"`
		Content struct {
			Title  string `json:"title"`
			TmdbID int    `json:"tmdbId"`
		} `json:"content"`
	}](t, rec)
	if len(list) != 1 || list[0].Content.Title != "Fight Club" || list[0].Status != "FINISHED" {
		t.Fatalf("unexpected watched list: %+v", list)
	}
}
