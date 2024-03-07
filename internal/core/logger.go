package core

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/viper"
)

var logger *slog.Logger

func InitializeLogger() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	switch {
	// We don't support Windows
	case isatty.IsCygwinTerminal(os.Stdout.Fd()):
		panic("Cygwin/MSYS2 Terminal is unsupported")

	// We log to stdout if forced, or if we don't have a terminal
	// (pipes, containers, service managers)
	case viper.GetBool("log.force_stdout"), !isatty.IsTerminal(os.Stdout.Fd()):
		logger = slog.New(slog.NewJSONHandler(os.Stdout, opts)).
			With("environment", "development").
			With("server", fmt.Sprintf("Corto/%s", Version))

	// We have a tty, log to file when possible
	default:
		logfile, err := os.OpenFile(
			"log/corto.log",
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			0o644,
		)
		if err != nil {
			// No log directory here; fall back to stdout
			logger = slog.New(slog.NewJSONHandler(os.Stdout, opts)).
				With("environment", "development").
				With("server", fmt.Sprintf("Corto/%s", Version))
			break
		}

		logger = slog.New(slog.NewJSONHandler(logfile, opts)).
			With("environment", "development").
			With("server", fmt.Sprintf("Corto/%s", Version))
	}

	slog.SetDefault(logger)
	return logger
}
