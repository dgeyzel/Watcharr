package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Config files from upstream Watcharr may still contain keys this fork
// removed. They must load without error and be ignored.
func TestOldConfigWithRemovedKeysLoads(t *testing.T) {
	old := DataPath
	DataPath = t.TempDir()
	t.Cleanup(func() { DataPath = old })

	oldCfg := `{
	"JWT_SECRET": "keep-me",
	"DEFAULT_COUNTRY": "GB",
	"JELLYFIN_HOST": "http://jellyfin:8096",
	"USE_EMBY": true,
	"SIGNUP_ENABLED": true,
	"TMDB_KEY": "abc",
	"PLEX_HOST": "http://plex:32400",
	"PLEX_MACHINE_ID": "machine",
	"HEADER_AUTH": {"enabled": true, "headerName": "Remote-User", "autoLogin": true, "logoutUrl": "https://x"},
	"SONARR": [{"name": "s", "host": "http://sonarr", "key": "k", "qualityProfile": 1, "rootFolder": 1, "languageProfile": 1, "automaticSearch": true}],
	"RADARR": [{"name": "r", "host": "http://radarr", "key": "k", "qualityProfile": 1, "rootFolder": 1, "automaticSearch": true}],
	"TWITCH": {"clientId": "id", "clientSecret": "secret"},
	"DEBUG": true
}`
	if err := os.WriteFile(filepath.Join(DataPath, "watcharr.json"), []byte(oldCfg), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Get()
	if err != nil {
		t.Fatalf("old config failed to load: %v", err)
	}
	if cfg.JWT_SECRET != "keep-me" || cfg.DEFAULT_COUNTRY != "GB" || cfg.TMDB_KEY != "abc" || !cfg.DEBUG {
		t.Fatalf("kept keys not read correctly: %+v", cfg)
	}

	// Writing drops the removed keys.
	if err := cfg.Write(); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(DataPath, "watcharr.json"))
	for _, k := range []string{"JELLYFIN_HOST", "SIGNUP_ENABLED", "PLEX_HOST", "HEADER_AUTH", "SONARR", "RADARR", "USE_EMBY", "TWITCH"} {
		if strings.Contains(string(b), k) {
			t.Fatalf("removed key %s written back to config: %s", k, b)
		}
	}
}

func TestRemovedKeysCannotBeUpdated(t *testing.T) {
	c := &ServerConfig{}
	for _, k := range []string{"JELLYFIN_HOST", "USE_EMBY", "SIGNUP_ENABLED", "PLEX_HOST"} {
		if err := c.UpdateConfig(k, "x"); err == nil {
			t.Fatalf("expected UpdateConfig(%s) to fail", k)
		}
	}
}
