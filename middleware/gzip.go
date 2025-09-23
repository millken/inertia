package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/millken/inertia"
)

var gzPool = sync.Pool{
	New: func() interface{} {
		w := gzip.NewWriter(io.Discard)
		return w
	},
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w *gzipResponseWriter) WriteHeader(status int) {
	w.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(status)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func Gzip() inertia.HandlerFunc {
	return func(c *inertia.Context) {
		if c.Request == nil || !strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		// set header first
		c.Writer.Header().Set("Content-Encoding", "gzip")

		gz := gzPool.Get().(*gzip.Writer)
		gz.Reset(c.Writer)
		defer func() {
			gz.Close()
			gzPool.Put(gz)
		}()

		orig := c.Writer
		c.Writer = &gzipResponseWriter{Writer: gz, ResponseWriter: orig}
		c.Next()
		// restore original writer (optional)
		c.Writer = orig
	}
}
