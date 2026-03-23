// Package web provides a fast standalone HTTP server built from the extracted raw engine.
package web

import (
	"context"
	"io"
	"time"
)

type Option func(*Engine) error

type Engine struct {
	addr            string
	server          *Server
	autoTLSCacheDir string
}

func New(options ...Option) (*Engine, error) {
	engine := &Engine{addr: ":5001", server: NewServer(), autoTLSCacheDir: ".autocert"}
	for _, option := range options {
		if err := option(engine); err != nil {
			return nil, err
		}
	}
	return engine, nil
}

func WithAddr(addr string) Option {
	return func(engine *Engine) error {
		engine.addr = addr
		return nil
	}
}

func WithErrorHandler(handler func(*Context, error)) Option {
	return func(engine *Engine) error {
		engine.server.SetErrorHandler(handler)
		return nil
	}
}

func WithAutoTLSCacheDir(dir string) Option {
	return func(engine *Engine) error {
		engine.autoTLSCacheDir = dir
		return nil
	}
}

func WithShutdownTimeout(timeout time.Duration) Option {
	return func(engine *Engine) error {
		engine.server.SetShutdownTimeout(timeout)
		return nil
	}
}

func WithReadTimeout(timeout time.Duration) Option {
	return func(engine *Engine) error {
		engine.server.SetReadTimeout(timeout)
		return nil
	}
}

func WithWriteTimeout(timeout time.Duration) Option {
	return func(engine *Engine) error {
		engine.server.SetWriteTimeout(timeout)
		return nil
	}
}

func WithIdleTimeout(timeout time.Duration) Option {
	return func(engine *Engine) error {
		engine.server.SetIdleTimeout(timeout)
		return nil
	}
}

func WithMaxHeaderBytes(limit int) Option {
	return func(engine *Engine) error {
		engine.server.SetMaxHeaderBytes(limit)
		return nil
	}
}

func WithReadBufferSize(size int) Option {
	return func(engine *Engine) error {
		engine.server.SetReadBufferSize(size)
		return nil
	}
}

func WithWriteBufferSize(size int) Option {
	return func(engine *Engine) error {
		engine.server.SetWriteBufferSize(size)
		return nil
	}
}

func WithMaxBodyBytes(limit int64) Option {
	return func(engine *Engine) error {
		engine.server.SetMaxBodyBytes(limit)
		return nil
	}
}

func (e *Engine) Addr() string                             { return e.addr }
func (e *Engine) Server() *Server                          { return e.server }
func (e *Engine) GET(path string, handler HandlerFunc)     { e.server.GET(path, handler) }
func (e *Engine) POST(path string, handler HandlerFunc)    { e.server.POST(path, handler) }
func (e *Engine) PUT(path string, handler HandlerFunc)     { e.server.PUT(path, handler) }
func (e *Engine) DELETE(path string, handler HandlerFunc)  { e.server.DELETE(path, handler) }
func (e *Engine) PATCH(path string, handler HandlerFunc)   { e.server.PATCH(path, handler) }
func (e *Engine) OPTIONS(path string, handler HandlerFunc) { e.server.OPTIONS(path, handler) }
func (e *Engine) HEAD(path string, handler HandlerFunc)    { e.server.HEAD(path, handler) }
func (e *Engine) ANY(path string, handler HandlerFunc)     { e.server.ANY(path, handler) }
func (e *Engine) Use(middleware ...HandlerFunc)            { e.server.Use(middleware...) }

func (e *Engine) Request(method string, target string, headers []Header, body []byte) Response {
	if body == nil {
		return e.server.Request(method, target, headers, nil)
	}
	reader := bytesReader(body)
	return e.server.Request(method, target, headers, reader)
}

func (e *Engine) Run() error { return e.server.Run(e.addr) }

func (e *Engine) RunTLS(certFile string, keyFile string) error {
	return e.server.RunTLS(e.addr, certFile, keyFile)
}

func (e *Engine) RunAutoTLS(hosts ...string) error {
	return e.server.RunAutoTLS(e.addr, e.autoTLSCacheDir, hosts...)
}

func (e *Engine) RunAutoTLSRedirect(httpAddr string, hosts ...string) error {
	return e.server.RunAutoTLSRedirect(e.addr, httpAddr, e.autoTLSCacheDir, hosts...)
}

func (e *Engine) Shutdown(ctx context.Context) error {
	return e.server.Shutdown(ctx)
}

type byteSliceReader struct {
	data []byte
	off  int
}

func bytesReader(data []byte) *byteSliceReader { return &byteSliceReader{data: data} }

func (r *byteSliceReader) Reset(data []byte) {
	r.data = data
	r.off = 0
}

func (r *byteSliceReader) Bytes() []byte { return r.data }

func (r *byteSliceReader) Read(p []byte) (int, error) {
	if r.off >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.off:])
	r.off += n
	return n, nil
}
