package app_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/sbondCo/Watcharr/app"
	"github.com/sbondCo/Watcharr/internal/testutil/testserver"
)

var (
	paramSeg    = regexp.MustCompile(`:[^/]+`)
	wildcardSeg = regexp.MustCompile(`\*[^/]*$`)
)

// concretePath turns a gin route pattern into a requestable path.
func concretePath(p string) string {
	return wildcardSeg.ReplaceAllString(paramSeg.ReplaceAllString(p, "1"), "x")
}

// The key security test: every registered /api route that is not on the
// explicit public allowlist must refuse anonymous and non admin requests. A
// route added later without admin protection fails this automatically.
func TestEveryNonPublicRouteRequiresAdmin(t *testing.T) {
	// Engine is in setup (no users), so the setup routes are registered too.
	s := testserver.New(t)
	userToken := s.UserToken("not-an-admin")

	checked := 0
	for _, r := range s.Engine.Routes() {
		if app.IsPublicRoute(r.Method, r.Path) {
			continue
		}
		p := concretePath(r.Path)
		if rec := s.Do(r.Method, p, nil, ""); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s (anonymous): expected 401, got %d", r.Method, r.Path, rec.Code)
		}
		if rec := s.Do(r.Method, p, nil, userToken); rec.Code != http.StatusUnauthorized && rec.Code != http.StatusForbidden {
			t.Errorf("%s %s (non admin): expected 401 or 403, got %d", r.Method, r.Path, rec.Code)
		}
		checked++
	}
	if checked < 40 {
		t.Fatalf("only %d routes checked, route walking looks broken", checked)
	}
}

// Guard against the allowlist being widened by accident.
func TestPublicRouteAllowlistIsExact(t *testing.T) {
	want := map[string]bool{
		"GET /api/public/*":            true,
		"POST /api/auth/":              true,
		"GET /api/auth/available":      true,
		"POST /api/setup/create_admin": true,
		"GET /api/img/*":               true,
		"HEAD /api/img/*":              true,
	}
	if len(app.PublicRoutes) != len(want) {
		t.Fatalf("expected %d public routes, got %d: %+v", len(want), len(app.PublicRoutes), app.PublicRoutes)
	}
	for _, r := range app.PublicRoutes {
		if !want[r.Method+" "+r.Path] {
			t.Errorf("unexpected public route %s %s", r.Method, r.Path)
		}
	}
	// Non GET methods on public prefixes are never public.
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		if app.IsPublicRoute(m, "/api/public/watched") {
			t.Errorf("%s /api/public/watched must not be public", m)
		}
	}
}

func TestAdminPassesAllowlist(t *testing.T) {
	s := testserver.New(t)
	token := s.SeedAdmin()
	for _, p := range []string{"/api/watched", "/api/user", "/api/server/config", "/api/tag"} {
		if rec := s.Do(http.MethodGet, p, nil, token); rec.Code != http.StatusOK {
			t.Errorf("GET %s as admin: expected 200, got %d: %s", p, rec.Code, rec.Body.String())
		}
	}
}
