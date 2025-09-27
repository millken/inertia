package quickjs

import (
	"fmt"
	"sync"

	"github.com/buke/quickjs-go"
	"github.com/millken/inertia/ssr"
)

var _ ssr.VM = (*VM)(nil)

type VM struct {
	*ssr.BaseVM
	runtime     *quickjs.Runtime
	context     *quickjs.Context
	initialized bool
	mu          sync.RWMutex
}

func NewVM(options ...ssr.Option) (vm *VM, err error) {
	// Create BaseVM first
	baseVM, err := ssr.NewBaseVM(options...)
	if err != nil {
		return nil, err
	}

	runtime := quickjs.NewRuntime()
	ctx := runtime.NewContext()
	vm = &VM{
		BaseVM:  baseVM,
		runtime: runtime,
		context: ctx,
	}

	defer func() {
		if err != nil {
			vm.Close()
			vm = nil
		}
	}()

	if vm.Options.BundlerJS != "" {
		script := fmt.Sprintf("var module = { exports: {} }; var exports = module.exports; %s;", vm.bundlerJS)
		ret := vm.context.Eval(script)
		if err = ret.ToError(); err != nil {
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
	if vm.context != nil {
		vm.context.Close()
		vm.context = nil
	}
	if vm.runtime != nil {
		vm.runtime.Close()
		vm.runtime = nil
	}
	vm.initialized = false
}

func (vm *VM) RenderTemplate(tpl string, data map[string]any) (string, error) {
	// Try cache first
	cacheKey := vm.GenerateCacheKey(tpl, data)
	if cached, found := vm.TryCache(cacheKey); found {
		return cached, nil
	}

	vm.mu.Lock()
	defer vm.mu.Unlock()

	if !vm.initialized {
		return "", fmt.Errorf("quickjs VM not initialized")
	}

	buf, err := ssr.JsonMarshal(data)
	if err != nil {
		return "", err
	}

	script := fmt.Sprintf(`module.exports.inertiaRenderTemplate(%q, %q)`, tpl, buf)
	ret := vm.context.Eval(script).Await()
	defer ret.Free()
	if ret.IsException() {
		return "", vm.context.Exception()
	}
	if err = ret.ToError(); err != nil {
		return "", err
	}
	vm.context.Loop()
	result := ret.ToString()

	// Cache the result
	vm.SetCache(cacheKey, result)

	return result, nil
}

func (vm *VM) RenderComponent(name string, data map[string]any) (string, error) {
	// Try cache first
	cacheKey := vm.GenerateCacheKey(name, data)
	if cached, found := vm.TryCache(cacheKey); found {
		return cached, nil
	}

	vm.mu.Lock()
	defer vm.mu.Unlock()

	if !vm.initialized {
		return "", fmt.Errorf("quickjs VM not initialized")
	}
	buf, err := ssr.JsonMarshal(data)
	if err != nil {
		return "", err
	}
	script := fmt.Sprintf(`module.exports.inertiaRenderComponent(%q, %q)`, name, buf)
	ret := vm.context.Eval(script).Await()
	defer ret.Free()
	if ret.IsException() {
		return "", vm.context.Exception()
	}
	if err = ret.ToError(); err != nil {
		return "", err
	}
	vm.context.Loop()
	result := ret.ToString()

	// Cache the result
	vm.SetCache(cacheKey, result)

	return result, nil
}
