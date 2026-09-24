package server

import (
	"log/slog"

	"github.com/sbondCo/Watcharr/database/entity"
	"gorm.io/gorm"
)

type ServerStats struct {
	WatchedMovies    int64          `json:"watchedMovies"`
	WatchedShows     int64          `json:"watchedShows"`
	WatchedSeasons   int64          `json:"watchedSeasons"`
	MostWatchedMovie entity.Content `json:"mostWatchedMovie"`
	MostWatchedShow  entity.Content `json:"mostWatchedShow"`
	Activities       int64          `json:"activities"`
}

// Collect and return server stats
// I cant sql so this the best yall gettin
func getServerStats(db *gorm.DB) ServerStats {
	stats := ServerStats{}
	// Watched seasons count.
	resp := db.Model(&entity.WatchedSeason{}).Count(&stats.WatchedSeasons)
	if resp.Error != nil {
		slog.Error("getServerStats - WatchedSeasons query failed", "error", resp.Error)
	}
	// Activities count.
	resp = db.Model(&entity.Activity{}).Count(&stats.Activities)
	if resp.Error != nil {
		slog.Error("getServerStats - Activities query failed", "error", resp.Error)
	}
	// Watched shows count.
	resp = db.
		Joins("JOIN contents ON contents.id = watcheds.content_id AND contents.type = ?", "tv").
		Find(&entity.Watched{}).
		Count(&stats.WatchedShows)
	if resp.Error != nil {
		slog.Error("getServerStats - WatchedShows query failed", "error", resp.Error)
	}
	// Watched movies count.
	resp = db.
		Joins("JOIN contents ON contents.id = watcheds.content_id AND contents.type = ?", "movie").
		Find(&entity.Watched{}).
		Count(&stats.WatchedMovies)
	if resp.Error != nil {
		slog.Error("getServerStats - WatchedMovies query failed", "error", resp.Error)
	}
	// Most watched show.
	var w entity.Watched
	resp = db.
		Model(&entity.Watched{}).
		Select("content_id, COUNT(*) AS mag").
		Joins("JOIN contents ON contents.type = ? AND contents.id = watcheds.content_id", "tv").
		Group("content_id").
		Order("mag DESC").
		Preload("Content").
		First(&w)
	if resp.Error != nil {
		slog.Error("getServerStats - MostWatchedShow query failed", "error", resp.Error)
	} else {
		stats.MostWatchedShow = *w.Content
	}
	// Most watched movie.
	resp = db.
		Model(&entity.Watched{}).
		Select("content_id, COUNT(*) AS mag").
		Joins("JOIN contents ON contents.type = ? AND contents.id = watcheds.content_id", "movie").
		Group("content_id").
		Order("mag DESC").
		Preload("Content").
		First(&w)
	if resp.Error != nil {
		slog.Error("getServerStats - MostWatchedMovie query failed", "error", resp.Error)
	} else {
		stats.MostWatchedMovie = *w.Content
	}
	return stats
}
