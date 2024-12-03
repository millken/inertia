package server

import (
	"bytes"
	"cmp"
	"compress/gzip"
	"context"
	"embed"
	"fmt"
	"hash/fnv"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

// Middleware is a function that receives a http.Handler and returns a http.Handler
// that can be used to wrap the original handler with some functionality.
type Middleware func(http.Handler) http.Handler

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), "requestID", time.Now().UnixNano()))
		next.ServeHTTP(w, r)
	})
}

// HttpDump is a middleware that dumps the request and response to the console.
func HttpDump(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := []string{"curl", "-i", "-X", r.Method}

		for k, v := range r.Header {
			ck := http.CanonicalHeaderKey(k)
			if ck != "Host" && ck != "User-Agent" {
				for _, vv := range v {
					parts = append(parts, "-H", fmt.Sprintf("'%s: %s'", k, vv))
				}

			}
		}

		if r.Method == "POST" || r.Method == "PATCH" || r.Method == "PUT" {
			b, err := io.ReadAll(r.Body)

			if err != nil {
				fmt.Println("dumphttp :", err)
			}
			if err == nil && len(b) > 0 {
				r.Body = io.NopCloser(bytes.NewBuffer(b))
				parts = append(parts, "-d '", string(b)+"'")
			}
		}

		scheme := "http"
		if r.TLS != nil {
			scheme += "s"
		}

		parts = append(parts, scheme+"://"+r.Host+r.URL.String())
		fmt.Println(strings.Join(parts, " "))
		next.ServeHTTP(w, r)
	})
}

// logger is a middleware that logs the request method and URL
// and the time it took to process the request.
func Logger(next http.Handler) http.Handler {
	logger := slog.Default()
	if os.Getenv("GO_ENV") == "production" {
		// Using json logger in production
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		lw, ok := w.(*responseWriter)
		if !ok {
			lw = &responseWriter{ResponseWriter: w}
		}

		defer func() {
			status := cmp.Or(lw.Status, http.StatusOK)
			logLevel := slog.LevelInfo

			if status >= http.StatusInternalServerError {
				logLevel = slog.LevelError
			}

			logger.Log(r.Context(), logLevel, "", "method", r.Method, "status", status, "url", r.URL.Path, "took", time.Since(start))
		}()

		next.ServeHTTP(lw, r)
	})
}

// Recovery is a middleware that recovers from panics and logs the error.
// The error stack trace is printed only when the application is in 'development' mode.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic", "error", err, "method", r.Method, "url", r.URL.Path, "stack", debug.Stack())

				if cmp.Or(os.Getenv("GO_ENV"), "development") == "development" {
					os.Stderr.WriteString(fmt.Sprint(err, "\n"))
					os.Stderr.Write(debug.Stack())
				}

				w.WriteHeader(http.StatusInternalServerError)
				errorHandlerMap[http.StatusInternalServerError](w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

type fileServer struct {
	root    fs.FS
	isEmbed bool
	mu      sync.Mutex
	etags   map[string]string
}

func newFileServer(root fs.FS) *fileServer {
	_, ok := root.(embed.FS)
	return &fileServer{
		root:    root,
		isEmbed: ok,
		etags:   make(map[string]string),
	}
}

func (fs *fileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" || path[len(path)-1] == '/' {
		path = path + "index.html"
	}

	if fs.isEmbed {
		ifNoneMatch := r.Header.Get("If-None-Match")
		etag, ok := fs.etags[path]
		if ok && ifNoneMatch != "" && ifNoneMatch == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	fi, err := fs.root.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer fi.Close()

	f, err := fi.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// 读取文件内容到内存中
	content, err := io.ReadAll(fi)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	// 生成 ETag
	h := fnv.New64a()
	h.Write(content)
	etag := fmt.Sprintf("W/%x", h.Sum64())
	if fs.isEmbed {
		fs.mu.Lock()
		defer fs.mu.Unlock()
		fs.etags[path] = etag
	} else {
		w.Header().Set("Last-Modified", f.ModTime().UTC().Format(http.TimeFormat))
	}

	// 设置缓存头
	w.Header().Set("Vary", "Accept-Encoding")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Header().Set("Etag", etag)

	// 提供文件内容
	http.ServeContent(w, r, f.Name(), f.ModTime(), bytes.NewReader(content))
}

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

func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Encoding", "gzip")

		gz := gzPool.Get().(*gzip.Writer)
		defer gzPool.Put(gz)

		gz.Reset(w)
		defer gz.Close()

		next.ServeHTTP(&gzipResponseWriter{ResponseWriter: w, Writer: gz}, r)
	})
}

type Content struct {
	key, value any
}

func WithContext(key, value any) Content {
	return Content{key, value}
}

// Context is a middleware that adds a context value to the request context.
func Context(mctx ...Content) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rctx := r.Context()
			for _, ctx := range mctx {
				rctx = context.WithValue(rctx, ctx.key, ctx.value)
			}
			next.ServeHTTP(w, r.WithContext(rctx))
		})
	}
}
