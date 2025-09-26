package inertia

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/millken/inertia/router"
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
  <script type="module" src="/main.js?v=<!--inertia-version-inertia-->"></script>
</body>`
)

type Mode byte

const (
	ModeProduction Mode = iota
	ModeDevelopment
	ModeSSR
)

type HandlerFunc func(c *Context)

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

func WithSSR(ssr SSR) Option {
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

type SSR interface {
	RenderTemplate(ctx context.Context, tpl string, data map[string]any) (string, error)
	RenderComponent(ctx context.Context, name string, data map[string]any) (string, error)
	RenderPage(ctx context.Context, url string, data map[string]any) (string, error)
	Close()
}

// Engine is the main Inertia instance that holds the router and middleware.

type Engine struct {
	mode               Mode
	devAddr            string
	MaxMultipartMemory int64
	rootHTML           string
	startTag, endTag   string
	ssr                SSR
	addr               string
	router             *router.Router[HandlerFunc]
	middleware         []HandlerFunc
}

func New(options ...Option) (*Engine, error) {
	var err error
	e := &Engine{
		mode:               ModeProduction,
		devAddr:            "http://localhost:5173",
		addr:               ":5000",
		rootHTML:           defaultRootHTML,
		startTag:           "<!--inertia-", //注释标记可以防止被前端框架（如 Vue、React）误删
		endTag:             "-inertia-->",
		MaxMultipartMemory: 32 << 20, // 32 MB
		router:             router.New[HandlerFunc](),
	}
	for _, option := range options {
		if err = option(e); err != nil {
			return nil, err
		}
	}
	return e, nil
}

// IsDevelopentMode returns true if the engine is in development mode
func (e *Engine) IsDevelopentMode() bool {
	return e.mode == ModeDevelopment
}

// IsSSRMode returns true if the engine is in SSR mode
func (e *Engine) IsSSRMode() bool {
	return e.mode == ModeSSR
}

func (e *Engine) GET(path string, fn func(c *Context)) {
	e.router.Add("GET", path, fn)
}

func (e *Engine) POST(path string, fn func(c *Context)) {
	e.router.Add("POST", path, fn)
}

func (e *Engine) PUT(path string, fn func(c *Context)) {
	e.router.Add("PUT", path, fn)
}

func (e *Engine) DELETE(path string, fn func(c *Context)) {
	e.router.Add("DELETE", path, fn)
}

func (e *Engine) PATCH(path string, fn func(c *Context)) {
	e.router.Add("PATCH", path, fn)
}

func (e *Engine) OPTIONS(path string, fn func(c *Context)) {
	e.router.Add("OPTIONS", path, fn)
}

func (e *Engine) HEAD(path string, fn func(c *Context)) {
	e.router.Add("HEAD", path, fn)
}

func (e *Engine) ANY(path string, fn func(c *Context)) {
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
	if e.IsDevelopentMode() {
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
		ctx.handlers = ctx.handlers[:0] // reset slice but keep capacity
		ctx.handlers = append(ctx.handlers, e.middleware...)
		ctx.handlers = append(ctx.handlers, fn)

		// Start execution chain
		ctx.Next()

		releaseContext(ctx)
		return
	}
	// 如果是 devMode，未命中路由的请求都转发到开发服务器
	if e.IsDevelopentMode() {
		e.proxyToDevServer(w, r)
		return
	}
	// not found
	defaultCatchAllHandler.ServeHTTP(w, r)
}

func (e *Engine) Serve() error {
	if e.IsDevelopentMode() {
		slog.Info(fmt.Sprintf("Starting in development mode, proxying to development server at %s", e.devAddr))
	} else if e.IsSSRMode() {
		slog.Info(fmt.Sprintf("Starting in SSR mode at http://%s", e.Addr()))
	} else {
		slog.Info(fmt.Sprintf("Starting in production mode at http://%s", e.Addr()))
	}
	return http.ListenAndServe(e.Addr(), e)
}

// proxyToDevServer uses ReverseProxy to forward the request (including websocket upgrades) to dev server
func (e *Engine) proxyToDevServer(w http.ResponseWriter, r *http.Request) {
	target, err := url.Parse(e.devAddr)
	if err != nil {
		slog.Error("proxyToDevServer: parse devAddr error", slog.Any("error", err))
		ErrorHandlerMap[http.StatusInternalServerError](w, r, err)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	// preserve original host header
	origDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		origDirector(req)
		// keep the original request URL path and query
		req.URL.Path = r.URL.Path
		req.URL.RawQuery = r.URL.RawQuery
		req.Host = target.Host
	}
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, eErr error) {
		slog.Error("proxyToDevServer: proxy error", slog.Any("error", eErr))
		ErrorHandlerMap[http.StatusBadGateway](rw, req, eErr)
	}
	proxy.ServeHTTP(w, r)
}
