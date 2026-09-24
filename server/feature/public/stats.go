package public

import (
	"errors"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/sbondCo/Watcharr/database/entity"
)

// StatsResponse is computed only from visible items. It never contains
// numeric ratings or averages of them.
type StatsResponse struct {
	Totals   StatsTotals   `json:"totals"`
	ByStatus StatsByStatus `json:"byStatus"`
	// Over finished and watching titles only (planned titles have no tier).
	Tiers         StatsTiers    `json:"tiers"`
	AddedPerMonth []MonthCount  `json:"addedPerMonth"`
	ByDecade      []DecadeCount `json:"byDecade"`
	TopGenres     []NameCount   `json:"topGenres"`
	Tags          []TagCount    `json:"tags"`
	// Total runtime of finished movies, in hours (one decimal).
	FinishedMovieHours float64 `json:"finishedMovieHours"`
}

type StatsTotals struct {
	Titles int `json:"titles"`
	Movies int `json:"movies"`
	Shows  int `json:"shows"`
}

type StatsByStatus struct {
	Finished int `json:"finished"`
	Watching int `json:"watching"`
	Planned  int `json:"planned"`
}

type StatsTiers struct {
	S int `json:"S"`
	A int `json:"A"`
	B int `json:"B"`
	C int `json:"C"`
	D int `json:"D"`
	F int `json:"F"`
	// Watched (finished/watching) but not yet rated.
	Unrated int `json:"unrated"`
}

type MonthCount struct {
	// YYYY-MM
	Month string `json:"month"`
	Count int    `json:"count"`
}

type DecadeCount struct {
	// e.g. 1990
	Decade int `json:"decade"`
	Count  int `json:"count"`
}

type NameCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type TagCount struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

const topGenresLimit = 10

func (s *Service) GetStats() (StatsResponse, error) {
	now := s.now().UTC()
	resp := StatsResponse{
		AddedPerMonth: lastTwelveMonths(now),
		ByDecade:      []DecadeCount{},
		TopGenres:     []NameCount{},
		Tags:          []TagCount{},
	}
	o, err := s.owner()
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return resp, nil
		}
		return resp, err
	}

	var watched []entity.Watched
	if res := s.visible(o.ID).Preload("Content").Preload("Tags").Find(&watched); res.Error != nil {
		slog.Error("public GetStats: query failed", "error", res.Error)
		return resp, res.Error
	}

	months := map[string]int{}
	for i, m := range resp.AddedPerMonth {
		months[m.Month] = i
	}
	decades := map[int]int{}
	genres := map[string]int{}
	tagCounts := map[uint]int{}
	var finishedMovieMinutes uint64

	for i := range watched {
		w := &watched[i]
		if w.Content == nil {
			continue
		}
		resp.Totals.Titles++
		if w.Content.Type == entity.MOVIE {
			resp.Totals.Movies++
		} else {
			resp.Totals.Shows++
		}

		switch w.Status {
		case entity.FINISHED:
			resp.ByStatus.Finished++
		case entity.WATCHING:
			resp.ByStatus.Watching++
		case entity.PLANNED:
			resp.ByStatus.Planned++
		}

		if w.Status == entity.FINISHED || w.Status == entity.WATCHING {
			addTier(&resp.Tiers, w.Tier)
		}

		if idx, ok := months[w.CreatedAt.UTC().Format("2006-01")]; ok {
			resp.AddedPerMonth[idx].Count++
		}

		if rd := w.Content.ReleaseDate; rd != nil && rd.Year() > 1 {
			decades[rd.Year()/10*10]++
		}

		for _, g := range strings.Split(w.Content.Genres, ",") {
			if g = strings.TrimSpace(g); g != "" {
				genres[g]++
			}
		}

		for _, t := range w.Tags {
			tagCounts[t.ID]++
		}

		if w.Status == entity.FINISHED && w.Content.Type == entity.MOVIE {
			finishedMovieMinutes += uint64(w.Content.Runtime)
		}
	}

	for d, c := range decades {
		resp.ByDecade = append(resp.ByDecade, DecadeCount{Decade: d, Count: c})
	}
	sort.Slice(resp.ByDecade, func(i, j int) bool { return resp.ByDecade[i].Decade < resp.ByDecade[j].Decade })

	for n, c := range genres {
		resp.TopGenres = append(resp.TopGenres, NameCount{Name: n, Count: c})
	}
	sort.Slice(resp.TopGenres, func(i, j int) bool {
		a, b := resp.TopGenres[i], resp.TopGenres[j]
		if a.Count != b.Count {
			return a.Count > b.Count
		}
		return a.Name < b.Name
	})
	if len(resp.TopGenres) > topGenresLimit {
		resp.TopGenres = resp.TopGenres[:topGenresLimit]
	}

	var tags []entity.Tag
	if res := s.db.Where("user_id = ?", o.ID).Order("name").Find(&tags); res.Error != nil {
		slog.Error("public GetStats: tags query failed", "error", res.Error)
		return resp, res.Error
	}
	for _, t := range tags {
		resp.Tags = append(resp.Tags, TagCount{ID: t.ID, Name: t.Name, Count: tagCounts[t.ID]})
	}

	resp.FinishedMovieHours = math.Round(float64(finishedMovieMinutes)/60*10) / 10
	return resp, nil
}

func addTier(g *StatsTiers, tier *entity.Tier) {
	if tier == nil {
		g.Unrated++
		return
	}
	switch *tier {
	case entity.TierS:
		g.S++
	case entity.TierA:
		g.A++
	case entity.TierB:
		g.B++
	case entity.TierC:
		g.C++
	case entity.TierD:
		g.D++
	case entity.TierF:
		g.F++
	default:
		g.Unrated++
	}
}

// lastTwelveMonths returns zeroed counts for the 12 months up to and
// including now's month, oldest first.
func lastTwelveMonths(now time.Time) []MonthCount {
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	out := make([]MonthCount, 0, 12)
	for i := 11; i >= 0; i-- {
		out = append(out, MonthCount{Month: first.AddDate(0, -i, 0).Format("2006-01")})
	}
	return out
}
