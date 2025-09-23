package inertia

import (
	"bytes"
	"embed"
	"fmt"
	"hash/fnv"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"sync"
)

type fileServer struct {
	prefix  string
	root    fs.FS
	isEmbed bool
	mu      sync.Mutex
	etags   map[string]string
}

// FileServer returns an inertia.HandlerFunc that serves files from the given fs.FS.
func FileServer(prefix string, root fs.FS) HandlerFunc {
	_, ok := root.(embed.FS)
	srv := &fileServer{
		prefix:  prefix,
		root:    root,
		isEmbed: ok,
		etags:   make(map[string]string),
	}

	return func(c *Context) {
		r := c.Request
		w := c.Writer

		path := strings.TrimPrefix(r.URL.Path, prefix)
		if path == "" || path[len(path)-1] == '/' {
			path = path + "index.html"
		}

		// friendly error handler helper
		handleError := func(status int, err error) {
			if h, ok := ErrorHandlerMap[status]; ok {
				h(w, r, err)
				return
			}
			if status == http.StatusInternalServerError {
				http.Error(w, http.StatusText(status)+": internal server error", status)
			} else {
				http.Error(w, http.StatusText(status), status)
			}
		}

		if srv.isEmbed {
			ifNoneMatch := r.Header.Get("If-None-Match")
			srv.mu.Lock()
			etag, ok := srv.etags[path]
			srv.mu.Unlock()
			if ok && ifNoneMatch != "" && ifNoneMatch == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		fi, err := srv.root.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				handleError(http.StatusNotFound, err)
				return
			}
			if os.IsPermission(err) {
				handleError(http.StatusForbidden, err)
				return
			}
			handleError(http.StatusInternalServerError, err)
			return
		}
		defer fi.Close()

		f, err := fi.Stat()
		if err != nil {
			if os.IsNotExist(err) {
				handleError(http.StatusNotFound, err)
				return
			}
			if os.IsPermission(err) {
				handleError(http.StatusForbidden, err)
				return
			}
			handleError(http.StatusInternalServerError, err)
			return
		}

		content, err := io.ReadAll(fi)
		if err != nil {
			handleError(http.StatusInternalServerError, err)
			return
		}

		h := fnv.New64a()
		h.Write(content)
		etag := fmt.Sprintf("W/%x", h.Sum64())
		if srv.isEmbed {
			srv.mu.Lock()
			srv.etags[path] = etag
			srv.mu.Unlock()
		} else {
			w.Header().Set("Last-Modified", f.ModTime().UTC().Format(http.TimeFormat))
		}

		w.Header().Set("Vary", "Accept-Encoding")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Header().Set("Etag", etag)

		http.ServeContent(w, r, f.Name(), f.ModTime(), bytes.NewReader(content))
	}
}
