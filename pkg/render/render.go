package render

import (
	"context"
	"io/fs"
	"net/http"
	"sync"
)

type Render struct {
	rootTemplateHTML []byte
	svelteFS         fs.FS
	compileFS        fs.FS
	startTag, endTag string

	compiled sync.Map
}

func New() *Render {
	return &Render{
		startTag: "<%",
		endTag:   "%>",
	}
}

func (r *Render) SetSvelteFS(fs fs.FS) {
	r.svelteFS = fs
}

func (r *Render) SetCompileFS(fs fs.FS) {
	r.compileFS = fs
}

func (r *Render) SetRootTemplateHTML(html []byte) {
	r.rootTemplateHTML = html
}

func (rr *Render) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), Ctx{}, &View{
				render:  rr,
				data:    make(map[string]any),
				request: r,
				writer:  w,
			})

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
