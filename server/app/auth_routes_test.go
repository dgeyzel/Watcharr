package app_test

import (
	"net/http"
	"testing"

	"github.com/sbondCo/Watcharr/feature/setup/setupglob"
	"github.com/sbondCo/Watcharr/internal/testutil/testserver"
)

// Routes for account creation, alternate logins, admin token promotion,
// Plex, Jellyfin and Sonarr/Radarr were removed.
func TestRemovedRoutesReturn404(t *testing.T) {
	s := testserver.New(t)
	token := s.SeedAdmin()
	removed := []struct{ method, path string }{
		{http.MethodPost, "/api/auth/register"},
		{http.MethodPost, "/api/auth/jellyfin"},
		{http.MethodPost, "/api/auth/plex"},
		{http.MethodPost, "/api/auth/proxy"},
		{http.MethodGet, "/api/auth/proxy_logout_details"},
		{http.MethodGet, "/api/auth/admin_token"},
		{http.MethodPost, "/api/auth/admin_token"},
		{http.MethodGet, "/api/plex/sync"},
		{http.MethodGet, "/api/jellyfin/sync"},
		{http.MethodGet, "/api/jellyfin/movie/x/1"},
		{http.MethodPost, "/api/server/config/plex_host"},
		{http.MethodGet, "/api/arr/son"},
		{http.MethodPost, "/api/arr/rad/add"},
		{http.MethodGet, "/api/arr/request"},
		{http.MethodPost, "/api/arr/request/approve/1"},
		// Social features.
		{http.MethodGet, "/api/follow"},
		{http.MethodPost, "/api/follow/2"},
		{http.MethodDelete, "/api/follow/2"},
		{http.MethodGet, "/api/follow/thoughts/movie/550"},
		{http.MethodGet, "/api/user/search"},
		{http.MethodGet, "/api/user/public/1/admin"},
		{http.MethodGet, "/api/watched/1/admin"},
		{http.MethodGet, "/api/server/users"},
		{http.MethodPost, "/api/server/users/1"},
		// Games.
		{http.MethodGet, "/api/game/1"},
		{http.MethodPost, "/api/game/config"},
		{http.MethodGet, "/api/features"},
	}
	for _, r := range removed {
		// With and without a (admin) token, so no auth middleware masks it.
		for _, tok := range []string{"", token} {
			if rec := s.Do(r.method, r.path, nil, tok); rec.Code != http.StatusNotFound {
				t.Errorf("%s %s (token=%v): expected 404, got %d", r.method, r.path, tok != "", rec.Code)
			}
		}
	}
}

func TestAvailableOnlyReportsSetupState(t *testing.T) {
	s := testserver.New(t)
	rec := s.Do(http.MethodGet, "/api/auth/available", nil, "")
	if rec.Code != http.StatusOK || rec.Body.String() != `{"isInSetup":true}` {
		t.Fatalf("unexpected /auth/available response %d: %s", rec.Code, rec.Body.String())
	}
	s.SeedAdmin()
	rec = s.Do(http.MethodGet, "/api/auth/available", nil, "")
	if rec.Body.String() != `{"isInSetup":false}` {
		t.Fatalf("expected setup to be finished: %s", rec.Body.String())
	}
}

func TestCreateAdminRequiresSetupToken(t *testing.T) {
	s := testserver.New(t)
	body := func(tok string) map[string]string {
		return map[string]string{"username": "owner", "password": "a-long-enough-password", "setupToken": tok}
	}
	if setupglob.SetupToken == "" {
		t.Fatal("expected a setup token to be generated while in setup")
	}
	if rec := s.Do(http.MethodPost, "/api/setup/create_admin", map[string]string{
		"username": "owner", "password": "a-long-enough-password",
	}, ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("missing token: expected 400, got %d", rec.Code)
	}
	if rec := s.Do(http.MethodPost, "/api/setup/create_admin", body("wrong-token"), ""); rec.Code != http.StatusForbidden {
		t.Fatalf("wrong token: expected 403, got %d", rec.Code)
	}
	short := body(setupglob.SetupToken)
	short["password"] = "short"
	if rec := s.Do(http.MethodPost, "/api/setup/create_admin", short, ""); rec.Code != http.StatusForbidden {
		t.Fatalf("short password: expected 403, got %d", rec.Code)
	}
	tok := setupglob.SetupToken
	if rec := s.Do(http.MethodPost, "/api/setup/create_admin", body(tok), ""); rec.Code != http.StatusOK {
		t.Fatalf("valid token: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	// Setup is over, the old token can't be reused.
	if rec := s.Do(http.MethodPost, "/api/setup/create_admin", body(tok), ""); rec.Code != http.StatusForbidden {
		t.Fatalf("after setup: expected 403, got %d", rec.Code)
	}
}

func TestLoginIsRateLimited(t *testing.T) {
	s := testserver.New(t)
	s.SeedAdmin()
	bad := map[string]string{"username": testserver.AdminUsername, "password": "wrong-password-123"}
	for i := 1; i <= 5; i++ {
		if rec := s.Do(http.MethodPost, "/api/auth/", bad, ""); rec.Code != http.StatusForbidden {
			t.Fatalf("attempt %d: expected 403, got %d", i, rec.Code)
		}
	}
	// Even the right password is refused once over the limit.
	good := map[string]string{"username": testserver.AdminUsername, "password": testserver.AdminPassword}
	if rec := s.Do(http.MethodPost, "/api/auth/", good, ""); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("attempt 6: expected 429, got %d", rec.Code)
	}
}

func TestAdminCanLogin(t *testing.T) {
	s := testserver.New(t)
	s.SeedAdmin()
	rec := s.Do(http.MethodPost, "/api/auth/", map[string]string{
		"username": testserver.AdminUsername, "password": testserver.AdminPassword,
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
