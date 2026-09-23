package public

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sbondCo/Watcharr/router"
)

type Router struct {
	br      *router.BaseRouter
	service *Service
}

func NewRouter(br *router.BaseRouter, service *Service) *Router {
	return &Router{br: br, service: service}
}

// All routes are GET and unauthenticated (they are on the public allowlist).
func (r *Router) AddRoutes() {
	// Generous per ip limit, just enough to stop someone hammering the api.
	limiter := router.NewRateLimiter(250*time.Millisecond, 120)
	public := r.br.Router.Group("/public").Use(limiter.Middleware())

	public.GET("/owner", noCache, r.GetOwner)
	public.GET("/watched", noCache, r.ListWatched)
	public.GET("/watched/:mediaType/:tmdbId", noCache, r.GetWatched)
	public.GET("/content/:mediaType/:tmdbId", cacheFor(5*time.Minute), r.GetContent)
	public.GET("/tags", noCache, r.ListTags)
	public.GET("/tag/:id", noCache, r.GetTag)
	public.GET("/tag/:id/watched", noCache, r.ListTagWatched)
}

// Owner edits must show up for visitors straight away, so these can be
// stored but must be revalidated.
func noCache(c *gin.Context) {
	c.Header("Cache-Control", "no-cache")
	c.Next()
}

func cacheFor(d time.Duration) gin.HandlerFunc {
	v := "public, max-age=" + strconv.Itoa(int(d.Seconds()))
	return func(c *gin.Context) {
		c.Header("Cache-Control", v)
		c.Next()
	}
}

func respond[T any](c *gin.Context, v T, err error) {
	if err != nil {
		// Never reveal why (hidden, admin only status, etc).
		c.Header("Cache-Control", "no-store")
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, router.ErrorResponse{Error: "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, router.ErrorResponse{Error: "something went wrong"})
		return
	}
	c.JSON(http.StatusOK, v)
}

func (r *Router) GetOwner(c *gin.Context) {
	v, err := r.service.GetOwner()
	respond(c, v, err)
}

func (r *Router) ListWatched(c *gin.Context) {
	var req ListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "invalid query params"})
		return
	}
	v, err := r.service.ListWatched(req)
	respond(c, v, err)
}

func (r *Router) GetWatched(c *gin.Context) {
	v, err := r.service.GetWatched(c.Param("mediaType"), c.Param("tmdbId"))
	respond(c, v, err)
}

func (r *Router) GetContent(c *gin.Context) {
	v, err := r.service.GetContent(c.Param("mediaType"), c.Param("tmdbId"))
	respond(c, v, err)
}

func (r *Router) ListTags(c *gin.Context) {
	v, err := r.service.ListTags()
	respond(c, v, err)
}

func (r *Router) GetTag(c *gin.Context) {
	v, err := r.service.GetTag(c.Param("id"))
	respond(c, v, err)
}

func (r *Router) ListTagWatched(c *gin.Context) {
	tag, err := r.service.GetTag(c.Param("id"))
	if err != nil {
		respond(c, WatchedPageResponse{}, err)
		return
	}
	var req ListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "invalid query params"})
		return
	}
	req.Tag = tag.ID
	v, err := r.service.ListWatched(req)
	respond(c, v, err)
}
