package ssr

import (
	"testing"
	"time"
)

func TestBaseVMDefaults(t *testing.T) {
	b, err := NewBaseVM(WithDefaultCache(2))
	if err != nil {
		t.Fatal(err)
	}
	if !b.Options.CacheEnabled {
		t.Error("expected cache enabled")
	}
	if b.Options.Timeout != 10*time.Second {
		t.Errorf("default timeout = %v, want 10s", b.Options.Timeout)
	}
	if _, ok := b.TryCache("missing"); ok {
		t.Error("expected miss on empty cache")
	}
	b.SetCache("k", "v")
	if got, ok := b.TryCache("k"); !ok || got != "v" {
		t.Errorf("TryCache(k) = %q,%v, want v,true", got, ok)
	}
}

func TestBaseVMTimeoutOption(t *testing.T) {
	b, err := NewBaseVM(WithTimeout(3 * time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if b.Options.Timeout != 3*time.Second {
		t.Errorf("timeout = %v, want 3s", b.Options.Timeout)
	}
}

func TestGenerateCacheKey(t *testing.T) {
	b, _ := NewBaseVM()
	k1 := b.GenerateCacheKey("Home", map[string]any{"msg": "hi"})
	k2 := b.GenerateCacheKey("Home", map[string]any{"msg": "hi"})
	k3 := b.GenerateCacheKey("Home", map[string]any{"msg": "bye"})
	if k1 != k2 {
		t.Error("same input should produce same key")
	}
	if k1 == k3 {
		t.Error("different data should produce different key")
	}
}

func TestLRUEviction(t *testing.T) {
	c := newLRUCache(2)
	c.Set("a", "1")
	c.Set("b", "2")
	c.Set("c", "3") // evicts least-recently-used "a"
	if _, ok := c.Get("a"); ok {
		t.Error("a should have been evicted")
	}
	if _, ok := c.Get("b"); !ok {
		t.Error("b should be present")
	}
	if _, ok := c.Get("c"); !ok {
		t.Error("c should be present")
	}
}

func TestJSONMarshal(t *testing.T) {
	out, err := JSONMarshal(map[string]any{"msg": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `{"msg":"hi"}` {
		t.Errorf("JSONMarshal = %s", out)
	}
}
