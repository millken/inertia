package inertia

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"unsafe"
)

type Context struct {
	writermem responseWriter
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

	data  map[string]any
	index int8
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
	c.data = make(map[string]any, 8)
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
	c.Writer.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(c).Encode(data)
}

func (c *Context) AbortWithError(code int, err error) {
	c.AbortWithStatus(code)
	c.Writer.Write([]byte(err.Error()))
}

func (c *Context) Set(key string, value any) {
	c.data[key] = value
}

func (c *Context) Render(view string) error {
	c.data["_ViEW_"] = view

	if c.GetHeader("X-Pjax") == "true" {
		return c.JSON(c.data)
	}

	// 在开发模式下，优先尝试从 DevHost 获取 root HTML，获取失败则回退到本地模板
	var tpl []byte
	if c.engine.DevMode() {
		body, err := http.Get(c.engine.devAddr)
		if err == nil && body.StatusCode == http.StatusOK {
			defer body.Body.Close()
			tpl, err = io.ReadAll(body.Body)
			if err != nil {
				tpl = s2b(c.engine.rootHTML)
			}
		} else {
			tpl = s2b(c.engine.rootHTML)
		}
	} else {
		tpl = s2b(c.engine.rootHTML)
	}

	_, err := executeFunc(tpl, c.engine.startTag, c.engine.endTag, c.Writer, func(w io.Writer, tag string) (int, error) {
		switch tag {
		case "view":
			return w.Write(s2b(view))
		case "data-page":
			pageJSON, _ := json.Marshal(c.data)
			return htmlEscape(w, pageJSON)
		default:
		}
		return 0, nil
	})

	return err
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

var (
	htmlQuot = []byte("&#34;") // shorter than "&quot;"
	htmlApos = []byte("&#39;") // shorter than "&apos;" and apos was not in HTML until HTML5
	htmlAmp  = []byte("&amp;")
	htmlLt   = []byte("&lt;")
	htmlGt   = []byte("&gt;")
	htmlNull = []byte("\uFFFD")
)

// HTMLEscape writes to w the escaped HTML equivalent of the plain text data b.
func htmlEscape(w io.Writer, b []byte) (int, error) {
	last := 0
	n := 0
	var err error
	for i, c := range b {
		var html []byte
		switch c {
		case '\000':
			html = htmlNull
		case '"':
			html = htmlQuot
		case '\'':
			html = htmlApos
		case '&':
			html = htmlAmp
		case '<':
			html = htmlLt
		case '>':
			html = htmlGt
		default:
			continue
		}
		wn, err := w.Write(b[last:i])
		if err != nil {
			return n, err
		}
		n += wn
		wn, err = w.Write(html)
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
