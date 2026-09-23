// Package app builds the fully wired Gin engine (services, routes and
// middleware). It is shared by the main binary and by tests, so tests run
// against exactly the same routing as production.
package app

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/sbondCo/Watcharr/config"
	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/feature/activity"
	"github.com/sbondCo/Watcharr/feature/arr"
	"github.com/sbondCo/Watcharr/feature/auth"
	"github.com/sbondCo/Watcharr/feature/content"
	"github.com/sbondCo/Watcharr/feature/discover"
	"github.com/sbondCo/Watcharr/feature/feature"
	"github.com/sbondCo/Watcharr/feature/follow"
	"github.com/sbondCo/Watcharr/feature/game"
	"github.com/sbondCo/Watcharr/feature/img"
	"github.com/sbondCo/Watcharr/feature/imprt"
	"github.com/sbondCo/Watcharr/feature/jellyfin"
	"github.com/sbondCo/Watcharr/feature/job"
	"github.com/sbondCo/Watcharr/feature/plex"
	"github.com/sbondCo/Watcharr/feature/profile"
	"github.com/sbondCo/Watcharr/feature/search"
	"github.com/sbondCo/Watcharr/feature/server"
	"github.com/sbondCo/Watcharr/feature/setup"
	"github.com/sbondCo/Watcharr/feature/tag"
	"github.com/sbondCo/Watcharr/feature/task"
	"github.com/sbondCo/Watcharr/feature/user"
	"github.com/sbondCo/Watcharr/feature/watched"
	"github.com/sbondCo/Watcharr/feature/watched/episode"
	"github.com/sbondCo/Watcharr/feature/watched/season"
	"github.com/sbondCo/Watcharr/media/tmdb"
	"github.com/sbondCo/Watcharr/router"
	"gorm.io/gorm"
)

type Options struct {
	// If set, requests that match no route (and are not under /api) are
	// reverse proxied to the UI server at this host:port.
	UIProxyHost string
}

// NewEngine creates the Gin engine with every feature's routes registered.
func NewEngine(db *gorm.DB, cfg *config.ServerConfig, opts Options) *gin.Engine {
	gine := gin.Default()

	// Register our custom validators
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("validsearchtype", domain.ValidSearchType)
		v.RegisterValidation("validdiscoverfilter", domain.ValidDiscoverFilter)
	}

	gine.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{
			"Content-Type",
			"Content-Length",
			"Accept-Encoding",
			"X-CSRF-Token",
			"Authorization",
			"accept",
			"origin",
			"Cache-Control",
			"X-Requested-With",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"watcharr-lastviewedseason-saved",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	gine.NoRoute(noRouteHandler(opts.UIProxyHost))

	api := gine.Group("/api")
	br := router.NewBaseRouter(db, api, cfg)

	tmdbService := tmdb.NewTMDB(cfg.TMDB_KEY, cfg.TMDBAPIBase(), cfg.TMDBImageBase())

	contentService := content.NewService(db, tmdbService)
	tmdbService.AddContentProvider(contentService)
	plexService := plex.NewService(cfg)
	authService := auth.NewService(db, cfg, plexService)
	authTrustedHeaderService := auth.NewTrustedHeaderService(db, cfg, authService)
	activityService := activity.NewService(db)
	userService := user.NewService(db)
	userManageService := user.NewManageService(db)
	gameService := game.NewService(db, &br.Cfg.TWITCH, activityService)
	watchedService := watched.NewService(
		db,
		contentService,
		gameService,
		activityService,
		userService)
	watchedSeasonService := season.NewService(db, activityService)
	watchedEpisodeService := episode.NewService(
		db,
		watchedService,
		watchedSeasonService,
		tmdbService,
		activityService,
		userService)
	jellyfinService := jellyfin.NewService(cfg)
	jellyfinSyncService := jellyfin.NewSyncService(
		cfg,
		jellyfinService,
		watchedService,
		watchedSeasonService,
		watchedEpisodeService,
		activityService)
	plexSyncService := plex.NewSyncService(
		plexService,
		watchedService,
		watchedSeasonService,
		watchedEpisodeService,
		activityService)
	featureService := feature.NewService(cfg)
	profileService := profile.NewService(db)
	followService := follow.NewService(db)
	tagService := tag.NewService(db, watchedService)
	searchService := search.NewService(db, br.Cfg, tmdbService, watchedService)
	discoverService := discover.NewService(db, br.Cfg, tmdbService)
	importService := imprt.NewService(
		db,
		watchedService,
		watchedSeasonService,
		watchedEpisodeService,
		tmdbService,
		activityService,
		tagService,
		searchService)
	importTraktService := imprt.NewTraktService(importService)

	auth.NewRouter(br, authService, authTrustedHeaderService).AddRoutes()
	content.NewRouter(br, contentService, watchedService, tmdbService).AddRoutes()
	watched.NewRouter(br, watchedService).AddRoutes()
	season.NewRouter(br, watchedSeasonService).AddRoutes()
	episode.NewRouter(br, watchedEpisodeService).AddRoutes()
	activity.NewRouter(br, activityService).AddRoutes()
	profile.NewRouter(br, profileService).AddRoutes()
	jellyfin.NewRouter(br, jellyfinService, jellyfinSyncService).AddRoutes()
	plex.NewRouter(br, plexSyncService).AddRoutes()
	user.NewRouter(br, userService, userManageService).AddRoutes()
	follow.NewRouter(br, followService).AddRoutes()
	imprt.NewRouter(br, importService, importTraktService).AddRoutes()
	server.NewRouter(br, plexService, authTrustedHeaderService, userManageService).AddRoutes()
	feature.NewRouter(br, featureService).AddRoutes()
	arr.NewRouter(br, contentService).AddRoutes()
	job.NewRouter(br).AddRoutes()
	task.NewRouter(br).AddRoutes()
	tag.NewRouter(br, tagService).AddRoutes()
	game.NewRouter(br, gameService, watchedService).AddRoutes()
	search.NewRouter(br, searchService, watchedService).AddRoutes()
	discover.NewRouter(br, discoverService, watchedService).AddRoutes()
	img.NewRouter(br).AddRoutes()

	// Only add setup routes if there are no users found in db.
	var userCount int64
	if uresp := db.Model(&entity.User{}).Count(&userCount); uresp.Error == nil {
		if userCount != 0 {
			slog.Debug("registered users found.. skipped creating setup routes.")
		} else {
			slog.Info("No users found.. creating setup routes.")
			setup.NewRouter(br, authService).AddRoutes()
		}
	} else {
		slog.Error("Failed to check if any users exist.. not registering setup routes",
			"error", uresp.Error)
	}

	return gine
}

// noRouteHandler returns a JSON 404 for unknown /api routes (so removed api
// routes never fall through to the UI), and otherwise proxies to the UI
// server when uiHost is set.
func noRouteHandler(uiHost string) gin.HandlerFunc {
	var proxy *httputil.ReverseProxy
	if uiHost != "" {
		proxy = &httputil.ReverseProxy{Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = uiHost
		}}
	}
	return func(c *gin.Context) {
		p := c.Request.URL.Path
		if proxy == nil || p == "/api" || strings.HasPrefix(p, "/api/") {
			c.JSON(http.StatusNotFound, router.ErrorResponse{Error: "not found"})
			return
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
