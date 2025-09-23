package inertia

import (
	"fmt"
	"io/fs"
	"net/http"

	"github.com/millken/inertia/pkg/router"
)

var (
	defaultRootHTML = `<!DOCTYPE html>
<html lang="en">

<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>inertia</title>
</head>

<body>
  <div id="app" data-page="<%data-page%>"></div>
  <script defer type="module" src="/main.js"></script>
</body>`
)

type HandlerFunc func(c *Context)

// Option is an option parameter that modifies Inertia.
type Option func(e *Engine) error

func WithErrorHandler(status int, errorHandlerFn errorHandlerFn) Option {
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

// Engine is the main Inertia instance that holds the router and middleware.

type Engine struct {
	DevMode            bool
	MaxMultipartMemory int64
	rootHTML           string
	startTag, endTag   string
	viewFS             fs.FS
	addr               string
	router             *router.Router[HandlerFunc]
	middleware         []HandlerFunc
}

func New(options ...Option) (*Engine, error) {
	e := &Engine{
		addr:               ":5000",
		rootHTML:           defaultRootHTML,
		startTag:           "<%",
		endTag:             "%>",
		MaxMultipartMemory: 32 << 20, // 32 MB
		router:             router.New[HandlerFunc](),
	}
	for _, option := range options {
		if err := option(e); err != nil {
			return nil, err
		}
	}
	return e, nil
}

// WithRootTemplateHTML sets the root template HTML for Inertia.
func WithRootTemplateHTML(html string) Option {
	return func(e *Engine) error {
		e.rootHTML = html
		return nil
	}
}

func (e *Engine) Get(path string, fn func(c *Context)) {

	e.router.Add("GET", path, fn)
}

func (e *Engine) Addr() string {
	return e.addr
}

func (e *Engine) handleHttpRequest(c *Context) {

}

// ServeAsset serves static assets from the given path
func (e *Engine) ServeAsset(path string, fs fs.FS) {
	e.Get(path+"*", FileServer(path, fs))
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
		ctx.index = -1

		// Start execution chain
		ctx.Next()

		releaseContext(ctx)
		return
	}
	defaultCatchAllHandler.ServeHTTP(w, r)
}

func (e *Engine) Serve() error {
	return http.ListenAndServe(e.Addr(), e)
}
