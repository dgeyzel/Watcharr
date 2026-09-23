package tmdb

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	gocache "github.com/robfig/go-cache"
	"github.com/sbondCo/Watcharr/database/entity"
)

var ContentStore = gocache.New(time.Hour*24, time.Minute)

// DefaultAPIBase is the TMDB API base url used when none is configured.
const DefaultAPIBase = "https://api.themoviedb.org/3"

// DefaultImageBase is the TMDB image base url used when none is configured.
const DefaultImageBase = "https://image.tmdb.org/t/p"

type ContentProvider interface {
	CacheContentShow(content ShowDetails, onlyUpdate bool) (entity.Content, error)
	CacheContentMovie(content MovieDetails, onlyUpdate bool) (entity.Content, error)
}

type TMDB struct {
	Key string
	// Base url of the TMDB API. Overridable so tests can point at a stub.
	BaseURL string
	// Base url for TMDB images (poster downloads). Overridable for tests.
	ImageBaseURL    string
	client          *http.Client
	contentProvider ContentProvider
}

// NewTMDB creates a TMDB client. Empty base urls use the TMDB defaults.
func NewTMDB(key string, baseURL string, imageBaseURL string) *TMDB {
	if baseURL == "" {
		baseURL = DefaultAPIBase
	}
	if imageBaseURL == "" {
		imageBaseURL = DefaultImageBase
	}
	return &TMDB{
		Key:          key,
		BaseURL:      baseURL,
		ImageBaseURL: imageBaseURL,
		client:       &http.Client{Timeout: 15 * time.Second},
	}
}

func (t *TMDB) AddContentProvider(contentProvider ContentProvider) {
	t.contentProvider = contentProvider
}

func (t *TMDB) GetKey() string {
	if t.Key != "" {
		return t.Key //Config.TMDB_KEY
	}
	return "d047fa61d926371f277e7a83c9c4ff2c"
}

func (t *TMDB) apiRequest(ep string, p map[string]string) ([]byte, error) {
	slog.Debug("tmdbAPIRequest", "endpoint", ep, "params", p)
	base, err := url.Parse(t.BaseURL)
	if err != nil {
		return nil, errors.New("failed to parse api uri")
	}

	// Path params
	base.Path += ep

	// Query params
	params := url.Values{}
	params.Add("api_key", t.GetKey())
	params.Add("language", "en-US")
	for k, v := range p {
		params.Add(k, v)
	}

	// Add params to url
	base.RawQuery = params.Encode()

	// Run get request
	res, err := t.client.Get(base.String())
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		slog.Error("TMDB non 200 status code:", "status_code", res.StatusCode)
		return nil, errors.New(string(body))
	}
	return body, nil
}

func (t *TMDB) req(ep string, p map[string]string, resp interface{}) error {
	body, err := t.apiRequest(ep, p)
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(body), &resp)
	if err != nil {
		return err
	}
	return nil
}
