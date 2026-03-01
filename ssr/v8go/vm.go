// Package ssr provides server-side rendering functionality using V8 JavaScript engine.
package v8go

import (
	"fmt"
	"sync"

	"github.com/millken/inertia/ssr"
	"github.com/tommie/v8go"
)

var _ ssr.VM = (*VM)(nil)

type VM struct {
	bundlerJS   string
	iso         *v8go.Isolate
	ctx         *v8go.Context
	initialized bool
	mu          sync.RWMutex
}

func NewVM(options ...ssr.Option) (vm *VM, err error) {
	iso := v8go.NewIsolate()
	ctx := v8go.NewContext(iso)
	vm = &VM{
		iso: iso,
		ctx: ctx,
	}
	defer func() {
		if err != nil {
			vm.Close()
			vm = nil
		}
	}()

	// apply ssr options
	var vmOpts ssr.VMOptions
	for _, option := range options {
		if err = option(&vmOpts); err != nil {
			return nil, err
		}
	}
	vm.bundlerJS = vmOpts.BundlerJS

	if vm.bundlerJS != "" {
		script := fmt.Sprintf("var module = { exports: {} }; var exports = module.exports; %s;", vm.bundlerJS)
		_, err = vm.ctx.RunScript(script, "_")
		if err != nil {
			err = fmt.Errorf("failed to run bundler script: %w", err)
			return nil, err
		}
		vm.initialized = true
	}

	return vm, nil
}

// Close releases VM resources.
func (vm *VM) Close() {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	if vm.ctx != nil {
		vm.ctx = nil
	}
	if vm.iso != nil {
		vm.iso.Dispose()
		vm.iso = nil
	}
	vm.initialized = false
}

func (vm *VM) RenderTemplate(tpl string, data map[string]any) (string, error) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if !vm.initialized {
		return "", fmt.Errorf("v8go VM not initialized")
	}
	buf, err := ssr.JSONMarshal(data)
	if err != nil {
		return "", err
	}
	script := fmt.Sprintf(`module.exports.inertiaRenderTemplate(%q, %q)`, tpl, buf)
	val, err := vm.ctx.RunScript(script, "_")
	if err != nil {
		return "", err
	}
	return val.String(), nil
}

func (vm *VM) RenderComponent(name string, data map[string]any) (string, error) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if !vm.initialized {
		return "", fmt.Errorf("v8go VM not initialized")
	}
	buf, err := ssr.JSONMarshal(data)
	if err != nil {
		return "", err
	}
	script := fmt.Sprintf(`module.exports.inertiaRenderComponent(%q, %q)`, name, buf)
	val, err := vm.ctx.RunScript(script, "_")
	if err != nil {
		return "", err
	}
	return val.String(), nil
}
