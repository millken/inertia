package middleware

import (
	"cmp"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"

	"github.com/millken/inertia"
)

// RecoveryConfig defines the config for Recovery middleware.
type RecoveryConfig struct {
	// OutputWriter is where to write the error output. Default is os.Stderr.
	OutputWriter io.Writer
	// EnableStackTrace enables printing stack trace in development mode.
	EnableStackTrace bool
	// RecoveryHandler is a custom recovery handler.
	RecoveryHandler func(c *inertia.Context, err any)
}

// DefaultRecoveryConfig returns a default recovery config.
func DefaultRecoveryConfig() RecoveryConfig {
	return RecoveryConfig{
		OutputWriter:     os.Stderr,
		EnableStackTrace: true,
		RecoveryHandler:  nil,
	}
}

// Recovery is a middleware that recovers from panics and logs the error.
// The error stack trace is printed only when the application is in 'development' mode.
func Recovery() inertia.HandlerFunc {
	return RecoveryWithConfig(DefaultRecoveryConfig())
}

// RecoveryWithConfig returns a Recovery middleware with config.
func RecoveryWithConfig(config RecoveryConfig) inertia.HandlerFunc {
	return func(c *inertia.Context) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic", "error", err, "method", c.Request.Method, "url", c.Request.URL.Path, "stack", debug.Stack())

				if config.EnableStackTrace && cmp.Or(os.Getenv("GO_ENV"), "development") == "development" {
					if config.OutputWriter != nil {
						fmt.Fprintf(config.OutputWriter, "%v\n", err)
						config.OutputWriter.Write(debug.Stack())
					}
				}

				// Use custom recovery handler if provided
				if config.RecoveryHandler != nil {
					config.RecoveryHandler(c, err)
					return
				}

				// Default error handling
				if !c.IsAborted() {
					c.AbortWithStatus(http.StatusInternalServerError)
					if errorHandler, exists := inertia.ErrorHandlerMap[http.StatusInternalServerError]; exists {
						errorHandler(c.Writer, c.Request, fmt.Errorf("%s", err))
					}
				}
			}
		}()

		c.Next()
	}
}
