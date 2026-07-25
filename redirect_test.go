package inertia

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

// pjaxRequest builds a GET carrying the PJAX marker header.
func pjaxRequest(target string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, target, nil)
	r.Header.Set(pjaxHeader, "true")
	return r
}

// TestRedirect covers both representations of a redirect. See Context.Redirect
// for why a PJAX client gets a payload instead of a 3xx.
func TestRedirect(t *testing.T) {
	cases := []struct {
		name string
		pjax bool
		// A PJAX redirect is a payload, not a 3xx: the browser must not act on it
		// itself, so no Location is sent.
		wantStatus   int
		wantLocation string
	}{
		{"pjax", true, http.StatusOK, ""},
		{"plain", false, http.StatusFound, "/admin/login"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e, _ := New()
			e.GET("/admin", func(c *Context) { _ = c.Redirect("/admin/login") })

			r := httptest.NewRequest(http.MethodGet, "/admin", nil)
			if tc.pjax {
				r = pjaxRequest("/admin")
			}
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, r)
			res := rec.Result()

			if res.StatusCode != tc.wantStatus {
				t.Errorf("status = %d, want %d", res.StatusCode, tc.wantStatus)
			}
			if got := res.Header.Get("Location"); got != tc.wantLocation {
				t.Errorf("Location = %q, want %q", got, tc.wantLocation)
			}
			if !tc.pjax {
				return
			}
			var body map[string]any
			if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if got := body["redirect"]; got != "/admin/login" {
				t.Errorf(`body["redirect"] = %v, want "/admin/login"`, got)
			}
		})
	}
}

// TestVaryHeader guards the correctness of serving two representations from one
// URL — see Context.varyOnPjax for what a missing Vary costs.
func TestVaryHeader(t *testing.T) {
	handlers := []struct {
		name string
		h    HandlerFunc
	}{
		{"Render", func(c *Context) { _ = c.Render("Home") }},
		{"Redirect", func(c *Context) { _ = c.Redirect("/elsewhere") }},
	}
	for _, tc := range handlers {
		for _, pjax := range []bool{false, true} {
			name := tc.name + ", plain"
			if pjax {
				name = tc.name + ", pjax"
			}
			t.Run(name, func(t *testing.T) {
				e, _ := New()
				e.GET("/", tc.h)

				r := httptest.NewRequest(http.MethodGet, "/", nil)
				if pjax {
					r = pjaxRequest("/")
				}
				rec := httptest.NewRecorder()
				e.ServeHTTP(rec, r)

				if got := rec.Result().Header.Get("Vary"); got != pjaxHeader {
					t.Errorf("Vary = %q, want %q", got, pjaxHeader)
				}
			})
		}
	}
}

// TestVaryPreservesUpstreamValue pins the reason varyOnPjax uses Add rather than
// Set: a middleware that already declared a Vary must keep it, or a shared cache
// can hand a gzipped body to a client that never asked for one.
func TestVaryPreservesUpstreamValue(t *testing.T) {
	e, _ := New()
	e.Use(func(c *Context) {
		c.Writer.Header().Add("Vary", "Accept-Encoding")
		c.Next()
	})
	e.GET("/admin", func(c *Context) { _ = c.Redirect("/admin/login") })

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, pjaxRequest("/admin"))

	got := rec.Result().Header.Values("Vary")
	want := []string{"Accept-Encoding", pjaxHeader}
	if !slices.Equal(got, want) {
		t.Fatalf("Vary = %v, want %v", got, want)
	}
}

// TestStatusOnlyHandlerReachesWire pins the engine-level header flush: a handler
// that sets a status and writes no body must still send that status rather than
// falling through to net/http's implicit 200.
func TestStatusOnlyHandlerReachesWire(t *testing.T) {
	e, _ := New()
	e.GET("/nc", func(c *Context) { c.Status(http.StatusNoContent) })
	e.GET("/denied", func(c *Context) { c.AbortWithStatus(http.StatusForbidden) })

	cases := map[string]int{"/nc": http.StatusNoContent, "/denied": http.StatusForbidden}
	for path, want := range cases {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if got := rec.Result().StatusCode; got != want {
			t.Errorf("GET %s: status = %d, want %d", path, got, want)
		}
	}
}
