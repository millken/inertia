# inertia

A small, fast [Inertia.js](https://inertiajs.com/)-style server framework for Go.
It pairs a Gin-flavored router/middleware API with server-side rendering (SSR),
so a Go backend can drive a Vue/React single-page app without writing a separate
JSON API.

- **Zero-allocation radix router** (generic `Router[T]`).
- **Three run modes** — production, development (Vite reverse proxy), and SSR.
- **Pooled everything** — `Context`, JSON buffers, gzip writers.
- **Security-conscious defaults** — `nosniff` + `no-store` on `JSON()`,
  document headers (`nosniff`, `X-Frame-Options`, optional CSP) on `Render()`,
  and HTML/JSON escaping for injected page data.
- **Optional SSR engines** split into their own modules (goja / QuickJS / V8) so
  you only pull in the heavy dependency you actually use.

> Status: pre-1.0 in spirit — the API may still change. Modules are versioned
> independently via a Go workspace.

## Install

```sh
go get github.com/millken/inertia
```

The middleware and SSR engines are separate modules:

```sh
go get github.com/millken/inertia/middleware
go get github.com/millken/inertia/ssr/quickjs   # or .../ssr/goja, .../ssr/v8go
```

## Quick start

```go
package main

import (
	"log"

	"github.com/millken/inertia"
	"github.com/millken/inertia/middleware"
)

func main() {
	e, err := inertia.New(
		inertia.WithMode(inertia.ModeProduction),
	)
	if err != nil {
		log.Fatal(err)
	}

	e.Use(
		middleware.Recovery(),
		middleware.AccessLog(),
		middleware.Gzip(),
	)

	e.GET("/", func(c *inertia.Context) {
		c.Set("message", "Hello, Inertia!")
		if err := c.Render("Home"); err != nil {
			log.Println(err)
		}
	})

	e.GET("/users/:id", func(c *inertia.Context) {
		id := c.Params.Get("id")
		_ = c.JSON(map[string]string{"id": id})
	})

	// Serve blocks until SIGINT/SIGTERM, then shuts down gracefully.
	log.Fatal(e.Serve())
}
```

## Run modes

`WithMode(...)` selects how requests are served:

| Mode | Constant | Behavior |
|------|----------|----------|
| Production | `ModeProduction` (default) | Serves the embedded root HTML template and static assets. |
| Development | `ModeDevelopment` | Requests that don't match a route are reverse-proxied to the Vite dev server (`WithDevAddr`, default `http://localhost:5173`), including websocket upgrades for HMR. |
| SSR | `ModeSSR` | Renders the page component server-side through a JS engine (`WithSSR`). |

## Rendering

`c.Render(view)` writes the root HTML document, substituting the comment-marked
placeholders in the template:

| Placeholder | Filled with |
|-------------|-------------|
| `<!--inertia-head-meta-inertia-->` | `c.Meta.ToHTML()` |
| `<!--inertia-ssr-content-inertia-->` | SSR output (SSR mode only) |
| `<!--inertia-data-page-inertia-->` | JSON of the data set via `c.Set(...)`, plus the view name, HTML-escaped for safe embedding |
| `<!--inertia-version-inertia-->` | a cache-busting build token |

The comment-style tags are intentional: they survive Vue/React templating without
being stripped. Override them with `WithTags(start, end)` and the document with
`WithRootHTML(html)`.

For partial updates, a request carrying `X-Pjax: true` gets the page data back as
JSON instead of a full HTML render.

## Static assets

```go
//go:embed dist
var dist embed.FS

e.StaticFS("/assets/", dist)
```

Embedded (`embed.FS`) assets get weak ETags and `If-None-Match` handling; ordinary
`fs.FS` values are streamed with `Last-Modified`. In development mode static
serving is skipped (the Vite dev server handles it).

## Middleware

`github.com/millken/inertia/middleware` provides:

- `Recovery()` / `RecoveryWithConfig(...)` — recover from panics, log a formatted stack.
- `AccessLog(...)` — structured request logging (`slog`).
- `Gzip(...)` — response compression with a pooled writer and extension excludes.
- `HTTPDump()` — print each request as an equivalent `curl` command (debugging only).

Register globally with `e.Use(...)`. A handler is just `func(c *inertia.Context)`;
call `c.Next()` to continue the chain and `c.Abort()` to stop it.

## Options

| Option | Default | Purpose |
|--------|---------|---------|
| `WithMode(mode)` | `ModeProduction` | Select run mode. |
| `WithRootHTML(html)` | built-in template | Override the root HTML document. |
| `WithTags(start, end)` | `<!--inertia-` / `-inertia-->` | Placeholder delimiters. |
| `WithDevAddr(addr)` | `http://localhost:5173` | Vite dev server to proxy to. |
| `WithSSR(vm)` | none | SSR engine (`ssr.VM`). |
| `WithErrorHandler(status, fn)` | built-ins | Custom error responses. |
| `WithReadHeaderTimeout(d)` | `10s` | Bounds header reads (Slowloris defense). |
| `WithReadTimeout(d)` | `0` (off) | Whole-request read deadline. Leave off for uploads/streaming. |
| `WithWriteTimeout(d)` | `0` (off) | Response write deadline. Leave off for SSE/websockets. |
| `WithIdleTimeout(d)` | `120s` | Keep-alive idle timeout. |
| `WithShutdownTimeout(d)` | `10s` | Grace period for in-flight requests on shutdown. |
| `WithTrustProxyHeaders(b)` | `true` | Whether `ClientIP` honors `X-Forwarded-For` / `X-Real-IP`. |
| `WithContentSecurityPolicy(csp)` | `""` (unset) | CSP header for HTML rendered by `Render`. See note below. |

## Response security headers

- `JSON()` sets `Content-Type: application/json`, `X-Content-Type-Options: nosniff`
  and `Cache-Control: no-store`. Document-oriented headers (CSP, `X-Frame-Options`)
  are inert on a JSON body and are intentionally not set here.
- `Render()` emits an HTML document with `Content-Type: text/html`,
  `X-Content-Type-Options: nosniff`, `X-Frame-Options: SAMEORIGIN`, and — if
  configured — your `Content-Security-Policy`.

> **CSP note:** the built-in root template contains an inline bootstrap
> `<script>`. A strict policy like `script-src 'self'` will block it. Supply a
> policy that permits the inline script (e.g. via a `nonce`/hash) and a matching
> `WithRootHTML` template, or leave CSP unset.

## Graceful shutdown

`Serve()` installs sane server timeouts and listens for `SIGINT`/`SIGTERM`. On
signal it drains in-flight requests within `WithShutdownTimeout` (default 10s); a
second signal forces an exit. You can also trigger shutdown programmatically from
another goroutine:

```go
go func() {
	<-someStopSignal
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = e.Shutdown(ctx)
}()
log.Fatal(e.Serve())
```

## Client IP

`c.ClientIP()` returns the first `X-Forwarded-For` hop (then `X-Real-IP`, then the
connection's `RemoteAddr`). These headers are trusted by default. **If the server
faces untrusted clients directly, disable trust** so the headers can't be spoofed:

```go
e, _ := inertia.New(inertia.WithTrustProxyHeaders(false))
```

## License

See [LICENSE](LICENSE).
