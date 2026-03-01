package inertia

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"maps"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"
)

var _ context.Context = (*Context)(nil)

// ContextKey is the key that a Context returns itself for.
const ContextKey = "_inertia/contextkey"

type ContextKeyType int

const ContextRequestKey ContextKeyType = 0

type Context struct {
	writermem responseWriter
	Meta      Meta
	Request   *http.Request
	Writer    ResponseWriter
	Params    Params
	// queryCache caches the query result from c.Request.URL.Query().
	queryCache url.Values

	// formCache caches c.Request.PostForm, which contains the parsed form data from POST, PATCH,
	// or PUT body parameters.
	formCache url.Values
	engine    *Engine

	// middleware control
	handlers []HandlerFunc
	mu       sync.RWMutex
	data     map[string]any
	index    int8
}

var contextPool = sync.Pool{
	New: func() any {
		return &Context{}
	},
}

// Query returns the keyed url query value if it exists,
// otherwise it returns an empty string `("")`.
// It is shortcut for `c.Request.URL.Query().Get(key)`
//
//	    GET /path?id=1234&name=Manu&value=
//		   c.Query("id") == "1234"
//		   c.Query("name") == "Manu"
//		   c.Query("value") == ""
//		   c.Query("wtf") == ""
func (c *Context) Query(key string) (value string) {
	value, _ = c.GetQuery(key)
	return
}

// DefaultQuery returns the keyed url query value if it exists,
// otherwise it returns the specified defaultValue string.
// See: Query() and GetQuery() for further information.
//
//	GET /?name=Manu&lastname=
//	c.DefaultQuery("name", "unknown") == "Manu"
//	c.DefaultQuery("id", "none") == "none"
//	c.DefaultQuery("lastname", "none") == ""
func (c *Context) DefaultQuery(key, defaultValue string) string {
	if value, ok := c.GetQuery(key); ok {
		return value
	}
	return defaultValue
}

// GetQuery is like Query(), it returns the keyed url query value
// if it exists `(value, true)` (even when the value is an empty string),
// otherwise it returns `("", false)`.
// It is shortcut for `c.Request.URL.Query().Get(key)`
//
//	GET /?name=Manu&lastname=
//	("Manu", true) == c.GetQuery("name")
//	("", false) == c.GetQuery("id")
//	("", true) == c.GetQuery("lastname")
func (c *Context) GetQuery(key string) (string, bool) {
	if values, ok := c.GetQueryArray(key); ok {
		return values[0], ok
	}
	return "", false
}

// QueryArray returns a slice of strings for a given query key.
// The length of the slice depends on the number of params with the given key.
func (c *Context) QueryArray(key string) (values []string) {
	values, _ = c.GetQueryArray(key)
	return
}

func (c *Context) initQueryCache() {
	if c.queryCache == nil {
		if c.Request != nil && c.Request.URL != nil {
			c.queryCache = c.Request.URL.Query()
		} else {
			c.queryCache = url.Values{}
		}
	}
}

// GetQueryArray returns a slice of strings for a given query key, plus
// a boolean value whether at least one value exists for the given key.
func (c *Context) GetQueryArray(key string) (values []string, ok bool) {
	c.initQueryCache()
	values, ok = c.queryCache[key]
	return
}

// QueryMap returns a map for a given query key.
func (c *Context) QueryMap(key string) (dicts map[string]string) {
	dicts, _ = c.GetQueryMap(key)
	return
}

// GetQueryMap returns a map for a given query key, plus a boolean value
// whether at least one value exists for the given key.
func (c *Context) GetQueryMap(key string) (map[string]string, bool) {
	c.initQueryCache()
	return getMapFromFormData(c.queryCache, key)
}

// PostForm returns the specified key from a POST urlencoded form or multipart form
// when it exists, otherwise it returns an empty string `("")`.
func (c *Context) PostForm(key string) (value string) {
	value, _ = c.GetPostForm(key)
	return
}

// DefaultPostForm returns the specified key from a POST urlencoded form or multipart form
// when it exists, otherwise it returns the specified defaultValue string.
// See: PostForm() and GetPostForm() for further information.
func (c *Context) DefaultPostForm(key, defaultValue string) string {
	if value, ok := c.GetPostForm(key); ok {
		return value
	}
	return defaultValue
}

// GetPostForm is like PostForm(key). It returns the specified key from a POST urlencoded
// form or multipart form when it exists `(value, true)` (even when the value is an empty string),
// otherwise it returns ("", false).
// For example, during a PATCH request to update the user's email:
//
//	    email=mail@example.com  -->  ("mail@example.com", true) := GetPostForm("email") // set email to "mail@example.com"
//		   email=                  -->  ("", true) := GetPostForm("email") // set email to ""
//	                            -->  ("", false) := GetPostForm("email") // do nothing with email
func (c *Context) GetPostForm(key string) (string, bool) {
	if values, ok := c.GetPostFormArray(key); ok {
		return values[0], ok
	}
	return "", false
}
func (c *Context) initFormCache() {
	if c.formCache == nil {
		c.formCache = make(url.Values)
		req := c.Request
		if err := req.ParseMultipartForm(c.engine.MaxMultipartMemory); err != nil {
			if !errors.Is(err, http.ErrNotMultipart) {
				// We ignore http.ErrNotMultipart error
				// which indicates that the request body isn't a multipart form.
				// Other errors are unexpected and should be handled.
				// See: https://golang.org/pkg/net/http/#Request.ParseMultipartForm
				return
			}
		}
		c.formCache = req.PostForm
	}
}

// GetPostFormArray returns a slice of strings for a given form key, plus
// a boolean value whether at least one value exists for the given key.
func (c *Context) GetPostFormArray(key string) (values []string, ok bool) {
	c.initFormCache()
	values, ok = c.formCache[key]
	return
}

func (c *Context) Write(b []byte) (int, error) {
	return c.Writer.Write(b)
}

func (c *Context) reset() {
	c.Writer = &c.writermem
	c.Params = c.Params[:0]
	c.handlers = c.handlers[:0]
	if c.data == nil {
		c.data = make(map[string]any, 8)
	} else {
		clear(c.data)
	}
	c.index = -1
	c.queryCache = nil
	c.formCache = nil
}

func releaseContext(c *Context) {
	c.reset()
	contextPool.Put(c)
}

func acquireContext() *Context {
	return contextPool.Get().(*Context)
}

// getMapFromFormData return a map which satisfies conditions.
// It parses from data with bracket notation like "key[subkey]=value" into a map.
func getMapFromFormData(m map[string][]string, key string) (map[string]string, bool) {
	d := make(map[string]string)
	found := false
	keyLen := len(key)

	for k, v := range m {
		if len(k) < keyLen+3 { // key + "[" + at least one char + "]"
			continue
		}

		if k[:keyLen] != key || k[keyLen] != '[' {
			continue
		}

		if j := strings.IndexByte(k[keyLen+1:], ']'); j > 0 {
			found = true
			d[k[keyLen+1:keyLen+1+j]] = v[0]
		}
	}

	return d, found
}

// Next should be used only inside middleware.
// It executes the pending handlers in the chain inside the calling handler.
func (c *Context) Next() {
	c.index++
	for c.index < int8(len(c.handlers)) {
		c.handlers[c.index](c)
		c.index++
	}
}

// IsAborted returns true if the current context was aborted.
func (c *Context) IsAborted() bool {
	return c.index >= abortIndex
}

// Abort prevents pending handlers from being called. Note that this will not stop the current handler.
// Let's say you have an authorization middleware that validates that the current request is authorized.
// If the authorization fails (ex: the password does not match), call Abort to ensure the remaining handlers
// for this request are not called.
func (c *Context) Abort() {
	c.index = abortIndex
}

// AbortWithStatus calls `Abort()` and writes the headers with the specified status code.
func (c *Context) AbortWithStatus(code int) {
	c.Writer.WriteHeader(code)
	c.Abort()
}

// Status sets the HTTP response code.
func (c *Context) Status(code int) {
	c.Writer.WriteHeader(code)
}

// Header is an intelligent shortcut for c.Writer.Header().Set(key, value).
// It writes a header in the response.
// If value == "", this method removes the header `c.Writer.Header().Del(key)`
func (c *Context) Header(key, value string) {
	if value == "" {
		c.Writer.Header().Del(key)
		return
	}
	c.Writer.Header().Set(key, value)
}

// GetHeader returns value from request headers.
func (c *Context) GetHeader(key string) string {
	return c.requestHeader(key)
}

func (c *Context) requestHeader(key string) string {
	return c.Request.Header.Get(key)
}
func (c *Context) JSON(data any) error {
	c.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	// Security headers
	// Prevents the browser from MIME-sniffing a response away from the declared content-type
	c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
	// Completely disable all forms of caching including back/forward cache
	c.Writer.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0")
	c.Writer.Header().Set("Pragma", "no-cache")
	c.Writer.Header().Set("Expires", "0")
	// Prevents some XSS attacks
	// Note: This header is deprecated and it's recommended to use Content-Security-Policy instead
	// However, some older browsers might still rely on it
	// For more information, see: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-XSS-Protection
	// Here we disable it to avoid conflicts with modern XSS protection mechanisms
	// that are implemented via Content-Security-Policy headers
	c.Writer.Header().Set("X-Xss-Protection", "0")
	c.Writer.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'self'; base-uri 'self'; form-action 'self';")
	// Prevents the page from being displayed in an iframe to avoid clickjacking attacks
	// Use "SAMEORIGIN" to allow iframes from the same origin
	// or "ALLOW-FROM uri" to allow from a specific origin
	// Here we use "DENY" to completely prevent framing
	// For more information, see: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Frame-Options
	c.Writer.Header().Set("X-Frame-Options", "sameorigin")
	// Encode the data to JSON and write to the response
	jsonContent, err := jsonMarshal(data, false)
	if err != nil {
		return err
	}
	_, err = c.Writer.Write(jsonContent)
	return err
}

func (c *Context) AbortWithError(code int, err error) {
	c.AbortWithStatus(code)
	c.Writer.Write([]byte(err.Error()))
}

func (c *Context) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data == nil {
		c.data = make(map[string]any, 8)
	}
	c.data[key] = value
}

func (c *Context) Get(key string) (value any, exists bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, exists = c.data[key]
	return
}

func (c *Context) Data() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.data == nil {
		return map[string]any{}
	}
	return maps.Clone(c.data)
}

func (c *Context) SetMeta(meta Meta) {
	c.Meta = meta
}

func (c *Context) ClientIP() string {
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		ip, _, _ := strings.Cut(xff, ",")
		return strings.TrimSpace(ip)
	}
	if xrip := c.GetHeader("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	if c.Request.RemoteAddr != "" {
		host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err != nil {
			return c.Request.RemoteAddr
		}
		return host
	}
	return ""
}

func (c *Context) Render(view string) error {
	var err error

	if c.GetHeader("X-Pjax") == "true" {
		c.Set("_ViEW_", view)
		c.mu.RLock()
		defer c.mu.RUnlock()
		return c.JSON(c.data)
	}

	// 在开发模式下，优先尝试从 DevHost 获取 root HTML，获取失败则回退到本地模板
	var tpl []byte
	if c.engine.IsDevelopmentMode() {
		tpl = c.fetchDevTemplate()
	} else {
		tpl = s2b(c.engine.rootHTML)
	}

	var ssrContent string
	if c.engine.IsSSRMode() && c.engine.ssr != nil {
		ssrContent, err = c.engine.ssr.RenderComponent(view, c.data)
		if err != nil {
			slog.Error("SSR render error", slog.Any("error", err))
		}
	}

	_, err = executeFunc(tpl, c.engine.startTag, c.engine.endTag, c.Writer, func(w io.Writer, tag string) (int, error) {
		switch tag {
		case "head-meta":
			return w.Write(s2b(c.Meta.ToHTML()))
		case "view":
			return w.Write(s2b(view))
		case "ssr-content":
			return w.Write(s2b(ssrContent))
		case "data-page":
			c.Set("_ViEW_", view)
			c.mu.RLock()
			pageJSON, _ := jsonMarshal(c.data, true)
			c.mu.RUnlock()
			return escapeJSON(w, pageJSON)
		case "version":
			return w.Write(s2b(strconv.FormatInt(c.engine.bootTime, 10)))
		default:
		}
		return 0, nil
	})

	return err
}

// fetchDevTemplate fetches root HTML from the dev server, falling back to the local template on any error.
func (c *Context) fetchDevTemplate() []byte {
	if c.Request == nil {
		return s2b(c.engine.rootHTML)
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, c.engine.devAddr, nil)
	if err != nil {
		return s2b(c.engine.rootHTML)
	}
	client := c.engine.devHTTPClient
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return s2b(c.engine.rootHTML)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return s2b(c.engine.rootHTML)
	}
	tpl, err := io.ReadAll(resp.Body)
	if err != nil {
		return s2b(c.engine.rootHTML)
	}
	return tpl
}

const abortIndex int8 = 63

func b2s(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func s2b(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

type TagFunc func(w io.Writer, tag string) (int, error)

// Use Template.ExecuteFunc for frozen templates.
func executeFunc(template []byte, startTag, endTag string, w io.Writer, f TagFunc) (int64, error) {
	s := template
	a := s2b(startTag)
	b := s2b(endTag)

	var nn int64
	var ni int
	var err error
	for {
		n := bytes.Index(s, a)
		if n < 0 {
			break
		}
		ni, err = w.Write(s[:n])
		nn += int64(ni)
		if err != nil {
			return nn, err
		}

		s = s[n+len(a):]
		n = bytes.Index(s, b)
		if n < 0 {
			// cannot find end tag - just write it to the output.
			ni, _ = w.Write(a)
			nn += int64(ni)
			break
		}

		ni, err = f(w, b2s(s[:n]))
		nn += int64(ni)
		if err != nil {
			return nn, err
		}
		s = s[n+len(b):]
	}
	ni, err = w.Write(s)
	nn += int64(ni)

	return nn, err
}

// HTML escaping.

func escapeJSON(w io.Writer, b []byte) (int, error) {
	last := 0
	n := 0
	var err error
	for i, c := range b {
		var quote []byte
		switch c {
		case '\\', '"':
			quote = []byte{'\\', c}
		case '\b':
			quote = []byte(`\b`)
		case '\f':
			quote = []byte(`\f`)
		case '\n':
			quote = []byte(`\n`)
		case '\r':
			quote = []byte(`\r`)
		case '\t':
			quote = []byte(`\t`)
		case '<':
			quote = []byte(`\u003c`)
		case '>':
			quote = []byte(`\u003e`)
		case '&':
			quote = []byte(`\u0026`)
		default:
			if c < 0x20 {
				// Control characters (U+0000 through U+001F)
				// must be escaped. This is done by replacing the
				// character with the six-character sequence
				// "\u00XX" where XX is the two-digit hexadecimal
				// representation of the character code.
				quote = []byte(`\u00`)
				quote = append(quote, "0123456789abcdef"[c>>4])
				quote = append(quote, "0123456789abcdef"[c&0xF])
			}
			if quote == nil {
				continue
			}
		}
		wn, err := w.Write(b[last:i])
		if err != nil {
			return n, err
		}
		n += wn
		wn, err = w.Write(quote)
		if err != nil {
			return n, err
		}
		n += wn
		last = i + 1
	}
	wn, err := w.Write(b[last:])
	if err != nil {
		return n, err
	}
	n += wn
	return n, nil
}

var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

const maxPooledJSONBufferCapacity = 256 * 1024

func jsonMarshal(v any, escapeHTML bool) ([]byte, error) {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer func() {
		buf.Reset()
		// Avoid retaining very large buffers in the pool.
		if buf.Cap() <= maxPooledJSONBufferCapacity {
			bufPool.Put(buf)
		}
	}()
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(escapeHTML)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	if buf.Len() > 0 && buf.Bytes()[buf.Len()-1] == '\n' {
		buf.Truncate(buf.Len() - 1)
	}
	out := bytes.Clone(buf.Bytes())
	return out, nil
}

/************************************/
/***** GOLANG.ORG/X/NET/CONTEXT *****/
/************************************/

// hasRequestContext returns whether c.Request has Context and fallback.
func (c *Context) hasRequestContext() bool {
	return c.Request != nil
}

// Deadline returns that there is no deadline (ok==false) when c.Request has no Context.
func (c *Context) Deadline() (deadline time.Time, ok bool) {
	if !c.hasRequestContext() {
		return
	}
	return c.Request.Context().Deadline()
}

// Done returns nil (chan which will wait forever) when c.Request has no Context.
func (c *Context) Done() <-chan struct{} {
	if !c.hasRequestContext() {
		return nil
	}
	return c.Request.Context().Done()
}

// Err returns nil when c.Request has no Context.
func (c *Context) Err() error {
	if !c.hasRequestContext() {
		return nil
	}
	return c.Request.Context().Err()
}

// Value returns the value associated with this context for key, or nil
// if no value is associated with key. Successive calls to Value with
// the same key returns the same result.
func (c *Context) Value(key any) any {
	if key == ContextRequestKey {
		return c.Request
	}
	if key == ContextKey {
		return c
	}
	if keyAsString, ok := key.(string); ok {
		if val, exists := c.Get(keyAsString); exists {
			return val
		}
	}
	if !c.hasRequestContext() {
		return nil
	}
	return c.Request.Context().Value(key)
}

func FromContext(ctx context.Context) (*Context, bool) {
	if ctx == nil {
		return nil, false
	}
	if v := ctx.Value(ContextKey); v != nil {
		if c, ok := v.(*Context); ok {
			return c, true
		}
	}
	return nil, false
}

func MustFromContext(ctx context.Context) *Context {
	if ctx == nil {
		panic("inertia: nil context")
	}
	if v := ctx.Value(ContextKey); v != nil {
		if c, ok := v.(*Context); ok {
			return c
		}
	}
	panic("inertia: context is not of type *inertia.Context")
}
