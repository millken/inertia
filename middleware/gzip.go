package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/millken/inertia"
)

var DefaultExcludedExtentions = []string{
	".png", ".gif", ".jpeg", ".jpg", ".js", ".css", ".woff", ".woff2", ".ttf", ".eot", ".svg", ".mp4", ".mp3", ".avi", ".mov", ".mkv", ".zip", ".rar", ".7z", ".gz", ".tar",
}

const (
	headerAcceptEncoding  = "Accept-Encoding"
	headerContentEncoding = "Content-Encoding"
	headerVary            = "Vary"
)

type gzipResponseWriter struct {
	inertia.ResponseWriter
	writer *gzip.Writer
}

func (g *gzipResponseWriter) WriteHeader(code int) {
	g.Header().Del("Content-Length")
	g.ResponseWriter.WriteHeader(code)
}

func (g *gzipResponseWriter) Write(data []byte) (int, error) {
	// Ensure Content-Length is not sent by underlying writer
	g.Header().Del("Content-Length")
	return g.writer.Write(data)
}

func (g *gzipResponseWriter) Flush() {
	// Flush gzip writer then underlying flusher if exists
	g.writer.Flush()
	if fl, ok := g.ResponseWriter.(http.Flusher); ok {
		fl.Flush()
	}
}

type GzipOption func(*gzipOptions)
type gzipOptions struct {
	Level                  int
	ExcludedExtentions     []string
	customShouldCompressFn func(req *http.Request) bool
}

func WithGzipLevel(level int) GzipOption {
	return func(opts *gzipOptions) {
		opts.Level = level
	}
}

func WithGzipShouldCompressFn(fn func(req *http.Request) bool) GzipOption {
	return func(opts *gzipOptions) {
		opts.customShouldCompressFn = fn
	}
}

func WithGzipExcludedExtentions(exts []string) GzipOption {
	return func(opts *gzipOptions) {
		opts.ExcludedExtentions = exts
	}
}

func Gzip(options ...GzipOption) inertia.HandlerFunc {
	// set default options
	opts := &gzipOptions{
		Level:                  gzip.DefaultCompression,
		ExcludedExtentions:     DefaultExcludedExtentions,
		customShouldCompressFn: func(req *http.Request) bool { return false },
	}
	for _, option := range options {
		option(opts)
	}
	if !isCompressionLevelValid(opts.Level) {
		// For web content, level 4 seems to be a sweet spot.
		opts.Level = 4
	}
	var gzipPool = sync.Pool{
		New: func() interface{} {
			gz, _ := gzip.NewWriterLevel(io.Discard, opts.Level)
			return gz
		},
	}
	var shouldCompress = func(req *http.Request) bool {
		if !strings.Contains(req.Header.Get(headerAcceptEncoding), "gzip") ||
			strings.Contains(req.Header.Get("Connection"), "Upgrade") {
			return false
		}

		// Check if the request path is excluded from compression
		extension := filepath.Ext(req.URL.Path)
		return opts.customShouldCompressFn(req) || slices.Contains(opts.ExcludedExtentions, extension)
	}
	return func(c *inertia.Context) {
		if c.Request == nil || !shouldCompress(c.Request) {
			c.Next()
			return
		}

		// set headers
		c.Writer.Header().Set("Content-Encoding", "gzip")
		c.Writer.Header().Add("Vary", "Accept-Encoding")

		gz := gzipPool.Get().(*gzip.Writer)
		gz.Reset(c.Writer)
		originalEtag := c.GetHeader("ETag")
		if originalEtag != "" && !strings.HasPrefix(originalEtag, "W/") {
			c.Header("ETag", "W/"+originalEtag)
		}
		c.Writer = &gzipResponseWriter{c.Writer, gz}
		defer func() {
			if c.Writer.Size() < 0 {
				// do not write gzip footer when nothing is written to the response body
				gz.Reset(io.Discard)
			}
			_ = gz.Close()
			if c.Writer.Size() > -1 {
				c.Header("Content-Length", strconv.Itoa(c.Writer.Size()))
			}
			gzipPool.Put(gz)
		}()
		c.Next()
	}
}

func isCompressionLevelValid(level int) bool {
	return level == gzip.DefaultCompression ||
		level == gzip.NoCompression ||
		(level >= gzip.BestSpeed && level <= gzip.BestCompression)
}
