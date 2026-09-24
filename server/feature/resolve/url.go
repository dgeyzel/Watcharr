// Package resolve turns a pasted TMDB, IMDb, Letterboxd or Rotten Tomatoes
// url into TMDB candidates (for fixing failed imports and adding by url).
package resolve

import (
	"errors"
	"net/url"
	"regexp"
	"strings"

	"github.com/sbondCo/Watcharr/util"
)

type Source string

const (
	SourceTMDB           Source = "tmdb"
	SourceIMDb           Source = "imdb"
	SourceLetterboxd     Source = "letterboxd"
	SourceLetterboxdLink Source = "boxd"
	SourceRottenTomatoes Source = "rottentomatoes"
)

// Domains each source may be on (exact domain or a real subdomain).
var sourceDomains = map[Source]string{
	SourceTMDB:           "themoviedb.org",
	SourceIMDb:           "imdb.com",
	SourceLetterboxd:     "letterboxd.com",
	SourceLetterboxdLink: "boxd.it",
	SourceRottenTomatoes: "rottentomatoes.com",
}

// FetchDomains are the only sites we ever fetch pages from.
var FetchDomains = []string{
	sourceDomains[SourceLetterboxd],
	sourceDomains[SourceLetterboxdLink],
	sourceDomains[SourceRottenTomatoes],
}

var ErrUnsupportedURL = errors.New("unsupported url, paste a TMDB, IMDb, Letterboxd or Rotten Tomatoes link")

// ParsedURL is a classified url.
type ParsedURL struct {
	Source Source
	// "movie" or "tv" when the url says (TMDB, Rotten Tomatoes).
	MediaType string
	// TMDB numeric id or IMDb tt id.
	ID string
	// Letterboxd film slug, Rotten Tomatoes slug or boxd.it code.
	Slug string
}

var (
	imdbID     = regexp.MustCompile(`^tt\d{5,10}$`)
	tmdbIDPart = regexp.MustCompile(`^(\d+)(-.*)?$`)
	slugPart   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)
)

// ParseURL classifies a pasted url. A missing scheme is allowed
// ("imdb.com/title/tt0137523").
func ParseURL(raw string) (ParsedURL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ParsedURL{}, ErrUnsupportedURL
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil {
		return ParsedURL{}, ErrUnsupportedURL
	}
	segs := []string{}
	for _, s := range strings.Split(u.Path, "/") {
		if s != "" {
			segs = append(segs, s)
		}
	}
	for src, domain := range sourceDomains {
		if !util.HostMatchesDomain(u.Host, domain) {
			continue
		}
		var p ParsedURL
		var ok bool
		switch src {
		case SourceTMDB:
			p, ok = parseTMDB(segs)
		case SourceIMDb:
			p, ok = parseIMDb(segs)
		case SourceLetterboxd:
			p, ok = parseLetterboxd(segs)
		case SourceLetterboxdLink:
			p, ok = parseBoxd(segs)
		case SourceRottenTomatoes:
			p, ok = parseRT(segs)
		}
		if !ok {
			return ParsedURL{}, ErrUnsupportedURL
		}
		p.Source = src
		return p, nil
	}
	return ParsedURL{}, ErrUnsupportedURL
}

// themoviedb.org/movie/{id}[-slug], /tv/{id}[-slug] (optionally prefixed by
// a language, e.g. /en/movie/550).
func parseTMDB(segs []string) (ParsedURL, bool) {
	for i := 0; i+1 < len(segs); i++ {
		t := strings.ToLower(segs[i])
		if t != "movie" && t != "tv" {
			continue
		}
		if m := tmdbIDPart.FindStringSubmatch(segs[i+1]); m != nil {
			return ParsedURL{MediaType: t, ID: m[1]}, true
		}
		return ParsedURL{}, false
	}
	return ParsedURL{}, false
}

// imdb.com/title/tt123 (any trailing path, m. and www. subdomains).
func parseIMDb(segs []string) (ParsedURL, bool) {
	for i := 0; i+1 < len(segs); i++ {
		if strings.ToLower(segs[i]) == "title" {
			id := strings.ToLower(segs[i+1])
			if imdbID.MatchString(id) {
				return ParsedURL{ID: id}, true
			}
			return ParsedURL{}, false
		}
	}
	return ParsedURL{}, false
}

// letterboxd.com/film/{slug}/... and diary/review paths
// letterboxd.com/{user}/film/{slug}/..., both become the film slug.
func parseLetterboxd(segs []string) (ParsedURL, bool) {
	for i := 0; i+1 < len(segs) && i <= 1; i++ {
		if strings.ToLower(segs[i]) == "film" && slugPart.MatchString(segs[i+1]) {
			return ParsedURL{Slug: strings.ToLower(segs[i+1])}, true
		}
	}
	return ParsedURL{}, false
}

// boxd.it/{code} short links (resolved by following the redirect).
func parseBoxd(segs []string) (ParsedURL, bool) {
	if len(segs) == 1 && slugPart.MatchString(segs[0]) {
		return ParsedURL{Slug: segs[0]}, true
	}
	return ParsedURL{}, false
}

// rottentomatoes.com/m/{slug}, /tv/{slug}.
func parseRT(segs []string) (ParsedURL, bool) {
	if len(segs) < 2 || !slugPart.MatchString(segs[1]) {
		return ParsedURL{}, false
	}
	switch strings.ToLower(segs[0]) {
	case "m":
		return ParsedURL{MediaType: "movie", Slug: strings.ToLower(segs[1])}, true
	case "tv":
		return ParsedURL{MediaType: "tv", Slug: strings.ToLower(segs[1])}, true
	}
	return ParsedURL{}, false
}
