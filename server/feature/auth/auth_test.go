package auth

import (
	"errors"
	"testing"

	"github.com/sbondCo/Watcharr/config"
	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/internal/testutil"
)

const goodPassword = "a-long-enough-password"

func newService(t *testing.T) *Service {
	t.Helper()
	return NewService(testutil.SetupDB(t), &config.ServerConfig{JWT_SECRET: "test", DEFAULT_COUNTRY: "US"})
}

func TestRegisterFirstUserIsAdminAndOnlyOnce(t *testing.T) {
	s := newService(t)
	if _, err := s.RegisterFirstUser(&UserRegisterRequest{Username: "owner", Password: goodPassword}); err != nil {
		t.Fatalf("first user: %v", err)
	}
	var u entity.User
	s.db.Where("username = ?", "owner").Take(&u)
	if u.Permissions != entity.PERM_ADMIN {
		t.Fatalf("expected first user to be admin, got perms %d", u.Permissions)
	}
	if _, err := s.RegisterFirstUser(&UserRegisterRequest{Username: "second", Password: goodPassword}); err == nil {
		t.Fatal("expected second RegisterFirstUser to fail")
	}
}

func TestRegisterFirstUserRequiresLongPassword(t *testing.T) {
	s := newService(t)
	_, err := s.RegisterFirstUser(&UserRegisterRequest{Username: "owner", Password: "short"})
	if !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestLoginOnlyAllowsAdmins(t *testing.T) {
	s := newService(t)
	if _, err := s.RegisterFirstUser(&UserRegisterRequest{Username: "owner", Password: goodPassword}); err != nil {
		t.Fatal(err)
	}
	// A leftover non admin user with a valid password must not be able to login.
	hash, err := s.hashPassword(goodPassword, entity.GetPassArgonParams())
	if err != nil {
		t.Fatal(err)
	}
	s.db.Create(&entity.User{Username: "guest", Password: hash, Permissions: entity.PERM_NONE})

	if _, err := s.Login(&entity.User{Username: "owner", Password: goodPassword}); err != nil {
		t.Fatalf("admin login failed: %v", err)
	}
	if _, err := s.Login(&entity.User{Username: "owner", Password: "wrong-password-here"}); err == nil {
		t.Fatal("expected wrong password to fail")
	}
	if _, err := s.Login(&entity.User{Username: "guest", Password: goodPassword}); err == nil {
		t.Fatal("expected non admin login to fail")
	}
}

func TestChangePasswordRequiresLongPassword(t *testing.T) {
	s := newService(t)
	if _, err := s.RegisterFirstUser(&UserRegisterRequest{Username: "owner", Password: goodPassword}); err != nil {
		t.Fatal(err)
	}
	var u entity.User
	s.db.Where("username = ?", "owner").Take(&u)
	err := s.UserChangePassword(UserPasswordUpdateRequest{OldPassword: goodPassword, NewPassword: "short"}, u.ID)
	if !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}
	if err := s.UserChangePassword(UserPasswordUpdateRequest{OldPassword: goodPassword, NewPassword: "another-long-password"}, u.ID); err != nil {
		t.Fatalf("valid change failed: %v", err)
	}
}
