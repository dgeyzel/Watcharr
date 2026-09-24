package resolve

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"

	"github.com/sbondCo/Watcharr/media/tmdb"
	"github.com/sbondCo/Watcharr/util/safefetch"
)

var (
	// The site couldn't be read (blocked, rate limited, down, markup changed).
	ErrFetchBlocked = errors.New("couldn't read that page (the site may be blocking requests), paste an IMDb or TMDB link instead")
	// Nothing on TMDB matches.
	ErrNoMatch = errors.New("no matching title found on TMDB")
)

// Max candidates returned for a search based match (Rotten Tomatoes).
const maxCandidates = 5

type Candidate struct {
	TmdbID     int    `json:"tmdbId"`
	MediaType  string `json:"mediaType"`
	Title      string `json:"title"`
	Year       int    `json:"year"`
	PosterPath string `json:"posterPath"`
}

type Request struct {
	URL string `json:"url" binding:"required"`
	// Optional "movie" or "tv", used to pick between results when the url
	// doesn't say.
	MediaTypeHint string `json:"mediaTypeHint" binding:"omitempty,oneof=movie tv"`
}

// Response has one candidate for an exact match (TMDB, IMDb, Letterboxd),
// or up to 5 the admin picks from (Rotten Tomatoes).
type Response struct {
	Candidates []Candidate `json:"candidates"`
}

// TMDB is the part of the TMDB client the resolver uses.
type TMDB interface {
	MovieDetails(o tmdb.MovieDetailsOptions) (tmdb.MovieDetails, error)
	ShowDetails(o tmdb.ShowDetailsOptions) (tmdb.ShowDetails, error)
	SearchByExternalId(id string, source string) (tmdb.SearchMultiResponse, error)
	SearchMovies(o tmdb.SearchMoviesOptions) (tmdb.SearchMoviesResponse, error)
	SearchShows(o tmdb.SearchShowsOptions) (tmdb.SearchShowsResponse, error)
}

type Resolver struct {
	tmdb  TMDB
	fetch *safefetch.Fetcher
}

func New(t TMDB, fetch *safefetch.Fetcher) *Resolver {
	return &Resolver{tmdb: t, fetch: fetch}
}

func (r *Resolver) Resolve(ctx context.Context, req Request) (Response, error) {
	p, err := ParseURL(req.URL)
	if err != nil {
		return Response{}, err
	}
	var cands []Candidate
	switch p.Source {
	case SourceTMDB:
		cands, err = r.byTMDBID(p.MediaType, p.ID)
	case SourceIMDb:
		cands, err = r.byIMDbID(p.ID, req.MediaTypeHint)
	case SourceLetterboxd:
		cands, err = r.byLetterboxdPage(ctx, "https://letterboxd.com/film/"+p.Slug+"/")
	case SourceLetterboxdLink:
		cands, err = r.byLetterboxdPage(ctx, "https://boxd.it/"+p.Slug)
	case SourceRottenTomatoes:
		cands, err = r.byRottenTomatoes(ctx, p)
	default:
		return Response{}, ErrUnsupportedURL
	}
	if err != nil {
		return Response{}, err
	}
	if len(cands) == 0 {
		return Response{}, ErrNoMatch
	}
	return Response{Candidates: cands}, nil
}

func yearOf(date string) int {
	if len(date) >= 4 {
		y, _ := strconv.Atoi(date[:4])
		return y
	}
	return 0
}

func (r *Resolver) byTMDBID(mediaType string, id string) ([]Candidate, error) {
	switch mediaType {
	case "movie":
		d, err := r.tmdb.MovieDetails(tmdb.MovieDetailsOptions{ID: id, DontRunDBCache: true})
		if err != nil || d.ID == 0 {
			return nil, ErrNoMatch
		}
		return []Candidate{{TmdbID: d.ID, MediaType: "movie", Title: d.Title, Year: yearOf(d.ReleaseDate), PosterPath: d.PosterPath}}, nil
	case "tv":
		d, err := r.tmdb.ShowDetails(tmdb.ShowDetailsOptions{ID: id, DontRunDBCache: true})
		if err != nil || d.ID == 0 {
			return nil, ErrNoMatch
		}
		return []Candidate{{TmdbID: d.ID, MediaType: "tv", Title: d.Name, Year: yearOf(d.FirstAirDate), PosterPath: d.PosterPath}}, nil
	}
	return nil, ErrUnsupportedURL
}

func (r *Resolver) byIMDbID(id string, hint string) ([]Candidate, error) {
	res, err := r.tmdb.SearchByExternalId(id, "imdb")
	if err != nil {
		return nil, ErrNoMatch
	}
	cands := []Candidate{}
	for _, v := range res.Results {
		switch v.MediaType {
		case "movie":
			cands = append(cands, Candidate{TmdbID: v.ID, MediaType: "movie", Title: v.Title, Year: yearOf(v.ReleaseDate), PosterPath: v.PosterPath})
		case "tv":
			cands = append(cands, Candidate{TmdbID: v.ID, MediaType: "tv", Title: v.Name, Year: yearOf(v.FirstAirDate), PosterPath: v.PosterPath})
		}
	}
	if hint != "" && len(cands) > 1 {
		filtered := []Candidate{}
		for _, c := range cands {
			if c.MediaType == hint {
				filtered = append(filtered, c)
			}
		}
		if len(filtered) > 0 {
			cands = filtered
		}
	}
	return cands, nil
}

func (r *Resolver) byLetterboxdPage(ctx context.Context, pageURL string) ([]Candidate, error) {
	body, _, err := r.fetch.Get(ctx, pageURL)
	if err != nil {
		if errors.Is(err, safefetch.ErrNotFound) {
			return nil, ErrNoMatch
		}
		slog.Warn("resolve: letterboxd fetch failed", "url", pageURL, "error", err)
		return nil, ErrFetchBlocked
	}
	mediaType, id, ok := ParseLetterboxd(string(body))
	if !ok {
		slog.Warn("resolve: no tmdb id on letterboxd page", "url", pageURL)
		return nil, ErrFetchBlocked
	}
	return r.byTMDBID(mediaType, strconv.Itoa(id))
}

func (r *Resolver) byRottenTomatoes(ctx context.Context, p ParsedURL) ([]Candidate, error) {
	prefix := "m"
	if p.MediaType == "tv" {
		prefix = "tv"
	}
	pageURL := "https://www.rottentomatoes.com/" + prefix + "/" + p.Slug
	ty := TitleYearFromRTSlug(p.Slug)
	if body, _, err := r.fetch.Get(ctx, pageURL); err == nil {
		if parsed, ok := ParseRottenTomatoes(string(body)); ok {
			ty = parsed
		}
	} else {
		// Fall back to the slug, it's usually "title_year".
		slog.Info("resolve: rotten tomatoes fetch failed, using slug", "url", pageURL, "error", err)
	}
	if strings.TrimSpace(ty.Title) == "" {
		return nil, ErrNoMatch
	}
	return r.searchCandidates(p.MediaType, ty)
}

func (r *Resolver) searchCandidates(mediaType string, ty TitleYear) ([]Candidate, error) {
	cands := []Candidate{}
	if mediaType == "tv" {
		res, err := r.tmdb.SearchShows(tmdb.SearchShowsOptions{
			SearchUniversalOptions: tmdb.SearchUniversalOptions{Query: ty.Title, Page: 1},
			PrimaryYear:            ty.Year,
		})
		if err != nil {
			return nil, ErrNoMatch
		}
		for _, v := range res.Results {
			cands = append(cands, Candidate{TmdbID: v.ID, MediaType: "tv", Title: v.Name, Year: yearOf(v.FirstAirDate), PosterPath: v.PosterPath})
		}
	} else {
		res, err := r.tmdb.SearchMovies(tmdb.SearchMoviesOptions{
			SearchUniversalOptions: tmdb.SearchUniversalOptions{Query: ty.Title, Page: 1},
			PrimaryYear:            ty.Year,
		})
		if err != nil {
			return nil, ErrNoMatch
		}
		for _, v := range res.Results {
			cands = append(cands, Candidate{TmdbID: v.ID, MediaType: "movie", Title: v.Title, Year: yearOf(v.ReleaseDate), PosterPath: v.PosterPath})
		}
	}
	if len(cands) > maxCandidates {
		cands = cands[:maxCandidates]
	}
	return cands, nil
}
