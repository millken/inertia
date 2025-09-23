package middleware

import (
	"cmp"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/millken/inertia"
)

type AccessLogOption func(*accessLogOptions)
type accessLogOptions struct {
	// Out is the output writer. If nil, defaults to os.Stdout or os.Stderr.
	Out io.Writer
}

func WithAccessLogOutput(out io.Writer) AccessLogOption {
	return func(opts *accessLogOptions) {
		opts.Out = out
	}
}

// AccessLog returns an inertia middleware that logs the request method, URL and duration.
// If out is nil the middleware uses the default logger (plain text, or JSON in production).
func AccessLog(options ...AccessLogOption) inertia.HandlerFunc {
	var opts = accessLogOptions{
		Out: os.Stdout,
	}
	for _, option := range options {
		option(&opts)
	}
	logger := slog.Default()
	out := opts.Out
	// if out is os.Stdout or os.Stderr use terminal handler
	if f, ok := out.(*os.File); ok {
		if f.Fd() == os.Stdout.Fd() || f.Fd() == os.Stderr.Fd() {
			// use terminal handler for stdout and stderr
			logger = slog.New(slog.NewTextHandler(out, nil))
		}
	} else if out != nil {
		// use JSON handler if out is provided
		logger = slog.New(slog.NewJSONHandler(out, nil))
	}
	return func(c *inertia.Context) {
		start := time.Now()

		var status int
		defer func() {
			// try to get status from writer if it exposes Status() int
			if sw, ok := c.Writer.(interface{ Status() int }); ok {
				status = cmp.Or(sw.Status(), http.StatusOK)
			} else {
				status = http.StatusOK
			}

			logLevel := slog.LevelInfo
			if status >= http.StatusInternalServerError {
				logLevel = slog.LevelError
			}

			logger.Log(c.Request.Context(), logLevel, "", "method", c.Request.Method, "status", status, "url", c.Request.URL.Path, "took", time.Since(start))
		}()

		c.Next()
	}
}
