package server

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/cache"
	"github.com/gin-gonic/gin"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/feature/auth/authmiddleware"
	"github.com/sbondCo/Watcharr/router"
)

type Router struct {
	br                 *router.BaseRouter
	userManageProvider domain.UserManageProvider
}

func NewRouter(
	br *router.BaseRouter,
	userManageProvider domain.UserManageProvider,
) *Router {
	return &Router{
		br,
		userManageProvider,
	}
}

func (r *Router) AddRoutes() {
	server := r.br.Router.Group("/server").Use(authmiddleware.AuthRequired(r.br.DB, r.br.Cfg), authmiddleware.AdminRequired())

	// Get server config (minus very sensitive fields, like JWT_SECRET)
	server.GET("/config", r.GetConfig)
	// Update config
	server.POST("/config", r.UpdateConfig)
	// Get server stats
	server.GET("/stats", cache.CachePage(r.br.MemStore, time.Minute*5, r.GetStats))
	// Get all server users (for manage users page)
	server.GET("/users", r.GetAllUsers)
	// Edit a user (for manage users page)
	server.POST("/users/:id", r.UpdateManageUser)
}

// Get server config (minus very sensitive fields, like JWT_SECRET)
func (r *Router) GetConfig(c *gin.Context) {
	// s should be provided when asking for the value of just one setting.
	s := c.Query("s")
	if s != "" {
		val, err := r.br.Cfg.Get(s)
		if err != nil {
			c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusOK, val)
		return
	}
	// Return new ServerConfig with only the fields we want to show in settings ui
	c.JSON(http.StatusOK, r.br.Cfg.GetSafe())
}

// Update config
func (r *Router) UpdateConfig(c *gin.Context) {
	var ur router.KeyValueRequest
	err := c.ShouldBindJSON(&ur)
	if err == nil {
		err := r.br.Cfg.UpdateConfig(ur.Key, ur.Value)
		if err != nil {
			c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
			return
		}
		c.Status(http.StatusOK)
		return
	}
	c.AbortWithStatusJSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
}

// Get server stats
func (r *Router) GetStats(c *gin.Context) {
	c.JSON(http.StatusOK, getServerStats(r.br.DB))
}

// Get all server users (for manage users page)
func (r *Router) GetAllUsers(c *gin.Context) {
	resp, err := r.userManageProvider.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, router.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Edit a user (for manage users page)
func (r *Router) UpdateManageUser(c *gin.Context) {
	userId, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		slog.Error("/users/:id failed to parse id as a uint", "error", err)
		c.JSON(http.StatusInternalServerError, router.ErrorResponse{Error: "failed to parse id"})
		return
	}
	var ur domain.UpdateUserRequest
	err = c.ShouldBindJSON(&ur)
	if err == nil {
		err := r.userManageProvider.Manage(uint(userId), ur)
		if err != nil {
			c.JSON(http.StatusInternalServerError, router.ErrorResponse{Error: err.Error()})
			return
		}
		c.Status(http.StatusOK)
		return
	}
	c.AbortWithStatusJSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
}
