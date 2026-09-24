// Package public is the unauthenticated, read only api visitors use. Every
// endpoint only ever returns "visible" items (see VisibleWatched) and uses
// dedicated response types (see dto.go).
package public

import (
	"errors"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/sbondCo/Watcharr/config"
	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/database/query"
	"github.com/sbondCo/Watcharr/media/tmdb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrNotFound = errors.New("not found")

const (
	defaultPageLimit = 40
	maxPageLimit     = 100
)

type Sort string

const (
	SortDateAdded    Sort = "DATEADDED"
	SortAlphabetical Sort = "ALPHA"
	SortDateReleased Sort = "DATERELEASED"
	// S to F (F to S with sortDir=asc), ungraded after graded, planned last.
	SortGrade Sort = "GRADE"
)

// ListRequest holds the (untrusted) query params for listing watched items.
type ListRequest struct {
	Page    int    `form:"page"`
	Limit   int    `form:"limit"`
	Sort    Sort   `form:"sort"`
	SortDir string `form:"sortDir"`
	// Comma separated list of statuses.
	Status string `form:"status"`
	// Comma separated list of media types (movie, tv).
	Type string `form:"type"`
	Tag  uint   `form:"tag"`
	// Comma separated grades (S..F) and/or "none" (watched but not rated).
	Grade string `form:"grade"`
	// Search the owner's list by title (never hits TMDB).
	Q string `form:"q"`
}

type Service struct {
	db   *gorm.DB
	cfg  *config.ServerConfig
	tmdb *tmdb.TMDB
	// Clock, overridable in tests.
	now func() time.Time
}

func NewService(db *gorm.DB, cfg *config.ServerConfig, tmdb *tmdb.TMDB) *Service {
	return &Service{db: db, cfg: cfg, tmdb: tmdb, now: time.Now}
}

// owner returns the site owner (the first admin).
func (s *Service) owner() (entity.User, error) {
	var u entity.User
	res := s.db.
		Where("(permissions & ?) != 0", entity.PERM_ADMIN).
		Order("id").
		Preload("Avatar").
		Limit(1).Find(&u)
	if res.Error != nil {
		slog.Error("public owner: query failed", "error", res.Error)
		return u, res.Error
	}
	// Find (not Take) so a miss isn't logged as an error.
	if res.RowsAffected == 0 {
		return u, ErrNotFound
	}
	return u, nil
}

func (s *Service) GetOwner() (OwnerResponse, error) {
	u, err := s.owner()
	if err != nil {
		return OwnerResponse{}, err
	}
	r := OwnerResponse{Username: u.Username, Bio: u.Bio}
	if u.Avatar.Path != "" {
		r.Avatar = &OwnerAvatar{Path: u.Avatar.Path, BlurHash: u.Avatar.BlurHash}
	}
	return r, nil
}

// visible starts a query over the owner's visible watched items.
func (s *Service) visible(ownerID uint) *gorm.DB {
	return s.db.Model(&entity.Watched{}).Scopes(VisibleWatched(ownerID))
}

// parseList turns a comma separated param into allowed values only.
func parseList[T ~string](raw string, allowed []T) []T {
	out := []T{}
	for _, v := range strings.Split(raw, ",") {
		v = strings.ToUpper(strings.TrimSpace(v))
		for _, a := range allowed {
			if strings.ToUpper(string(a)) == v {
				out = append(out, a)
			}
		}
	}
	return out
}

func (s *Service) ListWatched(req ListRequest) (WatchedPageResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 {
		req.Limit = defaultPageLimit
	}
	if req.Limit > maxPageLimit {
		req.Limit = maxPageLimit
	}
	resp := WatchedPageResponse{Page: req.Page, Limit: req.Limit, Results: []WatchedResponse{}}

	o, err := s.owner()
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return resp, nil
		}
		return resp, err
	}

	q := s.visible(o.ID).Joins("Content")
	if statuses := parseList(req.Status, PublicStatuses); len(statuses) > 0 {
		q = q.Where("watcheds.status IN ?", statuses)
	}
	if types := parseList(req.Type, PublicContentTypes); len(types) > 0 {
		q = q.Where("Content.type IN ?", types)
	}
	if req.Tag != 0 {
		q = q.Where("watcheds.id IN (?)",
			s.db.Table("watched_tags").Select("watched_id").Where("tag_id = ?", req.Tag))
	}
	if req.Grade != "" {
		q = query.FilterGrade(q, []string{req.Grade})
	}
	if search := strings.TrimSpace(req.Q); search != "" {
		q = q.Where("Content.title LIKE ?", "%"+search+"%")
	}

	if err := q.Count(&resp.TotalResults).Error; err != nil {
		slog.Error("public ListWatched: count failed", "error", err)
		return resp, err
	}
	resp.TotalPages = int(math.Ceil(float64(resp.TotalResults) / float64(req.Limit)))

	desc := strings.ToLower(req.SortDir) != "asc"
	ordered := q
	var col clause.Column
	colDesc := desc
	switch req.Sort {
	case SortGrade:
		ordered = query.OrderByGrade(ordered, !desc)
		// Then alphabetical within the same grade.
		col = clause.Column{Name: "`Content`.`title`", Raw: true}
		colDesc = false
	case SortAlphabetical:
		col = clause.Column{Name: "`Content`.`title`", Raw: true}
	case SortDateReleased:
		col = clause.Column{Name: "`Content`.`release_date`", Raw: true}
	default:
		col = clause.Column{Name: "watcheds.created_at"}
	}

	var watched []entity.Watched
	res := ordered.
		Preload("Tags").
		Order(clause.OrderByColumn{Column: col, Desc: colDesc}).
		// Stable order between pages.
		Order(clause.OrderByColumn{Column: clause.Column{Name: "watcheds.id"}, Desc: desc}).
		Offset((req.Page - 1) * req.Limit).
		Limit(req.Limit).
		Find(&watched)
	if res.Error != nil {
		slog.Error("public ListWatched: query failed", "error", res.Error)
		return resp, res.Error
	}
	for i := range watched {
		resp.Results = append(resp.Results, newWatchedResponse(&watched[i]))
	}
	return resp, nil
}

// contentType validates a media type path param.
func contentType(mediaType string) (entity.ContentType, bool) {
	switch mediaType {
	case "movie":
		return entity.MOVIE, true
	case "tv":
		return entity.SHOW, true
	}
	return "", false
}

// getVisible returns the visible watched entry for a title, or ErrNotFound.
func (s *Service) getVisible(mediaType string, tmdbID string) (entity.Watched, entity.User, error) {
	var w entity.Watched
	ct, ok := contentType(mediaType)
	id, err := strconv.Atoi(tmdbID)
	if !ok || err != nil {
		return w, entity.User{}, ErrNotFound
	}
	o, err := s.owner()
	if err != nil {
		return w, o, err
	}
	res := s.visible(o.ID).
		Joins("Content").
		Preload("Tags").
		Where("Content.type = ? AND Content.tmdb_id = ?", ct, id).
		Limit(1).Find(&w)
	if res.Error != nil {
		slog.Error("public getVisible: query failed", "error", res.Error)
		return w, o, res.Error
	}
	if res.RowsAffected == 0 {
		return w, o, ErrNotFound
	}
	return w, o, nil
}

func (s *Service) GetWatched(mediaType string, tmdbID string) (WatchedResponse, error) {
	w, _, err := s.getVisible(mediaType, tmdbID)
	if err != nil {
		return WatchedResponse{}, err
	}
	return newWatchedResponse(&w), nil
}

// GetContent returns TMDB details, only for visible titles, so the public api
// can never be used to look up arbitrary TMDB content.
func (s *Service) GetContent(mediaType string, tmdbID string) (ContentResponse, error) {
	w, o, err := s.getVisible(mediaType, tmdbID)
	if err != nil {
		return ContentResponse{}, err
	}
	country := s.cfg.DEFAULT_COUNTRY
	if o.Country != nil && *o.Country != "" {
		country = *o.Country
	}
	id := strconv.Itoa(w.Content.TmdbID)
	switch w.Content.Type {
	case entity.MOVIE:
		m, err := s.tmdb.MovieDetails(tmdb.MovieDetailsOptions{
			ID:      id,
			Country: country,
			Params:  map[string]string{"append_to_response": "videos,watch/providers"},
		})
		if err != nil {
			return ContentResponse{}, err
		}
		return newContentResponse(entity.MOVIE, m.AsMedia()), nil
	case entity.SHOW:
		m, err := s.tmdb.ShowDetails(tmdb.ShowDetailsOptions{
			ID:      id,
			Country: country,
			Params:  map[string]string{"append_to_response": "videos,watch/providers,external_ids"},
		})
		if err != nil {
			return ContentResponse{}, err
		}
		return newContentResponse(entity.SHOW, m.AsMedia()), nil
	}
	return ContentResponse{}, ErrNotFound
}

func (s *Service) ListTags() ([]TagResponse, error) {
	r := []TagResponse{}
	o, err := s.owner()
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return r, nil
		}
		return r, err
	}
	var tags []entity.Tag
	if res := s.db.Where("user_id = ?", o.ID).Order("name").Find(&tags); res.Error != nil {
		slog.Error("public ListTags: query failed", "error", res.Error)
		return r, res.Error
	}
	for _, t := range tags {
		r = append(r, newTagResponse(t))
	}
	return r, nil
}

func (s *Service) GetTag(id string) (TagResponse, error) {
	tid, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return TagResponse{}, ErrNotFound
	}
	o, err := s.owner()
	if err != nil {
		return TagResponse{}, err
	}
	var t entity.Tag
	res := s.db.Where("id = ? AND user_id = ?", tid, o.ID).Limit(1).Find(&t)
	if res.Error != nil {
		return TagResponse{}, res.Error
	}
	if res.RowsAffected == 0 {
		return TagResponse{}, ErrNotFound
	}
	return newTagResponse(t), nil
}
