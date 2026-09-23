package authmiddleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sbondCo/Watcharr/config"
	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/feature/auth/permission"
	"gorm.io/gorm"
)

// Auth middleware
// If db is passed, extra user info from the database will be fetched.
//
// **NOTE:** Instead of providing the `db` parameter, it is probably better to
// fetch what you need in the handler directly! We might follow that pattern
// from now on and potentially remove `db` from this func in the future.
func AuthRequired(db *gorm.DB, cfg *config.ServerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !Authenticate(c, db, cfg) {
			c.AbortWithStatus(401)
			return
		}
		c.Next()
	}
}

// Authenticate validates the request's token and sets the user's details
// (userId, userType and, if db is passed, username, userPermissions and
// userCountry) on the context. It does not abort or call the next handler,
// callers decide what to do when it returns false.
func Authenticate(c *gin.Context, db *gorm.DB, cfg *config.ServerConfig) bool {
	slog.Debug("Authenticate hit")
	atoken := c.GetHeader("Authorization")
	// Make sure auth header isn't empty
	if atoken == "" {
		slog.Debug("Authenticate: Authorization header not provided")
		return false
	}
	// Parse token, only accepting the HMAC method we sign with.
	token, err := jwt.ParseWithClaims(atoken, &entity.TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.JWT_SECRET), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		slog.Error("Authenticate failed to parse token", "error", err)
		return false
	}
	claims, ok := token.Claims.(*entity.TokenClaims)
	if !ok || !token.Valid {
		slog.Error("Token is **not** valid")
		return false
	}
	// Check if token issuedAt is from before `timeOfNewLoginRequired`.
	// Basically just so we can logout old tokens and force relogin...
	// since new changes require the user login again.
	timeOfNewLoginRequired, _ := time.Parse(time.RFC822, "18 Aug 23 20:30 UTC")
	if claims.IssuedAt == nil || claims.IssuedAt.Before(timeOfNewLoginRequired) {
		slog.Info("Token is from before timeOfNewLoginRequired.. returning 401", "token_issued_at", claims.IssuedAt, "time_of_new_login_required", timeOfNewLoginRequired)
		return false
	}
	slog.Debug("Token is valid", "claims", claims)
	c.Set("userId", claims.UserID)
	c.Set("userType", claims.Type)
	// If db passed, get extra user info and set as variables in req context
	if db != nil {
		slog.Debug("Authenticate: db passed.. getting extra user info")
		dbUser := new(entity.User)
		res := db.Where("id = ?", claims.UserID).Take(&dbUser)
		if res.Error != nil {
			slog.Error("Authenticate: Failed to select user from database", "error", res.Error)
			return false
		}
		c.Set("username", dbUser.Username)
		c.Set("userPermissions", dbUser.Permissions)
		if dbUser.Country != nil {
			c.Set("userCountry", *dbUser.Country)
		}
	}
	return true
}

// IsAdmin reports if the user set on the context by Authenticate (with db)
// has admin permissions.
func IsAdmin(c *gin.Context) bool {
	return permission.Has(c.GetInt("userPermissions"), entity.PERM_ADMIN)
}

// Admin only middleware (use after AuthRequired with extra info!)
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.GetUint("userId")
		perms := c.GetInt("userPermissions")
		if permission.Has(perms, entity.PERM_ADMIN) {
			slog.Debug("AdminRequired: User has permission to access admin only route", "user_id", userId)
			c.Next()
			return
		}
		slog.Info("AdminRequired: User denied permission to access admin only route", "user_id", userId)
		c.AbortWithStatus(401)
	}
}

// Specific perm only middleware (use after AuthRequired with extra info!)
func PermRequired(perm int) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.GetUint("userId")
		perms := c.GetInt("userPermissions")
		if permission.Has(perms, perm) {
			slog.Debug("PermRequired: User has permission to access perm only route", "user_id", userId, "required_perm", perm)
			c.Next()
			return
		}
		slog.Info("PermRequired: User denied permission to access perm only route", "user_id", userId, "required_perm", perm)
		c.AbortWithStatus(401)
	}
}
