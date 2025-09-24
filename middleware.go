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

// StaticFileServer returns an inertia.HandlerFunc that serves files from the given fs.FS.
func StaticFileServer(prefix string, root fs.FS) HandlerFunc {
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
		if path == "" || strings.HasSuffix(path, "/") {
			path = path + "index.html"
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
				ErrorHandlerMap[http.StatusNotFound](w, r, err)
				return
			}
			if os.IsPermission(err) {
				ErrorHandlerMap[http.StatusForbidden](w, r, err)
				return
			}
			ErrorHandlerMap[http.StatusInternalServerError](w, r, err)
			return
		}
		defer fi.Close()

		f, err := fi.Stat()
		if err != nil {
			if os.IsNotExist(err) {
				ErrorHandlerMap[http.StatusNotFound](w, r, err)
				return
			}
			if os.IsPermission(err) {
				ErrorHandlerMap[http.StatusForbidden](w, r, err)
				return
			}
			ErrorHandlerMap[http.StatusInternalServerError](w, r, err)
			return
		}

		w.Header().Set("Vary", "Accept-Encoding")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		if srv.isEmbed {
			content, err := io.ReadAll(fi)
			if err != nil {
				ErrorHandlerMap[http.StatusInternalServerError](w, r, err)
				return
			}
			h := fnv.New64a()
			h.Write(content)
			etag := fmt.Sprintf("W/%x", h.Sum64())

			srv.mu.Lock()
			srv.etags[path] = etag
			srv.mu.Unlock()

			ifNoneMatch := r.Header.Get("If-None-Match")
			if ifNoneMatch != "" && ifNoneMatch == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}

			w.Header().Set("Etag", etag)
			http.ServeContent(w, r, f.Name(), f.ModTime(), bytes.NewReader(content))
		} else {
			// 普通FS直接用文件句柄，避免读入内存（如果支持 Seek）；否则退回到内存读取
			w.Header().Set("Last-Modified", f.ModTime().UTC().Format(http.TimeFormat))
			if rs, ok := fi.(io.ReadSeeker); ok {
				http.ServeContent(w, r, f.Name(), f.ModTime(), rs)
			} else {
				content, err := io.ReadAll(fi)
				if err != nil {
					ErrorHandlerMap[http.StatusInternalServerError](w, r, err)
					return
				}
				http.ServeContent(w, r, f.Name(), f.ModTime(), bytes.NewReader(content))
			}
		}
	}
}
