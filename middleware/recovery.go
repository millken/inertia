package middleware

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

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
				const stackSkip = 3
				stack := stack(stackSkip)
				slog.Error("panic", "error", err, "method", c.Request.Method, "url", c.Request.URL.Path, "stack", stack)

				if config.EnableStackTrace {
					if config.OutputWriter != nil {
						fmt.Fprintf(config.OutputWriter, "%v\n", err)
						config.OutputWriter.Write(stack)
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

// stack returns a nicely formatted stack frame, skipping skip frames.
func stack(skip int) []byte {
	buf := new(bytes.Buffer) // the returned data
	// As we loop, we open files and read them. These variables record the currently
	// loaded file.
	var lines [][]byte
	var lastFile string
	for i := skip; ; i++ { // Skip the expected number of frames
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		// Print this much at least.  If we can't find the source, it won't show.
		fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
		if file != lastFile {
			data, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			lines = bytes.Split(data, []byte{'\n'})
			lastFile = file
		}
		fmt.Fprintf(buf, "\t%s: %s\n", function(pc), source(lines, line))
	}
	return buf.Bytes()
}

const dunno = "???"

var dunnoBytes = []byte(dunno)

// source returns a space-trimmed slice of the n'th line.
func source(lines [][]byte, n int) []byte {
	n-- // in stack trace, lines are 1-indexed but our array is 0-indexed
	if n < 0 || n >= len(lines) {
		return dunnoBytes
	}
	return bytes.TrimSpace(lines[n])
}

// function returns, if possible, the name of the function containing the PC.
func function(pc uintptr) string {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return dunno
	}
	name := fn.Name()
	// The name includes the path name to the package, which is unnecessary
	// since the file name is already included.  Plus, it has center dots.
	// That is, we see
	//	runtime/debug.*T·ptrmethod
	// and want
	//	*T.ptrmethod
	// Also the package path might contain dot (e.g. code.google.com/...),
	// so first eliminate the path prefix
	if lastSlash := strings.LastIndexByte(name, '/'); lastSlash >= 0 {
		name = name[lastSlash+1:]
	}
	if period := strings.IndexByte(name, '.'); period >= 0 {
		name = name[period+1:]
	}
	name = strings.ReplaceAll(name, "·", ".")
	return name
}

// timeFormat returns a customized time string for logger.
func timeFormat(t time.Time) string {
	return t.Format("2006/01/02 - 15:04:05")
}
