package main

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/sbondCo/Watcharr/config"
	"github.com/sbondCo/Watcharr/internal/testutil/testserver"
	"gorm.io/gorm"
)

func TestResetPasswordCommand(t *testing.T) {
	s := testserver.New(t)
	s.SeedAdmin()
	open := func() (*gorm.DB, *config.ServerConfig, error) { return s.DB, s.Cfg, nil }
	run := func(stdin string, args ...string) (int, string) {
		var out strings.Builder
		code := resetPasswordCmd(args, strings.NewReader(stdin), &out, open)
		return code, out.String()
	}
	login := func(pw string) int {
		return s.Do(http.MethodPost, "/api/auth/", map[string]string{"username": testserver.AdminUsername, "password": pw}, "").Code
	}

	if code, out := run("short\n"); code != 1 || !strings.Contains(out, "at least 12") {
		t.Fatalf("short password: %d %q", code, out)
	}
	if code, out := run("a-long-enough-password\n", "-username", "nobody"); code != 1 || !strings.Contains(out, "no admin account with that username") {
		t.Fatalf("unknown user: %d %q", code, out)
	}
	if code, _ := run("", "extra-arg"); code != 2 {
		t.Fatalf("stray argument: %d", code)
	}

	// Windows line endings from a piped file are trimmed.
	code, out := run("the-new-admin-password\r\n")
	if code != 0 || !strings.Contains(out, "was reset") || !strings.Contains(out, "JWT_SECRET") {
		t.Fatalf("reset: %d %q", code, out)
	}
	if c := login(testserver.AdminPassword); c == http.StatusOK {
		t.Fatal("old password still works")
	}
	if c := login("the-new-admin-password"); c != http.StatusOK {
		t.Fatalf("new password login: %d", c)
	}
}

func TestResetPasswordCommandReportsOpenErrors(t *testing.T) {
	var out strings.Builder
	open := func() (*gorm.DB, *config.ServerConfig, error) {
		return nil, nil, errors.New("no database at /nowhere/watcharr.db")
	}
	if code := resetPasswordCmd(nil, strings.NewReader("a-long-enough-password\n"), &out, open); code != 1 || !strings.Contains(out.String(), "no database") {
		t.Fatalf("%d %q", code, out.String())
	}
}
