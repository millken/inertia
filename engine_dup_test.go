package inertia

import (
	"strings"
	"testing"
)

func TestEngineRegistrationErrorOnDuplicate(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	h := func(c *Context) {}
	e.GET("/blog/post", h)
	e.GET("/blog/post", h) // duplicate registration

	regErr := e.RegistrationError()
	if regErr == nil {
		t.Fatal("want a registration error after duplicate GET, got nil")
	}
	if !strings.Contains(regErr.Error(), "/blog/post") || !strings.Contains(regErr.Error(), "GET") {
		t.Fatalf("error should name the method and route, got %v", regErr)
	}
}

func TestEngineNoRegistrationErrorForDistinctRoutes(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	h := func(c *Context) {}
	e.GET("/blog/post", h)
	e.POST("/blog/post", h)
	e.GET("/blog/post/:id", h)

	if regErr := e.RegistrationError(); regErr != nil {
		t.Fatalf("want no registration error for distinct routes, got %v", regErr)
	}
}

func TestEngineServeReturnsRegistrationError(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	h := func(c *Context) {}
	e.GET("/dup", h)
	e.GET("/dup", h)

	// Serve must fail fast on a registration error, before binding a listener,
	// so this returns immediately instead of blocking.
	if serveErr := e.Serve(); serveErr == nil {
		t.Fatal("want Serve to return the registration error, got nil")
	}
}
