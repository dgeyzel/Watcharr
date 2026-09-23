package app

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sbondCo/Watcharr/config"
	"github.com/sbondCo/Watcharr/feature/auth/authmiddleware"
	"github.com/sbondCo/Watcharr/router"
	"gorm.io/gorm"
)

// PublicRoute is a route that can be reached without an admin token.
type PublicRoute struct {
	Method string
	// Gin route pattern (as returned by c.FullPath()). A trailing `*` makes
	// it a prefix match.
	Path string
}

// PublicRoutes is the explicit allowlist of routes visitors (no token) can
// reach. EVERY other /api route requires a valid admin token.
var PublicRoutes = []PublicRoute{
	// Public read only api.
	{http.MethodGet, "/api/public/*"},
	// Admin login.
	{http.MethodPost, "/api/auth/"},
	// Tells the ui if the server is in setup.
	{http.MethodGet, "/api/auth/available"},
	// First run setup (requires the setup token, only exists while in setup).
	{http.MethodPost, "/api/setup/create_admin"},
	// Poster and avatar images.
	{http.MethodGet, "/api/img/*"},
	{http.MethodHead, "/api/img/*"},
}

// IsPublicRoute reports if method + gin route pattern is on the allowlist.
func IsPublicRoute(method string, fullPath string) bool {
	for _, r := range PublicRoutes {
		if r.Method != method {
			continue
		}
		if prefix, ok := strings.CutSuffix(r.Path, "*"); ok {
			if strings.HasPrefix(fullPath, prefix) {
				return true
			}
		} else if fullPath == r.Path {
			return true
		}
	}
	return false
}

// adminByDefault makes admin the default for the whole /api group: any route
// that isn't on the PublicRoutes allowlist requires a valid admin token. The
// per feature auth middleware still runs after this as a second layer.
func adminByDefault(db *gorm.DB, cfg *config.ServerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if IsPublicRoute(c.Request.Method, c.FullPath()) {
			c.Next()
			return
		}
		if !authmiddleware.Authenticate(c, db, cfg) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, router.ErrorResponse{Error: "unauthorized"})
			return
		}
		if !authmiddleware.IsAdmin(c) {
			slog.Warn("adminByDefault: non admin denied", "user_id", c.GetUint("userId"), "path", c.FullPath())
			c.AbortWithStatusJSON(http.StatusForbidden, router.ErrorResponse{Error: "forbidden"})
			return
		}
		c.Next()
	}
}
