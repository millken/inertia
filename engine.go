package inertia

import (
	"io/fs"
	"net/http"

	"github.com/millken/inertia/pkg/router"
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

type Engine struct {
	DevMode            bool
	MaxMultipartMemory int64
	rootTemplateHTML   string
	addr               string
	router             *router.Router[HandlerFunc]
	middleware         []HandlerFunc
}

func New(options ...Option) (*Engine, error) {
	e := &Engine{
		addr:               ":5000",
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
		e.rootTemplateHTML = html
		return nil
	}
}

func (e *Engine) Get(path string, fn func(c *Context)) {

	e.router.Add("GET", path, fn)
}
func (e *Engine) Render() {

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
		ctx := acquireContext()
		ctx.Request = r
		ctx.Writer = w
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
