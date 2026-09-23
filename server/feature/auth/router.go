package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/feature/auth/authmiddleware"
	"github.com/sbondCo/Watcharr/feature/setup/setupglob"
	"github.com/sbondCo/Watcharr/router"
)

type Router struct {
	br      *router.BaseRouter
	service *Service
	// Limits login attempts per client ip.
	loginLimiter *router.RateLimiter
}

func NewRouter(br *router.BaseRouter, service *Service) *Router {
	return &Router{
		br:           br,
		service:      service,
		loginLimiter: router.NewRateLimiter(12*time.Second, 5),
	}
}

func (r *Router) AddRoutes() {
	auth := r.br.Router.Group("/auth")

	// Login (admin only). 5 attempts, then 5 per minute, per client ip.
	auth.POST("/", r.loginLimiter.Middleware(), r.Login)
	// Tells the ui if the server is in setup.
	auth.GET("/available", r.GetAvailableAuthProviders)

	// IMPORTANT: Routes below here must be authenticated.
	auth.Use(authmiddleware.AuthRequired(nil, r.br.Cfg))
	{
		// Change password
		auth.POST("/change_password", r.UpdateUserPassword)
	}
}

// Login
func (r *Router) Login(c *gin.Context) {
	var user entity.User
	if c.ShouldBindJSON(&user) == nil {
		response, err := r.service.Login(&user)
		if err != nil {
			slog.Warn("Failed login attempt", "ip", c.ClientIP(), "username", user.Username)
			c.JSON(http.StatusForbidden, router.ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusOK, response)
		return
	}
	c.Status(400)
}

// Get available auth providers (only reports setup state now, the admin
// password login is the only way to sign in).
func (r *Router) GetAvailableAuthProviders(c *gin.Context) {
	c.JSON(http.StatusOK, &AvailableAuthProvidersResponse{
		IsInSetup: setupglob.ServerInSetup,
	})
}

// Change password
func (r *Router) UpdateUserPassword(c *gin.Context) {
	userId := c.MustGet("userId").(uint)
	var pwds UserPasswordUpdateRequest
	err := c.ShouldBindJSON(&pwds)
	if err == nil {
		err := r.service.UserChangePassword(pwds, userId)
		if err != nil {
			status := http.StatusForbidden
			if errors.Is(err, ErrPasswordTooShort) {
				status = http.StatusBadRequest
			}
			c.JSON(status, router.ErrorResponse{Error: err.Error()})
			return
		}
		c.Status(http.StatusOK)
		return
	}
	c.AbortWithStatusJSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
}
