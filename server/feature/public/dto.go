package public

import (
	"time"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/domain"
)

// Dedicated response types for the public api. They are built field by field
// from entities, so private data (numeric ratings, user ids, activity,
// settings, hidden items) can never leak by adding a field to an entity.
//
// Fields intentionally have no `omitempty`, so the key set of every response
// is fixed (tests assert the exact keys).

type OwnerResponse struct {
	Username string       `json:"username"`
	Bio      string       `json:"bio"`
	Avatar   *OwnerAvatar `json:"avatar"`
}

type OwnerAvatar struct {
	Path     string `json:"path"`
	BlurHash string `json:"blurHash"`
}

type TagResponse struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	Color   string `json:"color"`
	BgColor string `json:"bgColor"`
}

type WatchedResponse struct {
	// "movie" or "tv".
	MediaType   entity.ContentType   `json:"mediaType"`
	TmdbID      int                  `json:"tmdbId"`
	Title       string               `json:"title"`
	PosterPath  string               `json:"posterPath"`
	ReleaseDate *time.Time           `json:"releaseDate"`
	Status      entity.WatchedStatus `json:"status"`
	// S-F grade or null. Always null for planned titles (they have no grade
	// slot), even if a grade is saved.
	Grade     *entity.Grade `json:"grade"`
	Review    string        `json:"review"`
	Tags      []TagResponse `json:"tags"`
	CreatedAt time.Time     `json:"createdAt"`
	UpdatedAt time.Time     `json:"updatedAt"`
}

type WatchedPageResponse struct {
	Page         int               `json:"page"`
	Limit        int               `json:"limit"`
	TotalPages   int               `json:"totalPages"`
	TotalResults int64             `json:"totalResults"`
	Results      []WatchedResponse `json:"results"`
}

// ContentResponse is TMDB details for a visible title. It has no TMDB vote
// score either, visitors never see a numeric rating.
type ContentResponse struct {
	MediaType             entity.ContentType     `json:"mediaType"`
	TmdbID                int                    `json:"tmdbId"`
	Title                 string                 `json:"title"`
	Overview              string                 `json:"overview"`
	PosterPath            string                 `json:"posterPath"`
	BackdropPath          string                 `json:"backdropPath"`
	Genres                []domain.MediaGenre    `json:"genres"`
	Homepage              string                 `json:"homepage"`
	ReleaseDate           *time.Time             `json:"releaseDate"`
	ReleaseDateLast       *time.Time             `json:"releaseDateLast"`
	Runtime               uint                   `json:"runtime"`
	Status                string                 `json:"status"`
	Videos                []domain.MediaVideo    `json:"videos"`
	Seasons               []domain.MediaSeason   `json:"seasons"`
	Providers             []domain.MediaProvider `json:"providers"`
	ProvidersFullListLink string                 `json:"providersFullListLink"`
}

func newTagResponse(t entity.Tag) TagResponse {
	return TagResponse{ID: t.ID, Name: t.Name, Color: t.Color, BgColor: t.BgColor}
}

func newWatchedResponse(w *entity.Watched) WatchedResponse {
	r := WatchedResponse{
		Status:    w.Status,
		Review:    w.Thoughts,
		Grade:     publicGrade(w),
		Tags:      []TagResponse{},
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
	}
	if c := w.Content; c != nil {
		r.MediaType = c.Type
		r.TmdbID = c.TmdbID
		r.Title = c.Title
		r.PosterPath = c.PosterPath
		r.ReleaseDate = c.ReleaseDate
	}
	for _, t := range w.Tags {
		r.Tags = append(r.Tags, newTagResponse(t))
	}
	return r
}

func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func newContentResponse(t entity.ContentType, m domain.Media) ContentResponse {
	return ContentResponse{
		MediaType:             t,
		TmdbID:                m.IDs.TMDB,
		Title:                 m.Name,
		Overview:              m.Summary,
		PosterPath:            m.ExtPosterPath,
		BackdropPath:          m.ExtBackdropPath,
		Genres:                nonNil(m.Genres),
		Homepage:              m.Homepage,
		ReleaseDate:           timePtr(m.ReleaseDate),
		ReleaseDateLast:       timePtr(m.ReleaseDateLast),
		Runtime:               m.Runtime,
		Status:                m.Status,
		Videos:                nonNil(m.Videos),
		Seasons:               nonNil(m.Seasons),
		Providers:             nonNil(m.Providers),
		ProvidersFullListLink: m.ProvidersFullListLink,
	}
}

// publicGrade is the grade visitors see: none while a title is planned.
func publicGrade(w *entity.Watched) *entity.Grade {
	if w.Status == entity.PLANNED {
		return nil
	}
	return w.Grade
}
