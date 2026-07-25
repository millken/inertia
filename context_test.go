package inertia

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSetContext pins the read/write split: SetContext stores into the request
// context, not the rendered data map, yet the value is still reachable through
// Value (which falls through to the request context). See Context.SetContext.
func TestSetContext(t *testing.T) {
	e, _ := New()
	c := &Context{engine: e, Request: httptest.NewRequest(http.MethodGet, "/", nil)}

	c.SetContext("user", "alice")

	// Not in the data map, so it can never be serialized as a prop.
	if _, ok := c.Get("user"); ok {
		t.Fatal(`Get("user") succeeded: SetContext leaked into the data map`)
	}
	// Reachable through Value via the request-context fallback.
	if got := c.Value("user"); got != "alice" {
		t.Errorf(`Value("user") = %v, want "alice"`, got)
	}
}

// TestRenderDoesNotLeakRequestContext is the security pin for the data/context
// split: props set with Set travel to the client, but anything stored via
// SetContext (sessions, auth principals) must never appear in the rendered
// payload. It also pins that the view still ships under viewKey.
func TestRenderDoesNotLeakRequestContext(t *testing.T) {
	const secret = "do-not-leak"
	secretKey := struct{}{} // typed key, as SetContext's doc recommends

	e, _ := New()
	e.GET("/", func(c *Context) {
		c.SetContext(secretKey, secret)
		c.Set("greeting", "hi")
		_ = c.Render("Dashboard")
	})

	// The PJAX body is the payload JSON verbatim, so any leakage is visible.
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, pjaxRequest("/"))

	var body map[string]any
	if err := json.NewDecoder(rec.Result().Body).Decode(&body); err != nil {
		t.Fatalf("decode pjax body: %v", err)
	}
	if body["greeting"] != "hi" {
		t.Errorf(`greeting = %v, want "hi" (props must be sent)`, body["greeting"])
	}
	if got := body[viewKey]; got != "Dashboard" {
		t.Errorf(`%s = %v, want "Dashboard" (view must ship under viewKey)`, viewKey, got)
	}
	for k, v := range body {
		if v == secret {
			t.Fatalf("request-context value leaked into rendered payload under key %q", k)
		}
	}
}
