package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/sbondCo/Watcharr/config"
	"github.com/sbondCo/Watcharr/database"
	"github.com/sbondCo/Watcharr/feature/auth"
	"github.com/sbondCo/Watcharr/logging"
	"golang.org/x/term"
	"gorm.io/gorm"
)

const resetPasswordUsage = `Usage: watcharr reset-password [-username NAME]

Sets a new password for the admin account, for when it has been lost.
The new password is asked for twice, or read from the first line of
standard input when it isn't a terminal:

  echo 'new password here' | watcharr reset-password

Run it with the same data directory (WATCHARR_DATA) as the server.
`

// openDataDir opens the server's existing database and config, without
// creating either.
func openDataDir() (*gorm.DB, *config.ServerConfig, error) {
	dbPath := path.Join(config.DataPath, "watcharr.db")
	if _, err := os.Stat(dbPath); err != nil {
		return nil, nil, fmt.Errorf("no database at %s (set WATCHARR_DATA to the server's data directory)", dbPath)
	}
	// Record the reset in the server log, keep the terminal for the user.
	logging.SetupFileOnly(path.Join(config.DataPath, "watcharr.log"))
	cfg, err := config.Get()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read config: %w", err)
	}
	db, err := database.New()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}
	return db, cfg, nil
}

// readNewPassword prompts twice without echo on a terminal, otherwise reads
// the first line of in.
func readNewPassword(in io.Reader, out io.Writer) (string, error) {
	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		fmt.Fprint(out, "New password: ")
		first, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(out)
		if err != nil {
			return "", err
		}
		fmt.Fprint(out, "Repeat new password: ")
		second, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(out)
		if err != nil {
			return "", err
		}
		if string(first) != string(second) {
			return "", errors.New("the passwords don't match")
		}
		return string(first), nil
	}
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// resetPasswordCmd runs `watcharr reset-password`, returning the exit code.
func resetPasswordCmd(
	args []string,
	in io.Reader,
	out io.Writer,
	open func() (*gorm.DB, *config.ServerConfig, error),
) int {
	fs := flag.NewFlagSet("reset-password", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.Usage = func() { fmt.Fprint(out, resetPasswordUsage) }
	username := fs.String("username", "", "admin username (only needed if there is more than one admin)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fs.Usage()
		return 2
	}

	pw, err := readNewPassword(in, out)
	if err != nil {
		fmt.Fprintln(out, "Error:", err)
		return 1
	}
	db, cfg, err := open()
	if err != nil {
		fmt.Fprintln(out, "Error:", err)
		return 1
	}
	name, err := auth.NewService(db, cfg).ResetAdminPassword(*username, pw)
	if err != nil {
		fmt.Fprintln(out, "Error:", err)
		return 1
	}
	fmt.Fprintf(out, "The password for %q was reset. Sign in at /admin with the new password.\n", name)
	fmt.Fprintln(out, "Browsers that are already signed in stay signed in. To sign them all out, change JWT_SECRET in watcharr.json and restart the server.")
	return 0
}
