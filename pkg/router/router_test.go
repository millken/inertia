package router

import "testing"

func TestXxx(t *testing.T) {
	r := New[int]()
	r.Add("GET", "/assets*", 1)
	found, handler, params := r.Lookup("GET", "/assets/3")
	if !found {
		t.Fatal("handler not found")
	}
	if handler != 1 {
		t.Fatalf("expected handler 1, got %d", handler)
	}
	if len(params) != 1 {
		t.Fatalf("expected 1 param, got %v", params)
	}
}
