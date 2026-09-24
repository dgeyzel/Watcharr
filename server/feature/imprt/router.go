package imprt

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/feature/auth/authmiddleware"
	"github.com/sbondCo/Watcharr/feature/resolve"
	"github.com/sbondCo/Watcharr/router"
)

type Router struct {
	br           *router.BaseRouter
	service      *Service
	traktService *TraktService
	resolver     *resolve.Resolver
}

func NewRouter(br *router.BaseRouter, service *Service, traktService *TraktService, resolver *resolve.Resolver) *Router {
	return &Router{
		br,
		service,
		traktService,
		resolver,
	}
}

func (r *Router) AddRoutes() {
	imprt := r.br.Router.Group("/import").Use(authmiddleware.AuthRequired(nil, r.br.Cfg))

	imprt.POST("", r.ImportContent)
	imprt.POST("/trakt", r.ImportTrakt)
	// Resolve a pasted TMDB/IMDb/Letterboxd/Rotten Tomatoes url to TMDB
	// candidates (to fix failed imports, or add a title by url).
	imprt.POST("/resolve", r.Resolve)
}

// Import content (the client handle processing data and sends it to us in a uniform way).
func (r *Router) ImportContent(c *gin.Context) {
	userId := c.MustGet("userId").(uint)
	var ar domain.ImportRequest
	err := c.ShouldBindJSON(&ar)
	if err == nil {
		response, err := r.service.ImportContent(userId, ar)
		if err != nil {
			c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusOK, response)
		return
	}
	c.AbortWithStatusJSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
}

// Import Trakt.
func (r *Router) ImportTrakt(c *gin.Context) {
	userId := c.MustGet("userId").(uint)
	var ar TraktImportRequest
	err := c.ShouldBindJSON(&ar)
	if err == nil {
		response, err := r.traktService.TraktImportWatched(userId, ar)
		if err != nil {
			c.JSON(http.StatusForbidden, router.ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusOK, response)
		return
	}
	c.AbortWithStatusJSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
}

// Resolve a pasted url to TMDB candidates. Nothing is imported here, the
// client imports the candidate the admin confirms.
func (r *Router) Resolve(c *gin.Context) {
	var req resolve.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "a url is required"})
		return
	}
	resp, err := r.resolver.Resolve(c.Request.Context(), req)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, resolve.ErrUnsupportedURL):
			status = http.StatusBadRequest
		case errors.Is(err, resolve.ErrFetchBlocked):
			status = http.StatusUnprocessableEntity
		case errors.Is(err, resolve.ErrNoMatch):
			status = http.StatusNotFound
		}
		c.JSON(status, router.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}
