package ssr

import (
	"bytes"
	"encoding/json"
	"os"
	"sync"
)

type VM interface {
	RenderTemplate(tpl string, data map[string]any) (string, error)
	RenderComponent(name string, data map[string]any) (string, error)
	Close()
}

type VMOptions struct {
	BundlerJS string // JavaScript code that exports inertiaRenderTemplate and inertiaRenderComponent functions
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
