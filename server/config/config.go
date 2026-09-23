package config

import (
	"encoding/json"
	"errors"
	"log"
	"log/slog"
	"os"
	"path"
	"time"

	"github.com/sbondCo/Watcharr/logging"
	"github.com/sbondCo/Watcharr/media/igdb"
	"github.com/sbondCo/Watcharr/util"
)

var DataPath = func() string {
	path := os.Getenv("WATCHARR_DATA")
	if path == "" {
		path = "./data"
	}
	return path
}()

// ServerConfig is read from `watcharr.json` in the data dir.
//
// Keys removed in this fork (JELLYFIN_HOST, USE_EMBY, SIGNUP_ENABLED,
// PLEX_HOST, PLEX_MACHINE_ID, HEADER_AUTH, SONARR, RADARR) may still exist
// in old config files, they are ignored when read and dropped on next write.
type ServerConfig struct {
	// Used to sign JWT tokens. Make sure to make
	// it strong, just like a very long, complicated password.
	JWT_SECRET string `json:",omitempty"`

	// Default country for new users. This is used to set the default
	// region to get correct content streaming providers.
	// TODO Enforce iso_3166_1 validity (same as tmdb)
	DEFAULT_COUNTRY string `json:",omitempty"`

	// Optional: Provide your own TMDB API Key.
	// If unprovided, the default Watcharr API key will be used.
	TMDB_KEY string `json:",omitempty"`

	// Optional: Override the TMDB API base url (default
	// https://api.themoviedb.org/3). Used to point tests at a stub server.
	// The TMDB_API_BASE env var takes precedence over this.
	TMDB_API_BASE string `json:",omitempty"`

	// Optional: Override the TMDB image base url (default
	// https://image.tmdb.org/t/p). The TMDB_IMAGE_BASE env var takes
	// precedence over this.
	TMDB_IMAGE_BASE string `json:",omitempty"`

	// Optional: IPs/CIDRs of reverse proxies in front of Watcharr. Only
	// these are trusted to set X-Forwarded-For, which is used as the client
	// ip for rate limiting. When empty, the direct connection ip is used.
	TRUSTED_PROXIES []string `json:",omitempty"`

	TWITCH igdb.IGDB `json:",omitzero"`

	// Optional: Schedule for tasks.
	TASK_SCHEDULE map[string]int `json:",omitempty"`

	// Enable/disable debug logging. Useful for when trying
	// to figure out exactly what the server is doing at a point
	// of failure.
	// Set to `true` to enable.
	DEBUG bool `json:",omitempty"`
}

// ServerConfig, but with JWT_SECRET removed from json.
// Used for returning to user from get config api request.
//
// Technically only admins will have access to that api route,
// but I feel more comfortable removing it anyways (+ this is
// not editable on frontend, so not needed).
func (c *ServerConfig) GetSafe() ServerConfig {
	return ServerConfig{
		DEFAULT_COUNTRY: c.DEFAULT_COUNTRY,
		TMDB_KEY:        c.TMDB_KEY,
		DEBUG:           c.DEBUG,
		TWITCH: igdb.IGDB{
			ClientID:     c.TWITCH.ClientID,
			ClientSecret: c.TWITCH.ClientSecret,
		}, // Dont act safe, this contains twitch secrets, needed for config
	}
}

// TMDBAPIBase returns the configured TMDB API base url, preferring the
// TMDB_API_BASE env var. Empty means use the TMDB default.
func (c *ServerConfig) TMDBAPIBase() string {
	if v := os.Getenv("TMDB_API_BASE"); v != "" {
		return v
	}
	return c.TMDB_API_BASE
}

// TMDBImageBase returns the configured TMDB image base url, preferring the
// TMDB_IMAGE_BASE env var. Empty means use the TMDB default.
func (c *ServerConfig) TMDBImageBase() string {
	if v := os.Getenv("TMDB_IMAGE_BASE"); v != "" {
		return v
	}
	return c.TMDB_IMAGE_BASE
}

type ServerConfigGetByName struct {
	Value any `json:"value"`
}

// Get config item by name.
func (c *ServerConfig) Get(s string) (ServerConfigGetByName, error) {
	switch s {
	case "DEFAULT_COUNTRY":
		return ServerConfigGetByName{Value: c.DEFAULT_COUNTRY}, nil
	case "TMDB_KEY":
		return ServerConfigGetByName{Value: c.TMDB_KEY}, nil
	case "DEBUG":
		return ServerConfigGetByName{Value: c.DEBUG}, nil
	}
	return ServerConfigGetByName{}, errors.New("invalid setting")
}

// Update server config property
func (c *ServerConfig) UpdateConfig(k string, v any) error {
	slog.Debug("updateConfig", "k", k, "v", v)
	if v == nil {
		return errors.New("invalid value")
	}
	switch k {
	case "TMDB_KEY":
		s, ok := v.(string)
		if !ok {
			return errors.New("invalid value")
		}
		c.TMDB_KEY = s
	case "DEBUG":
		b, ok := v.(bool)
		if !ok {
			return errors.New("invalid value")
		}
		c.DEBUG = b
		logging.SetLevel(c.DEBUG)
	case "DEFAULT_COUNTRY":
		s, ok := v.(string)
		if !ok {
			return errors.New("invalid value")
		}
		c.DEFAULT_COUNTRY = s
	default:
		return errors.New("invalid setting")
	}
	err := c.Write()
	if err != nil {
		slog.Error("updateConfig: Failed to write updated config!", "error", err)
		return errors.New("failed to write config")
	}
	return nil
}

// Write current Config to file
func (c *ServerConfig) Write() error {
	barej, err := json.MarshalIndent(*c, "", "\t")
	if err != nil {
		return err
	}
	return os.WriteFile(path.Join(DataPath, "watcharr.json"), barej, 0755)
}

func (c *ServerConfig) SaveTwitchConfig(newt igdb.IGDB) error {
	// If existing client id and secret are same.. just return here
	if (c.TWITCH.ClientID != nil && newt.ClientID != nil && c.TWITCH.ClientSecret != nil && newt.ClientSecret != nil) &&
		*c.TWITCH.ClientID == *newt.ClientID && *c.TWITCH.ClientSecret == *newt.ClientSecret {
		slog.Info("SaveTwitchConfig: New ClientID and ClientSecret match old ClientID and ClientSecret.. ignoring request to update.")
		return nil
	}
	// Update our config
	c.TWITCH.ClientID = newt.ClientID
	c.TWITCH.ClientSecret = newt.ClientSecret
	c.TWITCH.AccessToken = ""
	c.TWITCH.AccessTokenExpires = time.Time{}
	// Try to init again
	err := c.TWITCH.Init()
	if err != nil {
		slog.Error("SaveTwitchConfig failed to initialize TWITCH", "error", err)
		return errors.New("initialization with credentials failed")
	}
	err = c.Write()
	if err != nil {
		slog.Error("SaveTwitchConfig failed to write config", "error", err)
		return errors.New("failed to save config")
	}
	return nil
}

func (c *ServerConfig) TwitchEnabled() bool {
	if c.TWITCH.ClientID != nil && c.TWITCH.ClientSecret != nil {
		return true
	}
	return false
}

// Read config file
// Calls generateConfig if file doesn't exist
func read() (*ServerConfig, error) {
	cfgFile, err := os.Open(path.Join(DataPath, "watcharr.json"))
	if err != nil {
		if os.IsNotExist(err) {
			slog.Info("Config file doesn't exist... generating.")
			if genCfg, err := generateConfig(); err == nil {
				return genCfg, nil
			}
		}
		return nil, err
	}
	defer cfgFile.Close()

	c := new(ServerConfig)
	// Unknown (e.g. removed) keys are ignored.
	dec := json.NewDecoder(cfgFile)
	if err = dec.Decode(c); err != nil {
		return nil, err
	}

	initFromConfig(c)

	return c, nil
}

// Ensure required config is provided
func initFromConfig(c *ServerConfig) {
	if c.JWT_SECRET == "" {
		log.Fatal("JWT_SECRET missing from config!")
	}
}

// Generate new barebones watcharr.json config file.
// Generates a JWT_SECRET and set default config.
func generateConfig() (*ServerConfig, error) {
	key, err := util.GenerateString(64)
	if err != nil {
		return nil, err
	}
	cfg := ServerConfig{
		JWT_SECRET: key,
		// Other defaults..
		DEFAULT_COUNTRY: "US",
	}
	barej, err := json.MarshalIndent(cfg, "", "\t")
	if err != nil {
		return nil, err
	}
	return &cfg, os.WriteFile(path.Join(DataPath, "watcharr.json"), barej, 0755)
}

// Get server config.
// Reads from config file.
func Get() (*ServerConfig, error) {
	cfg, err := read()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
