package router

import (
	"errors"
	"testing"
)

func noop(int) {}

func TestRouterAddRejectsExactDuplicate(t *testing.T) {
	r := New[Fn]()
	if err := r.Add("GET", "/blog/post", noop); err != nil {
		t.Fatalf("first add: %v", err)
	}
	if err := r.Add("GET", "/blog/post", noop); !errors.Is(err, ErrDuplicateRoute) {
		t.Fatalf("want ErrDuplicateRoute on duplicate, got %v", err)
	}
}

func TestRouterAddAllowsDistinctRoutes(t *testing.T) {
	r := New[Fn]()
	cases := []struct{ method, path string }{
		{"GET", "/blog/post"},  // same path,
		{"POST", "/blog/post"}, // different method → OK
		{"GET", "/blog/tag"},   // different path → OK
	}
	for _, c := range cases {
		if err := r.Add(c.method, c.path, noop); err != nil {
			t.Fatalf("%s %s should be allowed, got %v", c.method, c.path, err)
		}
	}
}

func TestRouterAddRejectsParamNameConflict(t *testing.T) {
	r := New[Fn]()
	if err := r.Add("GET", "/blog/post/:id", noop); err != nil {
		t.Fatalf("first add: %v", err)
	}
	// Different param name at the same position is the SAME shape to the radix
	// tree — a real conflict that must be rejected, not silently overwritten.
	if err := r.Add("GET", "/blog/post/:slug", noop); !errors.Is(err, ErrDuplicateRoute) {
		t.Fatalf("want ErrDuplicateRoute on param-name conflict, got %v", err)
	}
}

func TestRouterAddAllowsStaticAndParamAtSamePosition(t *testing.T) {
	r := New[Fn]()
	if err := r.Add("GET", "/blog/post/new", noop); err != nil {
		t.Fatalf("static add: %v", err)
	}
	// A static segment and a param segment coexist in the radix tree.
	if err := r.Add("GET", "/blog/post/:id", noop); err != nil {
		t.Fatalf("static and param at same position must coexist, got %v", err)
	}
}
