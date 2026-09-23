package setup

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sbondCo/Watcharr/feature/auth"
	"github.com/sbondCo/Watcharr/feature/setup/setupglob"
	"github.com/sbondCo/Watcharr/router"
)

type AuthProvider interface {
	RegisterFirstUser(urr *auth.UserRegisterRequest) (auth.AuthResponse, error)
}

type Router struct {
	br           *router.BaseRouter
	authProvider AuthProvider
}

func NewRouter(br *router.BaseRouter, authProvider AuthProvider) *Router {
	return &Router{
		br,
		authProvider,
	}
}

type CreateAdminRequest struct {
	auth.UserRegisterRequest
	// Token printed to the server log at startup.
	SetupToken string `json:"setupToken" binding:"required"`
}

// Since we cannot remove these setup routes after they are registered,
// each route/service should ensure we are still in setup before continuing.
// After server restart, these routes shouldn't exist if setup finished
// (currently it is finished if a user is created).
//
// Each controller can check ServerInSetup var first, then each service
// can double check what it needs to (eg create_admin service, registerFirstUser,
// will check that no users exist).
func (r *Router) AddRoutes() {
	setup := r.br.Router.Group("/setup")

	// Server setup routes are being added, so we are in setup now.
	setupglob.ServerInSetup = true

	token, err := setupglob.NewSetupToken()
	if err != nil {
		slog.Error("Failed to generate setup token, setup will not be possible", "error", err)
	} else {
		slog.Warn("Server is in setup. Use this setup token to create the admin account.", "setup_token", token)
		fmt.Printf("\n  ==> Setup token (needed to create the admin account): %s\n\n", token)
	}

	setup.POST("/create_admin", router.NewRateLimiter(12*time.Second, 5).Middleware(), r.CreateAdmin)
}

// Create first user (which will be an admin).
func (r *Router) CreateAdmin(c *gin.Context) {
	if !setupglob.ServerInSetup {
		c.JSON(http.StatusForbidden, router.ErrorResponse{Error: "not in setup"})
		return
	}
	var req CreateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "username, password and setup token are required"})
		return
	}
	if !setupglob.ValidSetupToken(req.SetupToken) {
		slog.Warn("CreateAdmin: invalid setup token provided", "ip", c.ClientIP())
		c.JSON(http.StatusForbidden, router.ErrorResponse{Error: "invalid setup token"})
		return
	}
	response, err := r.authProvider.RegisterFirstUser(&req.UserRegisterRequest)
	if err != nil {
		c.JSON(http.StatusForbidden, router.ErrorResponse{Error: err.Error()})
		return
	}
	// Setup is finished after the first user registered successfully.
	setupglob.ServerInSetup = false
	setupglob.SetupToken = ""
	c.JSON(http.StatusOK, response)
}
