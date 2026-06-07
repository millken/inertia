package inertia

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// freePort asks the OS for an available TCP port and returns it as ":port".
func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()
	return l.Addr().String()
}

func TestServeGracefulShutdown(t *testing.T) {
	addr := freePort(t)
	e, err := New(func(e *Engine) error { e.addr = addr; return nil })
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	e.GET("/ping", func(c *Context) { c.Writer.WriteString("pong") })

	serveErr := make(chan error, 1)
	go func() { serveErr <- e.Serve() }()

	// Wait until the server is accepting connections.
	deadline := time.Now().Add(2 * time.Second)
	for {
		resp, err := http.Get("http://" + addr + "/ping")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusOK)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("server never became reachable: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	select {
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("Serve returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not return after Shutdown")
	}
}

func TestShutdownBeforeServe(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := e.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown before Serve should be nil, got %v", err)
	}
}

func TestClientIPTrustProxyHeaders(t *testing.T) {
	newReq := func() *http.Request {
		r, _ := http.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = "10.0.0.1:1234"
		r.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")
		r.Header.Set("X-Real-IP", "9.9.9.9")
		return r
	}

	// Default: trust proxy headers -> first XFF hop.
	eTrust, _ := New()
	cTrust := &Context{engine: eTrust, Request: newReq()}
	if got := cTrust.ClientIP(); got != "1.2.3.4" {
		t.Fatalf("trust=true: got %q, want %q", got, "1.2.3.4")
	}

	// Disabled: ignore headers, fall back to RemoteAddr host.
	eNoTrust, _ := New(WithTrustProxyHeaders(false))
	cNoTrust := &Context{engine: eNoTrust, Request: newReq()}
	if got := cNoTrust.ClientIP(); got != "10.0.0.1" {
		t.Fatalf("trust=false: got %q, want %q", got, "10.0.0.1")
	}
}

func TestJSONHeaders(t *testing.T) {
	e, _ := New()
	e.GET("/data", func(c *Context) { _ = c.JSON(map[string]int{"n": 1}) })

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/data", nil))
	h := rec.Result().Header

	if got := h.Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", got)
	}
	if got := h.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q", got)
	}
	if got := h.Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q", got)
	}
	// Document-oriented and redundant headers must no longer appear on JSON.
	for _, k := range []string{"Content-Security-Policy", "X-Frame-Options", "X-Xss-Protection", "Pragma", "Expires"} {
		if v := h.Get(k); v != "" {
			t.Errorf("JSON should not set %s, got %q", k, v)
		}
	}
}

func TestRenderHTMLHeaders(t *testing.T) {
	e, _ := New(WithContentSecurityPolicy("default-src 'self'"))
	e.GET("/", func(c *Context) { _ = c.Render("Home") })

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	h := rec.Result().Header

	if got := h.Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", got)
	}
	if got := h.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q", got)
	}
	if got := h.Get("X-Frame-Options"); got != "SAMEORIGIN" {
		t.Errorf("X-Frame-Options = %q", got)
	}
	if got := h.Get("Content-Security-Policy"); got != "default-src 'self'" {
		t.Errorf("Content-Security-Policy = %q", got)
	}
}

func TestRenderNoCSPByDefault(t *testing.T) {
	e, _ := New()
	e.GET("/", func(c *Context) { _ = c.Render("Home") })

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if got := rec.Result().Header.Get("Content-Security-Policy"); got != "" {
		t.Errorf("CSP should be unset by default, got %q", got)
	}
}

func TestTimeoutOptions(t *testing.T) {
	e, err := New(
		WithReadHeaderTimeout(3*time.Second),
		WithReadTimeout(4*time.Second),
		WithWriteTimeout(5*time.Second),
		WithIdleTimeout(6*time.Second),
		WithShutdownTimeout(7*time.Second),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if e.readHeaderTimeout != 3*time.Second ||
		e.readTimeout != 4*time.Second ||
		e.writeTimeout != 5*time.Second ||
		e.idleTimeout != 6*time.Second ||
		e.shutdownTimeout != 7*time.Second {
		t.Fatalf("timeout options not applied: %+v", e)
	}
}
