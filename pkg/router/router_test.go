package router

import "testing"

func TestRoute(t *testing.T) {
	r := New[int]()
	tests := []struct {
		method  string
		pattern string
		handler int
		path    string
		found   bool
		params  []Parameter
	}{
		{"GET", "/", 0, "/", true, nil},
		{"GET", "/panic", 1, "/panic", true, nil},
		{"GET", "/assets/*", 2, "/assets/3", true, []Parameter{{Key: "", Value: "3"}}},
		{"GET", "/user/:id", 3, "/user/123", true, []Parameter{{Key: "id", Value: "123"}}},
	}
	for _, test := range tests {
		r.Add(test.method, test.pattern, test.handler)
		found, handler, params := r.Lookup(test.method, test.path)
		if found != test.found {
			t.Fatalf("expected found %v, got %v for path %s", test.found, found, test.path)
		}
		if found {
			if handler != test.handler {
				t.Fatalf("expected handler %d, got %d for path %s", test.handler, handler, test.path)
			}
			if len(params) != len(test.params) {
				t.Fatalf("expected %d params, got %d for path %s", len(test.params), len(params), test.path)
			}
			for i, param := range params {
				if param != test.params[i] {
					t.Fatalf("expected param %v, got %v for path %s", test.params[i], param, test.path)
				}
			}
		}
	}
}

func TestParameterRoute(t *testing.T) {
	r := New[int]()
	r.Add("GET", "/user/:id", 2)
	found, handler, params := r.Lookup("GET", "/user/123")
	if !found {
		t.Fatal("handler not found")
	}
	if handler != 2 {
		t.Fatalf("expected handler 2, got %d", handler)
	}
	if len(params) != 1 {
		t.Fatalf("expected 1 param, got %v", params)
	}
	if params[0].Key != "id" || params[0].Value != "123" {
		t.Fatalf("expected param {id: 123}, got %v", params[0])
	}
}

func TestStaticRoute(t *testing.T) {
	r := New[int]()
	r.Add("GET", "/static/path", 3)
	found, handler, params := r.Lookup("GET", "/static/path")
	if !found {
		t.Fatal("handler not found")
	}
	if handler != 3 {
		t.Fatalf("expected handler 3, got %d", handler)
	}
	if len(params) != 0 {
		t.Fatalf("expected no params, got %v", params)
	}
}

func BenchmarkRouter(b *testing.B) {
	r := New[int]()
	r.Add("GET", "/static/path", 1)
	r.Add("GET", "/user/:id", 2)
	r.Add("GET", "/assets/*", 3)

	paths := []string{
		"/static/path",
		"/user/123",
		"/assets/3",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range paths {
			r.LookupNoAlloc("GET", path, func(s1, s2 string) {})
		}
	}
}
