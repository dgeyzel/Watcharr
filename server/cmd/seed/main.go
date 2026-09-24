// seed creates a fresh Watcharr data dir (config + database) filled with a
// known set of fixture data for the e2e tests. Everything is created through
// the real API routes (in-process), with TMDB served from the fixture stub.
//
//	WATCHARR_DATA=/tmp/e2e-data go run ./cmd/seed
//
// It refuses to run against a data dir that already has a database, so it
// can never touch real data.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path"

	"github.com/gin-gonic/gin"
	"github.com/sbondCo/Watcharr/app"
	"github.com/sbondCo/Watcharr/config"
	"github.com/sbondCo/Watcharr/database"
	"github.com/sbondCo/Watcharr/feature/setup/setupglob"
	"github.com/sbondCo/Watcharr/internal/testutil/tmdbstub"
)

// Credentials of the seeded admin, the e2e tests log in with these.
const (
	AdminUsername = "admin"
	AdminPassword = "e2e-admin-password"
)

type seedItem struct {
	ContentType string
	TmdbID      int
	Status      string
	Rating      float64
	Thoughts    string
	Tagged      bool
	// Hidden from visitors (a draft).
	Hidden bool
	// S-F grade ("" = ungraded).
	Grade string
}

// The fixture list. It covers movies and tv, every status and a tag.
// Later phases extend it (hidden items, grades).
var items = []seedItem{
	{"movie", 550, "FINISHED", 9, "A seeded review of Fight Club.", true, false, "A"},
	{"movie", 603, "FINISHED", 8, "A hidden draft review of The Matrix.", false, true, "B"},
	{"movie", 680, "WATCHING", 0, "", false, false, ""},
	{"movie", 13, "PLANNED", 0, "", false, false, ""},
	{"movie", 27205, "HOLD", 7, "An on hold review.", true, false, "D"},
	{"movie", 157336, "DROPPED", 3, "A dropped review.", false, false, ""},
	{"tv", 1396, "FINISHED", 10, "A seeded review of Breaking Bad.", true, false, "S"},
	{"tv", 1399, "WATCHING", 0, "", false, false, "C"},
	{"tv", 66732, "PLANNED", 0, "", false, false, ""},
}

func main() {
	log.SetFlags(0)
	if os.Getenv("WATCHARR_DATA") == "" {
		log.Fatal("seed: WATCHARR_DATA must be set to a (new) directory")
	}
	if _, err := os.Stat(path.Join(config.DataPath, "watcharr.db")); err == nil {
		log.Fatalf("seed: %s already contains a database, refusing to seed", config.DataPath)
	}
	if err := os.MkdirAll(config.DataPath, 0764); err != nil {
		log.Fatalf("seed: create data dir: %v", err)
	}

	// Config written to disk, used by the real server afterwards. TMDB urls
	// are left out, the server is given them via env.
	cfg := &config.ServerConfig{
		JWT_SECRET:      "e2e-jwt-secret-not-for-production-use-000000000000",
		DEFAULT_COUNTRY: "US",
		TMDB_KEY:        "e2e-tmdb-key",
	}
	if err := cfg.Write(); err != nil {
		log.Fatalf("seed: write config: %v", err)
	}

	// In-process TMDB stub for seeding only.
	dir := os.Getenv("TMDB_FIXTURES_DIR")
	if dir == "" {
		dir = tmdbstub.FixturesDir()
	}
	stub := httptest.NewServer(tmdbstub.Handler(dir))
	defer stub.Close()
	seedCfg := *cfg
	seedCfg.TMDB_API_BASE = stub.URL
	seedCfg.TMDB_IMAGE_BASE = stub.URL + "/t/p"
	os.Unsetenv("TMDB_API_BASE")
	os.Unsetenv("TMDB_IMAGE_BASE")

	db, err := database.New()
	if err != nil {
		log.Fatalf("seed: open db: %v", err)
	}
	gin.SetMode(gin.TestMode)
	gin.DefaultWriter = io.Discard
	engine := app.NewEngine(db, &seedCfg, app.Options{})
	c := client{engine: engine}

	var auth struct {
		Token string `json:"token"`
	}
	c.do(http.MethodPost, "/api/setup/create_admin", map[string]string{
		"username":   AdminUsername,
		"password":   AdminPassword,
		"setupToken": setupglob.SetupToken,
	}, &auth)
	c.token = auth.Token

	var tag struct {
		ID int `json:"id"`
	}
	c.do(http.MethodPost, "/api/tag", map[string]string{
		"name": "Favourites", "color": "#ffffff", "bgColor": "#aa3355",
	}, &tag)

	for _, it := range items {
		var w struct {
			ID int `json:"id"`
		}
		body := map[string]any{
			"contentType": it.ContentType,
			"tmdbId":      it.TmdbID,
			"status":      it.Status,
			"rating":      it.Rating,
			"thoughts":    it.Thoughts,
		}
		if it.Grade != "" {
			body["grade"] = it.Grade
		}
		c.do(http.MethodPost, "/api/watched", body, &w)
		if it.Hidden {
			c.do(http.MethodPut, fmt.Sprintf("/api/watched/%d", w.ID), map[string]any{"hidden": true}, nil)
		}
		if it.Tagged {
			c.do(http.MethodPost, fmt.Sprintf("/api/watched/%d/tag/%d", w.ID, tag.ID), nil, nil)
		}
	}
	log.Printf("seed: seeded %d items into %s (admin: %s)", len(items), config.DataPath, AdminUsername)
}

type client struct {
	engine *gin.Engine
	token  string
}

// do runs a request in-process, exiting on any non 2xx response.
func (c *client) do(method string, p string, body any, out any) {
	var rb io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			log.Fatalf("seed: marshal %s: %v", p, err)
		}
		rb = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, p, rb)
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", c.token)
	}
	rec := httptest.NewRecorder()
	c.engine.ServeHTTP(rec, req)
	if rec.Code < 200 || rec.Code > 299 {
		log.Fatalf("seed: %s %s returned %d: %s", method, p, rec.Code, rec.Body.String())
	}
	if out != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			log.Fatalf("seed: decode %s: %v", p, err)
		}
	}
}
