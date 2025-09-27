package ssr

import (
	"bytes"
	"container/list"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"os"
	"sync"
)

type VM interface {
	RenderTemplate(tpl string, data map[string]any) (string, error)
	RenderComponent(name string, data map[string]any) (string, error)
	Close()
}

// Cacher defines a minimal interface for optional SSR result caching.
type Cacher interface {
	Get(key string) (string, bool)
	Set(key string, value string)
}

// VMOptions holds options for creating a VM. It now supports optional caching.
type VMOptions struct {
	BundlerJS     string // JavaScript code that exports inertiaRenderTemplate and inertiaRenderComponent functions
	CacheEnabled  bool   // whether caching is enabled
	CacheCapacity int    // LRU capacity when using the built-in LRU cache
	Cache         Cacher // optional custom cache implementation
}

// Option is a function that configures a VMOptions.
type Option func(*VMOptions) error

// WithBundlerJS sets the JavaScript code that exports inertiaRenderTemplate and inertiaRenderComponent functions.
func WithBundlerJS(bundlerJS string) Option {
	return func(opts *VMOptions) error {
		opts.BundlerJS = bundlerJS
		return nil
	}
}

func WithBundlerFile(path string) Option {
	return func(opts *VMOptions) error {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		opts.BundlerJS = string(data)
		return nil
	}
}

// WithDefaultCache enables a built-in LRU cache with the provided capacity. If capacity <= 0 a default is used.
func WithDefaultCache(capacity ...int) Option {
	return func(opts *VMOptions) error {
		cap := 0
		if len(capacity) > 0 {
			cap = capacity[0]
		}
		if cap <= 0 {
			cap = 128
		}
		opts.CacheEnabled = true
		opts.CacheCapacity = cap
		opts.Cache = newLRUCache(cap)
		return nil
	}
}

// WithCacher sets a custom Cache implementation to be used by the VM.
func WithCacher(c Cacher) Option {
	return func(opts *VMOptions) error {
		if c == nil {
			return nil
		}
		opts.CacheEnabled = true
		opts.Cache = c
		return nil
	}
}

// BaseVM provides common functionality for VM implementations including caching support.
type BaseVM struct {
	Options *VMOptions
}

// NewBaseVM creates a new BaseVM with the given options.
func NewBaseVM(opts ...Option) (*BaseVM, error) {
	options := &VMOptions{}
	for _, opt := range opts {
		if err := opt(options); err != nil {
			return nil, err
		}
	}
	return &BaseVM{Options: options}, nil
}

// TryCache attempts to get a cached result. If found, returns the result and true.
// If not found, returns empty string and false.
func (b *BaseVM) TryCache(cacheKey string) (string, bool) {
	if !b.Options.CacheEnabled || b.Options.Cache == nil {
		return "", false
	}
	return b.Options.Cache.Get(cacheKey)
}

// SetCache stores a result in the cache if caching is enabled.
func (b *BaseVM) SetCache(cacheKey, result string) {
	if b.Options.CacheEnabled && b.Options.Cache != nil {
		b.Options.Cache.Set(cacheKey, result)
	}
}

// GenerateCacheKey creates a cache key from template/component name and data.
func (b *BaseVM) GenerateCacheKey(name string, data map[string]any) string {
	dataBytes, _ := JsonMarshal(data)
	hash := md5.Sum(dataBytes)
	return name + ":" + hex.EncodeToString(hash[:])
}

var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// JsonMarshal marshals v to JSON using a buffer pool to reduce allocations.
func JsonMarshal(v any) ([]byte, error) {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)
	if err := json.NewEncoder(buf).Encode(v); err != nil {
		return nil, err
	}
	if buf.Len() > 0 && buf.Bytes()[buf.Len()-1] == '\n' {
		buf.Truncate(buf.Len() - 1)
	}
	return buf.Bytes(), nil
}

// lruCache is a simple thread-safe LRU cache implementing Cache.
type lruCache struct {
	mu  sync.Mutex
	cap int
	ll  *list.List
	m   map[string]*list.Element
}

type entry struct {
	key   string
	value string
}

func newLRUCache(capacity int) Cacher {
	if capacity <= 0 {
		capacity = 128
	}
	return &lruCache{
		cap: capacity,
		ll:  list.New(),
		m:   make(map[string]*list.Element, capacity),
	}
}

func (c *lruCache) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ele, ok := c.m[key]; ok {
		c.ll.MoveToFront(ele)
		return ele.Value.(*entry).value, true
	}
	return "", false
}

func (c *lruCache) Set(key string, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ele, ok := c.m[key]; ok {
		ele.Value.(*entry).value = value
		c.ll.MoveToFront(ele)
		return
	}
	ele := c.ll.PushFront(&entry{key: key, value: value})
	c.m[key] = ele
	if c.ll.Len() > c.cap {
		tail := c.ll.Back()
		if tail != nil {
			c.ll.Remove(tail)
			en := tail.Value.(*entry)
			delete(c.m, en.key)
		}
	}
}
