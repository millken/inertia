// Package inertia provides a Go web framework for building modern web applications using the Inertia.js approach.
package inertia

import (
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
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

func (e *Engine) Serve() error {
	if e.IsDevelopmentMode() {
		slog.Info("starting server", "mode", "development", "proxy", e.devAddr)
	} else if e.IsSSRMode() {
		slog.Info("starting server", "mode", "ssr", "addr", e.Addr())
	} else {
		slog.Info("starting server", "mode", "production", "addr", e.Addr())
	}
	return http.ListenAndServe(e.Addr(), e)
}

func (e *Engine) devProxyForRequest() (*httputil.ReverseProxy, error) {
	e.devProxyOnce.Do(func() {
		target, err := url.Parse(e.devAddr)
		if err != nil {
			e.devProxyErr = err
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(target)
		origDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			origPath := req.URL.Path
			origRawQuery := req.URL.RawQuery
			origDirector(req)
			// Keep the original request URL path and query (avoid target path joining).
			req.URL.Path = origPath
			req.URL.RawQuery = origRawQuery
			req.Host = target.Host
		}
		proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, eErr error) {
			slog.Error("proxyToDevServer: proxy error", slog.Any("error", eErr))
			ErrorHandlerMap[http.StatusBadGateway](rw, req, eErr)
		}
		e.devProxy = proxy
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
