package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logLevel = new(slog.LevelVar)
)

// Setup slog defaults
func Setup(logfp string) io.Writer {
	return setup(logfp, true)
}

// SetupFileOnly is Setup without also logging to stdout, for command line
// tools whose output is for the user.
func SetupFileOnly(logfp string) {
	setup(logfp, false)
}

func setup(logfp string, stdout bool) io.Writer {
	var multiw io.Writer = &lumberjack.Logger{
		Filename:   logfp,
		MaxSize:    1, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
		Compress:   false,
	}
	if stdout {
		multiw = io.MultiWriter(multiw, os.Stdout)
	}
	slog.SetDefault(slog.New(
		slog.NewTextHandler(multiw, &slog.HandlerOptions{
			Level:     logLevel,
			AddSource: true,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				// `AddSource=true` adds source code location to each log,
				// this replaces the entire file path added with just it's
				// last dirname and filename.
				if a.Key == slog.SourceKey {
					s := a.Value.Any().(*slog.Source)
					s.File = filepath.Base(filepath.Dir(s.File)) +
						"/" + filepath.Base(s.File)
					return slog.Any(a.Key, s)
				}
				return a
			},
		}),
	))
	return multiw
}

// Set loggin level from config
func SetLevel(debug bool) {
	if debug {
		logLevel.Set(slog.LevelDebug)
	} else {
		logLevel.Set(slog.LevelInfo)
	}
	slog.Info("Logging level set", "logging_level", logLevel)
}
