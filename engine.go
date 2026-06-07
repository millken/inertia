// Package inertia provides a Go web framework for building modern web applications using the Inertia.js approach.
package inertia

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/millken/inertia/router"
	"github.com/millken/inertia/ssr"
)

var (
	defaultRootHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<!--inertia-head-meta-inertia-->
</head>

<body>
  <div id="app"><!--inertia-ssr-content-inertia--></div>
  <script>window.__INERTIA_PAGE_DATA__="<!--inertia-data-page-inertia-->";</script>
  <script type="module" src="/main.js?_bt=<!--inertia-version-inertia-->"></script>
</body>`
)

type Mode byte

const (
	ModeProduction Mode = iota
	ModeDevelopment
	ModeSSR
)

type HandlerFunc func(c *Context)
type HandlerFuncs []HandlerFunc

// Option is an option parameter that modifies Inertia.
type Option func(e *Engine) error

func WithErrorHandler(status int, errorHandlerFn ErrorHandlerFunc) Option {
	return func(e *Engine) error {
		ErrorHandlerMap[status] = errorHandlerFn
		return nil
	}
}

func WithRootHTML(html string) Option {
	return func(e *Engine) error {
		e.rootHTML = html
		return nil
	}
}

func WithDevAddr(addr string) Option {
	return func(e *Engine) error {
		e.devAddr = addr
		return nil
	}
}

func WithTags(startTag, endTag string) Option {
	return func(e *Engine) error {
		e.startTag = startTag
		e.endTag = endTag
		return nil
	}
}

func WithSSR(ssr ssr.VM) Option {
	return func(e *Engine) error {
		e.ssr = ssr
		return nil
	}
}

func WithMode(mode Mode) Option {
	return func(e *Engine) error {
		e.mode = mode
		return nil
	}
}

// WithReadHeaderTimeout sets the amount of time allowed to read request headers.
// Defaults to 10s. A non-positive value disables the timeout (not recommended).
func WithReadHeaderTimeout(d time.Duration) Option {
	return func(e *Engine) error {
		e.readHeaderTimeout = d
		return nil
	}
}

// WithReadTimeout sets the maximum duration for reading the entire request,
// including the body. Defaults to 0 (no timeout). Avoid setting this when
// serving large uploads, websockets or SSE.
func WithReadTimeout(d time.Duration) Option {
	return func(e *Engine) error {
		e.readTimeout = d
		return nil
	}
}

// WithWriteTimeout sets the maximum duration before timing out writes of the
// response. Defaults to 0 (no timeout). Avoid setting this when serving
// streaming responses, websockets or SSE.
func WithWriteTimeout(d time.Duration) Option {
	return func(e *Engine) error {
		e.writeTimeout = d
		return nil
	}
}

// WithIdleTimeout sets the maximum amount of time to wait for the next request
// when keep-alives are enabled. Defaults to 120s.
func WithIdleTimeout(d time.Duration) Option {
	return func(e *Engine) error {
		e.idleTimeout = d
		return nil
	}
}

// WithShutdownTimeout sets how long Serve waits for in-flight requests to finish
// during graceful shutdown before giving up. Defaults to 10s.
func WithShutdownTimeout(d time.Duration) Option {
	return func(e *Engine) error {
		e.shutdownTimeout = d
		return nil
	}
}

// WithContentSecurityPolicy sets the Content-Security-Policy header emitted on
// HTML documents rendered by Context.Render. It is empty (unset) by default.
// Note: the default root template includes an inline bootstrap <script>, so a
// strict policy such as "script-src 'self'" will block it unless you also allow
// it (e.g. with a nonce or hash).
func WithContentSecurityPolicy(csp string) Option {
	return func(e *Engine) error {
		e.csp = csp
		return nil
	}
}

// WithTrustProxyHeaders controls whether Context.ClientIP honors the
// X-Forwarded-For / X-Real-IP request headers. Defaults to true. Set it to
// false when the server faces untrusted clients directly so that ClientIP
// always reports the real RemoteAddr and cannot be spoofed.
func WithTrustProxyHeaders(trust bool) Option {
	return func(e *Engine) error {
		e.trustProxyHeaders = trust
		return nil
	}
}

// Engine is the main Inertia instance that holds the router and middleware.

type Engine struct {
	bootTime           int64
	mode               Mode
	devAddr            string
	devHTTPClient      *http.Client
	devProxyOnce       sync.Once
	devProxy           *httputil.ReverseProxy
	devProxyErr        error
	MaxMultipartMemory int64
	rootHTML           string
	startTag, endTag   string
	ssr                ssr.VM
	addr               string
	router             *router.Router[HandlerFuncs]
	middleware         HandlerFuncs

	readHeaderTimeout time.Duration
	readTimeout       time.Duration
	writeTimeout      time.Duration
	idleTimeout       time.Duration
	shutdownTimeout   time.Duration

	// trustProxyHeaders controls whether ClientIP honors X-Forwarded-For /
	// X-Real-IP. Defaults to true. Disable it when the server is exposed
	// directly to untrusted clients to prevent IP spoofing.
	trustProxyHeaders bool

	// csp is the Content-Security-Policy applied to HTML rendered by Render.
	// Empty by default: CSP is app-specific and a strict policy must account
	// for the inline bootstrap script (e.g. via a nonce), so opt in explicitly.
	csp string

	serverMu sync.Mutex
	server   *http.Server
}

func New(options ...Option) (*Engine, error) {
	var err error
	e := &Engine{
		bootTime:           time.Now().Unix(),
		mode:               ModeProduction,
		devAddr:            "http://localhost:5173",
		devHTTPClient:      &http.Client{Timeout: 2 * time.Second},
		addr:               ":5000",
		rootHTML:           defaultRootHTML,
		startTag:           "<!--inertia-", //注释标记可以防止被前端框架（如 Vue、React）误删
		endTag:             "-inertia-->",
		MaxMultipartMemory: 32 << 20, // 32 MB
		router:             router.New[HandlerFuncs](),
		// ReadHeaderTimeout guards against Slowloris-style attacks and is safe even
		// for long-lived connections (websocket/SSE) since it only bounds header reads.
		readHeaderTimeout: 10 * time.Second,
		// readTimeout/writeTimeout default to 0 to avoid breaking streaming, large
		// uploads, websocket proxying (dev mode) and SSE. Set them via options if needed.
		idleTimeout:       120 * time.Second,
		shutdownTimeout:   10 * time.Second,
		trustProxyHeaders: true,
	}
	for _, option := range options {
		if err = option(e); err != nil {
			return nil, err
		}
	}
	return e, nil
}

// IsDevelopmentMode returns true if the engine is in development mode.
func (e *Engine) IsDevelopmentMode() bool {
	return e.mode == ModeDevelopment
}

// IsSSRMode returns true if the engine is in SSR mode
func (e *Engine) IsSSRMode() bool {
	return e.mode == ModeSSR
}

func (e *Engine) GET(path string, fn ...HandlerFunc) {
	e.router.Add("GET", path, fn)
}

func (e *Engine) POST(path string, fn ...HandlerFunc) {
	e.router.Add("POST", path, fn)
}

func (e *Engine) PUT(path string, fn ...HandlerFunc) {
	e.router.Add("PUT", path, fn)
}

func (e *Engine) DELETE(path string, fn ...HandlerFunc) {
	e.router.Add("DELETE", path, fn)
}

func (e *Engine) PATCH(path string, fn ...HandlerFunc) {
	e.router.Add("PATCH", path, fn)
}

func (e *Engine) OPTIONS(path string, fn ...HandlerFunc) {
	e.router.Add("OPTIONS", path, fn)
}

func (e *Engine) HEAD(path string, fn ...HandlerFunc) {
	e.router.Add("HEAD", path, fn)
}

func (e *Engine) ANY(path string, fn ...HandlerFunc) {
	for _, method := range []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"} {
		e.router.Add(method, path, fn)
	}
}

func (e *Engine) Addr() string {
	return e.addr
}

func (e *Engine) DevAddr() string {
	return e.devAddr
}

// StaticFS serves static assets from the given path
func (e *Engine) StaticFS(path string, fs fs.FS) {
	if e.IsDevelopmentMode() {
		// in dev mode, we do not serve static assets, they are served by the dev server
		return
	}
	e.GET(path+"*", StaticFileServer(path, fs))
}

// Use allows to specify a middleware that should be executed for all the handlers
// in the group
func (e *Engine) Use(middleware ...HandlerFunc) {
	e.middleware = append(e.middleware, middleware...)
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	found, fn, params := e.router.Lookup(r.Method, r.URL.Path)
	if found {
		if fn == nil {
			ErrorHandlerMap[http.StatusInternalServerError](w, r, fmt.Errorf("handler is nil for %s %s", r.Method, r.URL.Path))
			return
		}
		ctx := acquireContext()
		ctx.writermem.reset(w)
		ctx.reset()
		ctx.Request = r
		ctx.Params = params
		ctx.engine = e

		// Combine middleware and handler into handlers chain
		ctx.handlers = append(ctx.handlers, e.middleware...)
		ctx.handlers = append(ctx.handlers, fn...)

		// Start execution chain
		ctx.Next()

		releaseContext(ctx)
		return
	}
	// 如果是 devMode，未命中路由的请求都转发到开发服务器
	if e.IsDevelopmentMode() {
		e.proxyToDevServer(w, r)
		return
	}
	// not found
	defaultCatchAllHandler.ServeHTTP(w, r)
}

// Serve starts the HTTP server with the configured timeouts and blocks until
// the process receives SIGINT/SIGTERM or ListenAndServe fails. On signal it
// performs a graceful shutdown bounded by the shutdown timeout.
func (e *Engine) Serve() error {
	if e.IsDevelopmentMode() {
		slog.Info("starting server", "mode", "development", "proxy", e.devAddr)
	} else if e.IsSSRMode() {
		slog.Info("starting server", "mode", "ssr", "addr", e.Addr())
	} else {
		slog.Info("starting server", "mode", "production", "addr", e.Addr())
	}

	srv := &http.Server{
		Addr:              e.Addr(),
		Handler:           e,
		ReadHeaderTimeout: e.readHeaderTimeout,
		ReadTimeout:       e.readTimeout,
		WriteTimeout:      e.writeTimeout,
		IdleTimeout:       e.idleTimeout,
	}
	e.serverMu.Lock()
	e.server = srv
	e.serverMu.Unlock()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		stop() // restore default signal handling so a second signal force-quits
		slog.Info("shutting down server", "timeout", e.shutdownTimeout)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), e.shutdownTimeout)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

// Shutdown gracefully shuts down the running server without interrupting any
// active connections, respecting the provided context's deadline. It is safe to
// call from another goroutine. Returns nil if Serve was never started.
func (e *Engine) Shutdown(ctx context.Context) error {
	e.serverMu.Lock()
	srv := e.server
	e.serverMu.Unlock()
	if srv == nil {
		return nil
	}
	return srv.Shutdown(ctx)
}

func (e *Engine) devProxyForRequest() (*httputil.ReverseProxy, error) {
	e.devProxyOnce.Do(func() {
		target, err := url.Parse(e.devAddr)
		if err != nil {
			e.devProxyErr = err
			return
		}

		e.devProxy = &httputil.ReverseProxy{
			Rewrite: func(pr *httputil.ProxyRequest) {
				pr.SetURL(target)
				// Keep the original request path and query (avoid target path joining).
				pr.Out.URL.Path = pr.In.URL.Path
				pr.Out.URL.RawQuery = pr.In.URL.RawQuery
				pr.SetXForwarded()
			},
			ErrorHandler: func(rw http.ResponseWriter, req *http.Request, eErr error) {
				slog.Error("proxyToDevServer: proxy error", slog.Any("error", eErr))
				ErrorHandlerMap[http.StatusBadGateway](rw, req, eErr)
			},
		}
	})

	if e.devProxyErr != nil {
		return nil, e.devProxyErr
	}
	return e.devProxy, nil
}

// proxyToDevServer uses ReverseProxy to forward the request (including websocket upgrades) to dev server
func (e *Engine) proxyToDevServer(w http.ResponseWriter, r *http.Request) {
	proxy, err := e.devProxyForRequest()
	if err != nil {
		slog.Error("proxyToDevServer: init proxy error", slog.Any("error", err))
		ErrorHandlerMap[http.StatusInternalServerError](w, r, err)
		return
	}
	proxy.ServeHTTP(w, r)
}
