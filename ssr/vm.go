// Package ssr provides server-side rendering functionality using V8 JavaScript engine.
package ssr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"rogchap.com/v8go"
)

var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

func jsonMarshal(v any) ([]byte, error) {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)
	if err := json.NewEncoder(buf).Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type VMOptions func(*VM) error

type VM struct {
	bundlerJS   string
	isolate     *v8go.Isolate
	context     *v8go.Context
	initialized bool
	mu          sync.RWMutex
}

func WithBundler(js string) VMOptions {
	return func(r *VM) error {
		r.bundlerJS = js
		return nil
	}
}

func WithBundlerFile(path string) VMOptions {
	return func(r *VM) error {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		r.bundlerJS = string(data)
		return nil
	}
}

func NewVM(options ...VMOptions) (*VM, error) {
	isolate := v8go.NewIsolate()
	global := v8go.NewObjectTemplate(isolate)

	// // Fetch support
	// if err := fetch.InjectTo(isolate, global); err != nil {
	// 	isolate.TerminateExecution()
	// 	isolate.Dispose()
	// 	return nil, err
	// }
	// // setTimeout & setInterval support
	// if err := timers.InjectTo(isolate, global); err != nil {
	// 	isolate.TerminateExecution()
	// 	isolate.Dispose()
	// 	return nil, err
	// }
	// Create the context
	context := v8go.NewContext(isolate, global)
	// // URL support
	// if err := url.InjectTo(context); err != nil {
	// 	context.Close()
	// 	isolate.TerminateExecution()
	// 	isolate.Dispose()
	// 	return nil, err
	// }
	// // Console support
	// if err := console.InjectTo(context, console.WithOutput(os.Stdout)); err != nil {
	// 	context.Close()
	// 	isolate.TerminateExecution()
	// 	isolate.Dispose()
	// 	return nil, err
	// }
	//structuredClone support
	// 	context.RunScript(`globalThis.structuredClone = function (value) {
	// 		return JSON.parse(JSON.stringify(value));
	// 	};`, "structuredClone.js")
	// 	context.RunScript(`
	// if (typeof console !== 'undefined' && typeof console.error !== 'function') {
	//   console.error = console.log;
	// }
	// `, "console-error-polyfill.js")

	vm := &VM{
		isolate: isolate,
		context: context,
	}

	for _, option := range options {
		if err := option(vm); err != nil {
			vm.Close()
			return nil, err
		}
	}

	// 预先初始化 bundlerJS，只执行一次
	if vm.bundlerJS != "" {
		_, err := vm.context.RunScript(vm.bundlerJS, "bundler.js")
		if err != nil {
			vm.Close()
			return nil, fmt.Errorf("failed to initialize bundler: %w", err)
		}
		vm.initialized = true
	}

	return vm, nil
}

// Close releases VM resources.
func (r *VM) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.context != nil {
		r.context.Close()
		r.context = nil
	}
	if r.isolate != nil {
		r.isolate.TerminateExecution()
		r.isolate.Dispose()
		r.isolate = nil
	}
	r.initialized = false
}

func (r *VM) RenderTemplate(tpl string, data map[string]any) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.initialized {
		return "", fmt.Errorf("VM not initialized")
	}

	props, err := jsonMarshal(data)
	if err != nil {
		return "", err
	}

	// 只执行渲染调用，不重新加载 bundlerJS
	script := fmt.Sprintf(`__inertia__.renderTemplate(%q, JSON.parse(%q))`, tpl, props)
	value, err := r.evaluate(script)
	if err != nil {
		return "", err
	}
	return value.String(), nil
}

func (r *VM) RenderComponent(name string, data map[string]any) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.initialized {
		return "", fmt.Errorf("VM not initialized")
	}

	props, err := jsonMarshal(data)
	if err != nil {
		return "", err
	}

	script := fmt.Sprintf(`__inertia__.renderComponent(%q, JSON.parse(%q))`, name, props)
	value, err := r.evaluate(script)
	if err != nil {
		return "", err
	}
	return value.String(), nil
}

func (r *VM) RenderPage(name string, data map[string]any) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.initialized {
		return "", fmt.Errorf("VM not initialized")
	}

	props, err := jsonMarshal(data)
	if err != nil {
		return "", err
	}

	script := fmt.Sprintf(`__inertia__.renderPage(%q, JSON.parse(%q))`, name, props)
	value, err := r.evaluate(script)
	if err != nil {
		return "", err
	}
	return value.String(), nil
}

// evaluate runs the given script in the VM context and returns the result.
// It also handles promises by waiting for their resolution.

func (r *VM) evaluate(script string) (*v8go.Value, error) {
	value, err := r.context.RunScript(script, "render.js")
	if err != nil {
		return nil, err
	}
	// Handle promises
	if value.IsPromise() {
		prom, err := value.AsPromise()
		if err != nil {
			return nil, err
		}
		// TODO: this could run forever
		for prom.State() == v8go.Pending {
			continue
		}
		return prom.Result(), nil
	}
	return value, nil
}
