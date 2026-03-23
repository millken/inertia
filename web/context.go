package web

import (
	"maps"
	"strings"
)

const abortIndex = int(^uint(0) >> 1)

type Context struct {
	Request      Request
	Response     Response
	server       *Server
	routeHandler HandlerFunc
	nextIndex    int
	data         map[string]any
}

func (c *Context) Query(key string) string { return c.Request.Query().Param(key) }

func (c *Context) DefaultQuery(key, defaultValue string) string {
	if value := c.Query(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *Context) GetQuery(key string) (string, bool) {
	value := c.Query(key)
	if value == "" {
		return "", false
	}
	return value, true
}

func (c *Context) PostForm(key string) string { return c.Request.PostForm(key) }

func (c *Context) DefaultPostForm(key, defaultValue string) string {
	return c.Request.DefaultPostForm(key, defaultValue)
}

func (c *Context) GetPostForm(key string) (string, bool) { return c.Request.GetPostForm(key) }
func (c *Context) IsAborted() bool { return c.nextIndex >= abortIndex }
func (c *Context) Abort() { c.nextIndex = abortIndex }

func (c *Context) AbortWithStatus(code int) {
	c.Status(code)
	c.Abort()
}

func (c *Context) AbortWithError(code int, err error) {
	c.AbortWithStatus(code)
	if err != nil {
		_, _ = c.Response.WriteString(err.Error())
	}
}

func (c *Context) GetHeader(key string) string { return c.Request.Header(key) }

func (c *Context) JSON(data any) error {
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	c.Header("X-Xss-Protection", "0")
	c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'self'; base-uri 'self'; form-action 'self';")
	c.Header("X-Frame-Options", "sameorigin")
	jsonContent, err := jsonMarshal(data, false)
	if err != nil {
		return err
	}
	_, err = c.Response.Write(jsonContent)
	return err
}

func (c *Context) Set(key string, value any) {
	if c.data == nil {
		c.data = make(map[string]any, 8)
	}
	c.data[key] = value
}

func (c *Context) Get(key string) (value any, exists bool) {
	value, exists = c.data[key]
	return
}

func (c *Context) Data() map[string]any {
	if c.data == nil {
		return map[string]any{}
	}
	return maps.Clone(c.data)
}

func (c *Context) ClientIP() string {
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		ip, _, _ := strings.Cut(xff, ",")
		return strings.TrimSpace(ip)
	}
	if xrip := c.GetHeader("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	return c.Request.RemoteAddr()
}

func (c *Context) Next() error {
	if c.IsAborted() {
		return nil
	}
	if c.nextIndex < len(c.server.middleware) {
		handler := c.server.middleware[c.nextIndex]
		c.nextIndex++
		return handler(c)
	}
	if c.nextIndex == len(c.server.middleware) {
		c.nextIndex++
		if c.routeHandler != nil {
			return c.routeHandler(c)
		}
	}
	return nil
}

func (c *Context) Status(status int) { c.Response.WriteHeader(status) }

func (c *Context) Header(key, value string) {
	if value == "" {
		c.Response.DeleteHeader(key)
		return
	}
	c.Response.SetHeader(key, value)
}

func (c *Context) String(body string) error {
	_, err := c.Response.WriteString(body)
	return err
}

func (c *Context) Bytes(body []byte) error {
	_, err := c.Response.Write(body)
	return err
}