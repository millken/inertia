package web

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/millken/inertia/router"
)

func TestRequestQueryAndParams(t *testing.T) {
	var req Request
	req.reset()
	req.headers = append(req.headers,
		Header{Key: "Host", Value: "example.com"},
		Header{Key: "User-Agent", Value: "web-test"},
	)
	req.params = append(req.params, router.Parameter{Key: "id", Value: "42"})
	req.setContext(context.Background())
	req.setRemoteAddr("127.0.0.1:9000")
	req.setURLParts("http", "", "/posts/42", Query("tag=go&name=hello+world"))
	if req.Host() != "example.com" || req.Param("id") != "42" || req.Query().Param("name") != "hello world" {
		t.Fatal("request metadata mismatch")
	}
}

func TestRequestReadBody(t *testing.T) {
	var req Request
	req.reset()
	req.headers = append(req.headers, Header{Key: "Content-Length", Value: "7"})
	req.reader.Reset(strings.NewReader("payload"))
	data, err := io.ReadAll(&req)
	if err != nil || string(data) != "payload" {
		t.Fatalf("ReadAll() = %q %v", string(data), err)
	}
}

func TestRequestPostForm(t *testing.T) {
	var req Request
	req.reset()
	req.setOwnedBody([]byte("tag=go&tag=rust&role=&na%6De=millken"))
	if value, ok := req.GetPostForm("name"); !ok || value != "millken" {
		t.Fatalf("GetPostForm(name) = %q %v", value, ok)
	}
	values, ok := req.GetPostFormArray("tag")
	if !ok || len(values) != 2 {
		t.Fatalf("GetPostFormArray(tag) = %#v %v", values, ok)
	}
}

func TestResponseSerializeAndWriteTo(t *testing.T) {
	var res Response
	res.reset()
	res.SetHeader("Content-Type", "text/plain")
	res.WriteHeader(201)
	_, _ = res.WriteString("hello")
	payload := string(res.AppendHTTP(nil))
	if !strings.Contains(payload, "HTTP/1.1 201") || !strings.HasSuffix(payload, "\r\n\r\nhello") {
		t.Fatalf("payload = %q", payload)
	}
	var buffer bytes.Buffer
	if _, err := res.WriteTo(&buffer); err != nil {
		t.Fatalf("WriteTo() error = %v", err)
	}
}

func TestNewEngine(t *testing.T) {
	engine, err := New(WithAddr(":7000"))
	if err != nil || engine.Addr() != ":7000" {
		t.Fatalf("New() = %v, %v", engine, err)
	}
	engine.GET("/hello/:name", func(c *Context) error {
		return c.String("hello " + c.Request.Param("name"))
	})
	response := engine.Request("GET", "/hello/millken", nil, nil)
	if string(response.Body()) != "hello millken" {
		t.Fatalf("body = %q", string(response.Body()))
	}
}

func TestEngineRunAutoTLSRequiresHosts(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := engine.RunAutoTLS(); err == nil {
		t.Fatal("RunAutoTLS() error = nil")
	}
}

func TestEngineRunTLSInvalidFiles(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := engine.RunTLS("missing.crt", "missing.key"); err == nil {
		t.Fatal("RunTLS() error = nil")
	}
}

func TestServerGracefulShutdown(t *testing.T) {
	server := NewServer()
	server.GET("/", func(c *Context) error {
		return c.String("ok")
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- server.serve(listener)
	}()

	<-server.Ready()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("serve() error = %v", err)
	}
}

func TestEngineRuntimeOptions(t *testing.T) {
	engine, err := New(
		WithReadTimeout(time.Second),
		WithWriteTimeout(2*time.Second),
		WithIdleTimeout(3*time.Second),
		WithShutdownTimeout(4*time.Second),
		WithReadBufferSize(8192),
		WithWriteBufferSize(4096),
		WithMaxHeaderBytes(2048),
		WithMaxBodyBytes(1024),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if engine.server.currentReadTimeout() != time.Second {
		t.Fatalf("read timeout = %v", engine.server.currentReadTimeout())
	}
	if engine.server.currentWriteTimeout() != 2*time.Second {
		t.Fatalf("write timeout = %v", engine.server.currentWriteTimeout())
	}
	if engine.server.currentIdleTimeout() != 3*time.Second {
		t.Fatalf("idle timeout = %v", engine.server.currentIdleTimeout())
	}
	if engine.server.currentShutdownTimeout() != 4*time.Second {
		t.Fatalf("shutdown timeout = %v", engine.server.currentShutdownTimeout())
	}
	if engine.server.currentMaxHeaderBytes() != 2048 {
		t.Fatalf("maxHeaderBytes = %d", engine.server.currentMaxHeaderBytes())
	}
	if engine.server.currentReadBufferSize() != 8192 {
		t.Fatalf("readBufferSize = %d", engine.server.currentReadBufferSize())
	}
	if engine.server.currentWriteBufferSize() != 4096 {
		t.Fatalf("writeBufferSize = %d", engine.server.currentWriteBufferSize())
	}
	if engine.server.currentMaxBodyBytes() != 1024 {
		t.Fatalf("maxBodyBytes = %d", engine.server.currentMaxBodyBytes())
	}
}

func TestServerRejectsLargeHeaders(t *testing.T) {
	server := NewServer()
	server.SetMaxHeaderBytes(32)
	client, conn := net.Pipe()
	defer client.Close()

	go server.handleConnection(conn)

	request := "GET /hello HTTP/1.1\r\nHost: example.com\r\nX-Long: 12345678901234567890\r\nConnection: close\r\n\r\n"
	if _, err := io.WriteString(client, request); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	response, err := io.ReadAll(bufio.NewReader(client))
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !strings.Contains(string(response), "431 Request Header Fields Too Large") {
		t.Fatalf("response = %q", string(response))
	}
}

func TestServerRejectsLargeBody(t *testing.T) {
	server := NewServer()
	server.SetMaxBodyBytes(8)
	client, conn := net.Pipe()
	defer client.Close()

	go server.handleConnection(conn)

	request := "POST /submit HTTP/1.1\r\nHost: example.com\r\nContent-Length: 12\r\nConnection: close\r\n\r\n"
	if _, err := io.WriteString(client, request); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	response, err := io.ReadAll(bufio.NewReader(client))
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !strings.Contains(string(response), "413 Payload Too Large") {
		t.Fatalf("response = %q", string(response))
	}
}

func TestBuildHTTPSRedirectURL(t *testing.T) {
	tests := []struct {
		host       string
		requestURI string
		port       string
		want       string
	}{
		{host: "example.com:80", requestURI: "/foo?bar=baz", port: "443", want: "https://example.com/foo?bar=baz"},
		{host: "example.com", requestURI: "/", port: "8443", want: "https://example.com:8443/"},
	}
	for _, test := range tests {
		if got := buildHTTPSRedirectURL(test.host, test.requestURI, test.port); got != test.want {
			t.Fatalf("buildHTTPSRedirectURL() = %q, want %q", got, test.want)
		}
	}
}

func TestAutoHTTPRedirectHandler(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://example.com/plain", nil)
	response := httptest.NewRecorder()
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, buildHTTPSRedirectURL(request.Host, request.URL.RequestURI(), "443"), http.StatusMovedPermanently)
	})
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d", response.Code)
	}
	if location := response.Header().Get("Location"); location != "https://example.com/plain" {
		t.Fatalf("Location = %q", location)
	}
}

func TestServerSyntheticRequest(t *testing.T) {
	server := NewServer()
	server.Use(func(c *Context) error {
		c.Header("X-Middleware", "ok")
		return c.Next()
	})
	server.GET("/users/:id", func(c *Context) error {
		return c.String(c.Request.Param("id") + ":" + c.Request.Query().Param("tag"))
	})
	response := server.Request("GET", "http://example.com/users/42?tag=go", []Header{{Key: "Host", Value: "example.com"}}, nil)
	if response.Status() != 200 || string(response.Body()) != "42:go" || response.Header("X-Middleware") != "ok" {
		t.Fatalf("unexpected response: %d %q", response.Status(), string(response.Body()))
	}
}

func TestServerHandleConnection(t *testing.T) {
	server := NewServer()
	server.GET("/hello", func(c *Context) error {
		c.Header("Content-Type", "text/plain")
		return c.String("world")
	})
	client, conn := net.Pipe()
	defer client.Close()
	go server.handleConnection(conn)
	if _, err := io.WriteString(client, "GET /hello HTTP/1.1\r\nHost: example.com\r\nConnection: close\r\n\r\n"); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	response, err := io.ReadAll(bufio.NewReader(client))
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	payload := string(response)
	if !strings.Contains(payload, "HTTP/1.1 200") || !strings.HasSuffix(payload, "\r\n\r\nworld") {
		t.Fatalf("payload = %q", payload)
	}
}

func TestServerNotFoundAndJSONAndPostForm(t *testing.T) {
	server := NewServer()
	if response := server.Request("GET", "/missing", nil, nil); response.Status() != 404 {
		t.Fatalf("status = %d", response.Status())
	}
	server.GET("/json", func(c *Context) error {
		if c.ClientIP() != "10.0.0.1" {
			t.Fatalf("ClientIP() = %q", c.ClientIP())
		}
		return c.JSON(map[string]string{"ok": "true"})
	})
	response := server.Request("GET", "/json", []Header{{Key: "X-Forwarded-For", Value: "10.0.0.1, 10.0.0.2"}}, nil)
	if !strings.Contains(string(response.Body()), `"ok":"true"`) {
		t.Fatalf("body = %q", string(response.Body()))
	}
	server.POST("/submit", func(c *Context) error {
		name, ok := c.GetPostForm("name")
		if !ok {
			return errors.New("missing form field")
		}
		return c.String(name + ":" + c.DefaultPostForm("role", "guest"))
	})
	response = server.Request("POST", "/submit", []Header{{Key: "Content-Length", Value: "23"}}, strings.NewReader("name=millken&role=admin"))
	if string(response.Body()) != "millken:admin" {
		t.Fatalf("body = %q", string(response.Body()))
	}
}