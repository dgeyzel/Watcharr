package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sbondCo/Watcharr/config"
	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/feature/auth/permission"
	"golang.org/x/crypto/argon2"
	"gorm.io/gorm"
)

// MinPasswordLength is the minimum length of the admin password. The whole
// site is protected by this one password, so it must be strong.
const MinPasswordLength = 12

var ErrPasswordTooShort = fmt.Errorf("password must be at least %d characters", MinPasswordLength)

// We use a separate struct for registration to avoid confusion
// and possible accidents where we allow a user to pass in a
// property from the main User struct that shouldn't be allowed.
type UserRegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

type UserPasswordUpdateRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

type AvailableAuthProvidersResponse struct {
	IsInSetup bool `json:"isInSetup"`
}

type Service struct {
	db  *gorm.DB
	cfg *config.ServerConfig
}

func NewService(db *gorm.DB, cfg *config.ServerConfig) *Service {
	return &Service{
		db,
		cfg,
	}
}

func validatePassword(pw string) error {
	if utf8.RuneCountInString(pw) < MinPasswordLength {
		return ErrPasswordTooShort
	}
	return nil
}

// RegisterFirstUser creates the one and only (admin) account. It fails if any
// user already exists.
func (s *Service) RegisterFirstUser(ur *UserRegisterRequest) (AuthResponse, error) {
	if err := validatePassword(ur.Password); err != nil {
		return AuthResponse{}, err
	}
	// Ensure no users exist
	var userCount int64
	uresp := s.db.Model(&entity.User{}).Count(&userCount)
	if uresp.Error != nil {
		slog.Error("registerFirstUser: User count query failed!", "error", uresp.Error)
		return AuthResponse{}, errors.New("failed to query db for a count of users")
	}
	if userCount != 0 {
		slog.Warn("registerFirstUser: registered users already exist.")
		return AuthResponse{}, errors.New("first user already registered")
	}
	slog.Info("Registering first (admin) user.", "username", ur.Username)
	hash, err := s.hashPassword(ur.Password, entity.GetPassArgonParams())
	if err != nil {
		slog.Error("registerFirstUser: Failed to hash password", "error", err)
		return AuthResponse{}, errors.New("failed to hash password")
	}
	user := entity.User{
		Username:    ur.Username,
		Password:    hash,
		Permissions: entity.PERM_ADMIN,
		UserSettings: entity.UserSettings{
			Country: &s.cfg.DEFAULT_COUNTRY,
		},
	}
	if res := s.db.Create(&user); res.Error != nil {
		slog.Error("registerFirstUser: Creating user failed", "error", res.Error)
		return AuthResponse{}, errors.New("failed to create user")
	}
	token, err := s.signJWT(&user)
	if err != nil {
		slog.Error("registerFirstUser: Failed to sign new jwt", "error", err)
		return AuthResponse{}, errors.New("failed to get auth token")
	}
	return AuthResponse{Token: token}, nil
}

// Login the admin. Only users with PERM_ADMIN can login.
func (s *Service) Login(userL *entity.User) (AuthResponse, error) {
	slog.Debug("A User Is Logging In", "username", userL.Username)
	dbUser := new(entity.User)
	res := s.db.Where("username = ? AND (type IS NULL OR type = 0)", userL.Username).Take(&dbUser)
	if res.Error != nil {
		slog.Warn("Login: user not found", "username", userL.Username)
		return AuthResponse{}, errors.New("incorrect details")
	}

	match, err := s.compareHash(userL.Password, dbUser.Password)
	if err != nil {
		slog.Error("Failed to compare pass to hash for login", "error", err)
		return AuthResponse{}, errors.New("incorrect details")
	}
	if !match {
		slog.Warn("Login: incorrect password", "username", userL.Username)
		return AuthResponse{}, errors.New("incorrect details")
	}
	if !permission.Has(dbUser.Permissions, entity.PERM_ADMIN) {
		slog.Warn("Login: non admin user refused", "username", userL.Username)
		return AuthResponse{}, errors.New("incorrect details")
	}

	token, err := s.signJWT(dbUser)
	if err != nil {
		slog.Error("Failed to sign new jwt", "error", err)
		return AuthResponse{}, errors.New("failed to get auth token")
	}
	return AuthResponse{Token: token}, nil
}

func (s *Service) signJWT(user *entity.User) (token string, err error) {
	// Create new jwt with claim data
	jwt := jwt.NewWithClaims(jwt.SigningMethodHS256, entity.TokenClaims{
		UserID:   user.ID,
		Username: user.Username,
		Type:     user.Type,
		RegisteredClaims: jwt.RegisteredClaims{
			// ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt: jwt.NewNumericDate(time.Now()),
			Issuer:   "watcharr",
		},
	})

	// Sign and get the complete encoded token as a string using the secret
	return jwt.SignedString([]byte(s.cfg.JWT_SECRET))
}

func (s *Service) hashPassword(password string, p *entity.ArgonParams) (encodedHash string, err error) {
	salt, err := s.generateRandomBytes(p.SaltLength)
	if err != nil {
		return "", err
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		p.Iterations,
		p.Memory,
		p.Parallelism,
		p.KeyLength,
	)

	// Base64 encode the salt and hashed password.
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Format hash in standard way.
	encodedHash = fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		p.Memory,
		p.Iterations,
		p.Parallelism,
		b64Salt,
		b64Hash,
	)

	return encodedHash, nil
}

func (s *Service) generateRandomBytes(n uint32) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (s *Service) compareHash(password, encodedHash string) (match bool, err error) {
	// Extract the parameters, salt and derived key from the encoded password
	// hash.
	p, salt, hash, err := s.decodeHash(encodedHash)
	if err != nil {
		return false, err
	}

	// Derive the key from the other password using the same parameters.
	otherHash := argon2.IDKey(
		[]byte(password),
		salt,
		p.Iterations,
		p.Memory,
		p.Parallelism,
		p.KeyLength,
	)

	// Check that the contents of the hashed passwords are identical. Note
	// that we are using the subtle.ConstantTimeCompare() function for this
	// to help prevent timing attacks.
	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		return true, nil
	}
	return false, nil
}

func (s *Service) decodeHash(encodedHash string) (p *entity.ArgonParams, salt, hash []byte, err error) {
	vals := strings.Split(encodedHash, "$")
	if len(vals) != 6 {
		return nil, nil, nil, errors.New("the encoded hash is not in the correct format")
	}

	var version int
	_, err = fmt.Sscanf(vals[2], "v=%d", &version)
	if err != nil {
		return nil, nil, nil, err
	}
	if version != argon2.Version {
		return nil, nil, nil, errors.New("incompatible version of argon2")
	}

	p = &entity.ArgonParams{}
	_, err = fmt.Sscanf(vals[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Iterations, &p.Parallelism)
	if err != nil {
		return nil, nil, nil, err
	}

	salt, err = base64.RawStdEncoding.Strict().DecodeString(vals[4])
	if err != nil {
		return nil, nil, nil, err
	}
	p.SaltLength = uint32(len(salt))

	hash, err = base64.RawStdEncoding.Strict().DecodeString(vals[5])
	if err != nil {
		return nil, nil, nil, err
	}
	p.KeyLength = uint32(len(hash))

	return p, salt, hash, nil
}

func (s *Service) UserChangePassword(pwds UserPasswordUpdateRequest, userId uint) error {
	slog.Debug("userChangePassword request running", "user_id", userId)
	if err := validatePassword(pwds.NewPassword); err != nil {
		return err
	}
	user := new(entity.User)
	res := s.db.Where("id = ?", userId).Select("password").Take(&user)
	if res.Error != nil {
		slog.Error("userChangePassword failed - failed to retrieve user from database", "user_id", userId, "error", res.Error)
		return errors.New("failed to retrieve user")
	}
	slog.Debug("userChangePassword user found", "user_id", userId)
	match, err := s.compareHash(pwds.OldPassword, user.Password)
	if err != nil {
		slog.Error("userChangePassword failed - failed to compare passwords", "user_id", userId, "error", err)
		return errors.New("failed to compare passwords")
	}
	if !match {
		slog.Error("userChangePassword failed - current password hash doesn't match password hash in database", "user_id", userId, "error", err)
		return errors.New("current password provided doesn't match password in database")
	}
	slog.Debug("userChangePassword hash for current password matches hash in the database", "user_id", userId)
	slog.Debug("userChangePassword hashing new password", "user_id", userId)
	hash, err := s.hashPassword(pwds.NewPassword, entity.GetPassArgonParams())
	if err != nil {
		slog.Error("userChangePassword failed - failed to hash new password", "user_id", userId, "error", err)
		return errors.New("failed to hash new password")
	}
	slog.Debug("userChangePassword new password hashed", "user_id", userId)
	if err := s.db.Model(&entity.User{}).Where("id = ?", userId).Update("password", hash).Error; err != nil {
		slog.Error("userChangePassword failed - failed to update password in database", "user_id", userId, "error", err)
		return errors.New("failed to update password")
	} else {
		slog.Debug("userChangePassword password updated", "user_id", userId)
	}
	return nil
}

var (
	ErrNoAdmin        = errors.New("there is no admin account yet, create one at /setup")
	ErrAdminNotFound  = errors.New("no admin account with that username")
	ErrAdminAmbiguous = errors.New("more than one admin account exists, pass the username")
)

// ResetAdminPassword sets a new password for the admin, for when it has been
// lost (run from the command line, not the api). With an empty username the
// only admin account is used. Returns the username that was updated.
func (s *Service) ResetAdminPassword(username string, newPassword string) (string, error) {
	if err := validatePassword(newPassword); err != nil {
		return "", err
	}
	var users []entity.User
	if err := s.db.Where("type IS NULL OR type = 0").Select("id", "username", "permissions").Find(&users).Error; err != nil {
		return "", fmt.Errorf("failed to read users: %w", err)
	}
	admins := []entity.User{}
	for _, u := range users {
		if permission.Has(u.Permissions, entity.PERM_ADMIN) {
			admins = append(admins, u)
		}
	}
	var admin *entity.User
	switch {
	case len(admins) == 0:
		return "", ErrNoAdmin
	case username != "":
		for i := range admins {
			if admins[i].Username == username {
				admin = &admins[i]
			}
		}
		if admin == nil {
			return "", ErrAdminNotFound
		}
	case len(admins) > 1:
		return "", ErrAdminAmbiguous
	default:
		admin = &admins[0]
	}
	hash, err := s.hashPassword(newPassword, entity.GetPassArgonParams())
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	if err := s.db.Model(&entity.User{}).Where("id = ?", admin.ID).Update("password", hash).Error; err != nil {
		return "", fmt.Errorf("failed to save password: %w", err)
	}
	slog.Warn("Admin password was reset from the command line", "username", admin.Username)
	return admin.Username, nil
}
