# inertia/web

inertia/web is a standalone high-performance HTTP server module extracted from the raw engine prototype.

It does not replace the root inertia Engine. The root package keeps its existing net/http-based behavior, while this module is intended for building independent fast web services.

## Features

- standalone TCP HTTP server
- optional AutoSSL via ACME autocert
- shared router implementation from inertia/router
- lightweight context and middleware chain
- synthetic request API for tests and benchmarks
- hand-written request parsing and response serialization

## Install

```bash
go get github.com/millken/inertia/web
```

## Quick Start

```go
package main

import (
	"log"

	"github.com/millken/inertia/web"
)

func main() {
	engine, err := web.New(web.WithAddr(":8080"))
	if err != nil {
		log.Fatal(err)
	}

	engine.Use(func(c *web.Context) error {
		c.Header("X-Powered-By", "inertia/web")
		return c.Next()
	})

	engine.GET("/", func(c *web.Context) error {
		return c.String("hello, web")
	})

	engine.GET("/users/:id", func(c *web.Context) error {
		return c.JSON(map[string]any{
			"id":   c.Request.Param("id"),
			"tag":  c.Query("tag"),
			"addr": c.ClientIP(),
		})
	})

	log.Printf("listening on %s", engine.Addr())
	log.Fatal(engine.Run())
}
```

## Examples

- HTTP: [examples/hello/main.go](examples/hello/main.go)
- Manual TLS: [examples/tls/main.go](examples/tls/main.go)
- AutoSSL: [examples/autossl/main.go](examples/autossl/main.go)

## Run Example

```bash
cd web
go run ./examples/hello
```

Then open:

- http://127.0.0.1:8080/
- http://127.0.0.1:8080/users/42?tag=go

Manual TLS example:

```bash
cd web
go run ./examples/tls
```

Expected local files:

- `server.crt`
- `server.key`

AutoSSL example:

```bash
cd web
go run ./examples/autossl
```

Before running AutoSSL:

- replace `example.com` and `www.example.com` in [examples/autossl/main.go](examples/autossl/main.go)
- make sure ports `80` and `443` are publicly reachable
- make sure the domain resolves to this server

## API Overview

### Engine

- `web.New(options...)`
- `web.WithAddr(addr)`
- `web.WithErrorHandler(handler)`
- `web.WithShutdownTimeout(timeout)`
- `web.WithReadTimeout(timeout)`
- `web.WithWriteTimeout(timeout)`
- `web.WithIdleTimeout(timeout)`
- `web.WithReadBufferSize(size)`
- `web.WithWriteBufferSize(size)`
- `web.WithMaxHeaderBytes(limit)`
- `web.WithMaxBodyBytes(limit)`
- `(*Engine).Use(...)`
- `(*Engine).GET/POST/PUT/DELETE/PATCH/OPTIONS/HEAD/ANY(...)`
- `(*Engine).Run()`
- `(*Engine).RunTLS(certFile, keyFile)`
- `(*Engine).RunAutoTLS(hosts...)`
- `(*Engine).RunAutoTLSRedirect(httpAddr, hosts...)`
- `(*Engine).Shutdown(ctx)`
- `web.WithAutoTLSCacheDir(dir)`

## Graceful Shutdown

`inertia/web` now supports graceful shutdown for the raw server and the optional AutoSSL redirect server.

```go
engine, err := web.New(
	web.WithAddr(":8080"),
	web.WithShutdownTimeout(10*time.Second),
)
if err != nil {
	log.Fatal(err)
}

go func() {
	log.Fatal(engine.Run())
}()

shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := engine.Shutdown(shutdownCtx); err != nil {
	log.Fatal(err)
}
```

Shutdown behavior:

- stop accepting new TCP connections
- stop the optional AutoSSL HTTP redirect server
- wait for active connections to finish until the shutdown timeout expires
- force-close remaining connections if the timeout is exceeded

## Runtime Limits

`inertia/web` also supports finer runtime controls for connection handling:

```go
engine, err := web.New(
	web.WithAddr(":8080"),
	web.WithReadTimeout(5*time.Second),
	web.WithWriteTimeout(10*time.Second),
	web.WithIdleTimeout(30*time.Second),
	web.WithReadBufferSize(8<<10),
	web.WithWriteBufferSize(4<<10),
	web.WithMaxHeaderBytes(1<<20),
	web.WithMaxBodyBytes(8<<20),
	web.WithShutdownTimeout(10*time.Second),
)
if err != nil {
	log.Fatal(err)
}
```

Behavior:

- `WithReadTimeout`: limits how long a request may take to arrive and finish reading
- `WithWriteTimeout`: limits how long a response write may block
- `WithIdleTimeout`: limits how long a keep-alive connection may sit idle waiting for the next request
- `WithReadBufferSize`: controls the initial `bufio.Reader` buffer size used for socket reads
- `WithWriteBufferSize`: controls the initial response serialization buffer size before writing to the connection
- `WithMaxHeaderBytes`: rejects oversized request line + headers with `431 Request Header Fields Too Large`
- `WithMaxBodyBytes`: rejects oversized request bodies with `413 Payload Too Large`

## Manual TLS

If you already have a certificate and private key, run the server directly with TLS:

```go
engine, err := web.New(web.WithAddr(":443"))
if err != nil {
	log.Fatal(err)
}

engine.GET("/", func(c *web.Context) error {
	return c.String("hello over tls")
})

log.Fatal(engine.RunTLS("server.crt", "server.key"))
```

## AutoSSL

`inertia/web` now supports a minimal built-in AutoSSL path using ACME and `autocert`.

```go
package main

import (
	"log"

	"github.com/millken/inertia/web"
)

func main() {
	engine, err := web.New(
		web.WithAddr(":443"),
		web.WithAutoTLSCacheDir(".autocert"),
	)
	if err != nil {
		log.Fatal(err)
	}

	engine.GET("/", func(c *web.Context) error {
		return c.String("hello over https")
	})

	log.Fatal(engine.RunAutoTLS("example.com", "www.example.com"))
}
```

If you also want HTTP port 80 to redirect to HTTPS and allow HTTP-01 handling, use:

```go
log.Fatal(engine.RunAutoTLSRedirect(":80", "example.com", "www.example.com"))
```

Requirements:

- the domain must resolve to this server
- port 443 must be reachable from the public internet
- this mode uses TLS-ALPN-01 challenge, so direct TLS termination must happen in this process
- `RunAutoTLSRedirect` also starts an HTTP server for redirect and ACME HTTP handling
- if you already terminate TLS in Caddy or Nginx, do not use `RunAutoTLS`; keep `Run()` behind the proxy instead

### Context

- `c.Next()`
- `c.Query(key)`
- `c.GetPostForm(key)`
- `c.ClientIP()`
- `c.Header(key, value)`
- `c.Status(code)`
- `c.String(body)`
- `c.Bytes(body)`
- `c.JSON(data)`

### Request

- `c.Request.Method()`
- `c.Request.Path()`
- `c.Request.Param(name)`
- `c.Request.Query()`
- `c.Request.Header(key)`
- `c.Request.Body()`

## Notes

- graceful shutdown is supported through `(*Engine).Shutdown(ctx)`
- AutoSSL is available through ACME TLS-ALPN-01, but HTTP/2 is still not supported
- the root inertia Engine and inertia/web are intentionally separate runtimes