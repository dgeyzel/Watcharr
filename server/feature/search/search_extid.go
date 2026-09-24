package search

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/feature/resolve"
)

// Perform "special"  direct search if possible using search query.
// Eg: Search term is in provider:id format or is a supported url.
func (s *Service) searchExtProviderById(
	query string,
	resp *domain.SearchResponse,
) bool {
	queryLower := strings.ToLower(query)

	provider, providerID := s.getExtProviderFromQuery(queryLower)

	if provider == "" || providerID == "" {
		return false
	}

	slog.Debug("searchExtProviderById: Processing.",
		"provider", provider,
		"provider_id", providerID)

	switch provider {
	case "movie":
		if err := s.searchMovieById(providerID, resp); err == nil {
			return true
		}
	case "tv":
		if err := s.searchTvById(providerID, resp); err == nil {
			return true
		}
	default:
		// By default, if provider name isn't caught in above cases, just send
		// it to tmdb external id search.
		tmdbRes, err := s.tmdb.SearchByExternalId(
			providerID,
			provider,
		)
		if err != nil {
			slog.Error("searchExtProviderById: Failed to search tmdb!",
				"error", err)
			return false
		}
		resLen := len(tmdbRes.Results)
		if resLen <= 0 {
			return false
		}
		for _, v := range tmdbRes.Results {
			resp.Results = append(
				resp.Results,
				v.AsMedia(),
			)
		}
		resp.Page = 1
		resp.TotalPages = 1
		resp.TotalResults = int64(resLen)
		return true
	}

	return false
}

// Takes in query and returns (Provider, ProviderID) if found.
func (s *Service) getExtProviderFromQuery(queryLower string) (string, string) {
	var provider string

	// Before checking for provider:providerid format, check if query is
	// a supported url.
	if p, i := s.getExtProviderFromURL(queryLower); p != "" && i != "" {
		slog.Debug("getExtProviderFromQuery: Returning from parsed url.")
		return p, i
	}

	querySplit := strings.Split(queryLower, ":")

	if len(querySplit) != 2 {
		slog.Debug("getExtProviderFromQuery: querySplit len != 2")
		return "", ""
	}

	switch querySplit[0] {
	case "movie", // TMDB ID target
		"tv", // TMDB ID target
		// The rest below are sent as is to tmdbs find by (external) id api.
		"imdb",
		"tvdb",
		"youtube",
		"wikidata",
		"facebook",
		"instagram",
		"twitter",
		"tiktok":
		provider = querySplit[0]
		// Any aliases we want to support
	case "i", "imd":
		provider = "imdb"
	case "wd", "wdt":
		provider = "wikidata"
	case "yt":
		provider = "youtube"
	case "thetvdb":
		provider = "tvdb"
	case "series":
		provider = "tv"
	default:
		slog.Debug("getExtProviderFromQuery: No provider found.")
		return "", ""
	}

	return provider, querySplit[1]
}

// Takes in what may be a url. If it is a supported TMDB or IMDb url, returns
// (Provider, ProviderID). Host matching is strict (see resolve.ParseURL), so
// lookalike hosts never match.
func (s *Service) getExtProviderFromURL(maybeaurl string) (string, string) {
	if !strings.Contains(maybeaurl, "://") && !strings.Contains(maybeaurl, "/") {
		return "", ""
	}
	p, err := resolve.ParseURL(maybeaurl)
	if err != nil {
		return "", ""
	}
	switch p.Source {
	case resolve.SourceTMDB:
		return p.MediaType, p.ID
	case resolve.SourceIMDb:
		return "imdb", p.ID
	}
	return "", ""
}

// searchByResolvedURL handles Letterboxd and Rotten Tomatoes urls (which
// need their page fetched) through the resolver.
func (s *Service) searchByResolvedURL(query string, resp *domain.SearchResponse) bool {
	if s.resolver == nil {
		return false
	}
	p, err := resolve.ParseURL(query)
	if err != nil || p.Source == resolve.SourceTMDB || p.Source == resolve.SourceIMDb {
		return false
	}
	res, err := s.resolver.Resolve(context.Background(), resolve.Request{URL: query})
	if err != nil {
		slog.Info("searchByResolvedURL: couldn't resolve url", "error", err)
		return false
	}
	for _, c := range res.Candidates {
		m := domain.Media{
			IDs:           domain.MediaIDs{TMDB: c.TmdbID},
			Name:          c.Title,
			ExtPosterPath: c.PosterPath,
			Type:          domain.MediaTypeTMDBMovie,
		}
		if c.MediaType == "tv" {
			m.Type = domain.MediaTypeTMDBShow
		}
		if c.Year > 0 {
			m.ReleaseDate = time.Date(c.Year, 1, 1, 0, 0, 0, 0, time.UTC)
		}
		resp.Results = append(resp.Results, m)
	}
	resp.Page = 1
	resp.TotalPages = 1
	resp.TotalResults = int64(len(resp.Results))
	return len(resp.Results) > 0
}
