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

func TestResetAdminPassword(t *testing.T) {
	s := newService(t)
	if _, err := s.ResetAdminPassword("", goodPassword); !errors.Is(err, ErrNoAdmin) {
		t.Fatalf("no admin yet: %v", err)
	}
	if _, err := s.RegisterFirstUser(&UserRegisterRequest{Username: "owner", Password: goodPassword}); err != nil {
		t.Fatal(err)
	}
	// A leftover non admin user is never picked.
	s.db.Create(&entity.User{Username: "guest", Password: "x", Permissions: entity.PERM_NONE})

	if _, err := s.ResetAdminPassword("", "short"); !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("short password: %v", err)
	}
	if _, err := s.ResetAdminPassword("guest", "another-long-password"); !errors.Is(err, ErrAdminNotFound) {
		t.Fatalf("non admin username: %v", err)
	}

	const newPassword = "a-brand-new-password"
	name, err := s.ResetAdminPassword("", newPassword)
	if err != nil || name != "owner" {
		t.Fatalf("reset: %q %v", name, err)
	}
	if _, err := s.Login(&entity.User{Username: "owner", Password: goodPassword}); err == nil {
		t.Fatal("old password still works")
	}
	if _, err := s.Login(&entity.User{Username: "owner", Password: newPassword}); err != nil {
		t.Fatalf("new password: %v", err)
	}
	var guest entity.User
	s.db.Where("username = ?", "guest").Take(&guest)
	if guest.Password != "x" {
		t.Fatal("non admin password changed")
	}

	// A second admin (should never happen) needs the username.
	hash, _ := s.hashPassword(goodPassword, entity.GetPassArgonParams())
	s.db.Create(&entity.User{Username: "owner2", Password: hash, Permissions: entity.PERM_ADMIN})
	if _, err := s.ResetAdminPassword("", newPassword); !errors.Is(err, ErrAdminAmbiguous) {
		t.Fatalf("two admins: %v", err)
	}
	if name, err := s.ResetAdminPassword("owner2", newPassword); err != nil || name != "owner2" {
		t.Fatalf("named reset: %q %v", name, err)
	}
}
