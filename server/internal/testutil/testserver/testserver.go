// Package testserver builds a fully wired Watcharr engine (the same one
// production uses, via app.NewEngine) on a fresh test database, with TMDB
// pointed at a local stub.
//
// It lives apart from testutil so packages that app depends on can keep
// using testutil in their tests without an import cycle.
package testserver

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sbondCo/Watcharr/app"
	"github.com/sbondCo/Watcharr/config"
	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/internal/testutil"
	"github.com/sbondCo/Watcharr/internal/testutil/tmdbstub"
	"gorm.io/gorm"
)

const (
	AdminUsername = "admin"
	AdminPassword = "correct-horse-battery-staple"
	testJWTSecret = "test-jwt-secret-do-not-use-in-prod"
)

type Server struct {
	t      *testing.T
	Engine *gin.Engine
	DB     *gorm.DB
	Cfg    *config.ServerConfig
	TMDB   *httptest.Server
}

// New creates a test server on a fresh database. config.DataPath is pointed
// at a temp dir for the duration of the test.
func New(t *testing.T) *Server {
	t.Helper()
	return NewWithOptions(t, app.Options{})
}

func NewWithOptions(t *testing.T, opts app.Options) *Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	gin.DefaultWriter = io.Discard

	oldDataPath := config.DataPath
	config.DataPath = t.TempDir()
	t.Cleanup(func() { config.DataPath = oldDataPath })

	stub := tmdbstub.NewServer(t)
	cfg := &config.ServerConfig{
		JWT_SECRET:      testJWTSecret,
		DEFAULT_COUNTRY: "US",
		SIGNUP_ENABLED:  true,
		TMDB_KEY:        "test-tmdb-key",
		TMDB_API_BASE:   stub.URL,
		TMDB_IMAGE_BASE: stub.URL + "/t/p",
	}
	db := testutil.SetupDB(t)
	return &Server{
		t:      t,
		Engine: app.NewEngine(db, cfg, opts),
		DB:     db,
		Cfg:    cfg,
		TMDB:   stub,
	}
}

// Do performs a request against the engine. body (if not nil) is sent as
// json and token (if not empty) as the Authorization header.
func (s *Server) Do(method string, path string, body any, token string) *httptest.ResponseRecorder {
	s.t.Helper()
	var rb io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			s.t.Fatalf("Do: failed to marshal body: %v", err)
		}
		rb = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rb)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	rec := httptest.NewRecorder()
	s.Engine.ServeHTTP(rec, req)
	return rec
}

// SeedAdmin creates the first (admin) user through the real setup route and
// returns their auth token.
func (s *Server) SeedAdmin() string {
	s.t.Helper()
	rec := s.Do(http.MethodPost, "/api/setup/create_admin", map[string]string{
		"username": AdminUsername,
		"password": AdminPassword,
	}, "")
	if rec.Code != http.StatusOK {
		s.t.Fatalf("SeedAdmin: create_admin returned %d: %s", rec.Code, rec.Body.String())
	}
	return DecodeJSON[struct {
		Token string `json:"token"`
	}](s.t, rec).Token
}

// UserToken inserts a user without admin permissions and returns a signed
// token for them. Used for negative (non-admin) tests.
func (s *Server) UserToken(username string) string {
	s.t.Helper()
	user := entity.User{Username: username, Password: "not-a-real-hash", Permissions: entity.PERM_NONE}
	if res := s.DB.Create(&user); res.Error != nil {
		s.t.Fatalf("UserToken: failed to create user: %v", res.Error)
	}
	return s.SignToken(user)
}

// SignToken signs a token for user the same way the auth service does.
func (s *Server) SignToken(user entity.User) string {
	s.t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, entity.TokenClaims{
		UserID:   user.ID,
		Username: user.Username,
		Type:     user.Type,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
			Issuer:   "watcharr",
		},
	}).SignedString([]byte(s.Cfg.JWT_SECRET))
	if err != nil {
		s.t.Fatalf("SignToken: %v", err)
	}
	return token
}

// DecodeJSON decodes the recorder's body into T, failing the test on error.
func DecodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("DecodeJSON: %v (body: %s)", err, rec.Body.String())
	}
	return v
}
