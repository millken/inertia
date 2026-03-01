package goja

import (
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/dop251/goja"
	"github.com/millken/inertia/ssr"
)

var _ ssr.VM = (*VM)(nil)

type VM struct {
	bundlerJS       string
	runtime         *goja.Runtime
	renderComponent goja.Callable
	renderTemplate  goja.Callable
	initialized     bool
	mu              sync.RWMutex
}

func NewVM(options ...ssr.Option) (vm *VM, err error) {
	gojaVM := goja.New()
	vm = &VM{
		runtime: gojaVM,
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

	script := fmt.Sprintf(`var module = { exports: {} }; var exports = module.exports; %s; `, vm.bundlerJS)
	if vm.bundlerJS != "" {
		_, err = vm.runtime.RunString(script)
		if err != nil {
			err = fmt.Errorf("failed to run bundler script: %w", err)
		}
		vm.initialized = true
	}
	gojaVM.Set("console", map[string]interface{}{
		"log":   os.Stdout.WriteString,
		"error": os.Stderr.WriteString,
	})
	// 获取 module.exports 对象
	module := gojaVM.Get("module").ToObject(gojaVM)
	exports := module.Get("exports").ToObject(gojaVM)

	// 检查导出的键名（调试用）
	slog.Debug("VM exports", "keys", exports.Keys())
	var ok bool
	vm.renderComponent, ok = goja.AssertFunction(exports.Get("inertiaRenderComponent"))
	if !ok {
		err = fmt.Errorf("inertiaRenderComponent is not a function")
		return
	}
	vm.renderTemplate, ok = goja.AssertFunction(exports.Get("inertiaRenderTemplate"))
	if !ok {
		err = fmt.Errorf("inertiaRenderTemplate is not a function")
		return
	}

	return
}

// Close releases VM resources.
func (vm *VM) Close() {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if vm.runtime != nil {
		vm.runtime = nil
	}
	vm.initialized = false
}

func (vm *VM) RenderTemplate(tpl string, data map[string]any) (string, error) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if !vm.initialized {
		return "", fmt.Errorf("VM not initialized")
	}
	buf, _ := ssr.JSONMarshal(data)
	value, err := vm.renderTemplate(goja.Undefined(), vm.runtime.ToValue(tpl), vm.runtime.ToValue(string(buf)))
	if err != nil {
		return "", err
	}
	result, err := vm.promiseValue(value)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(result), nil
}

func (vm *VM) RenderComponent(name string, data map[string]any) (string, error) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if !vm.initialized {
		return "", fmt.Errorf("VM not initialized")
	}
	buf, _ := ssr.JSONMarshal(data)
	value, err := vm.renderComponent(goja.Undefined(), vm.runtime.ToValue(name), vm.runtime.ToValue(string(buf)))
	if err != nil {
		return "", err
	}
	result, err := vm.promiseValue(value)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(result), nil
}

func (vm *VM) promiseValue(value goja.Value) (any, error) {
	if p, ok := value.Export().(*goja.Promise); ok {
		switch p.State() {
		case goja.PromiseStateRejected:
			return nil, fmt.Errorf("promise rejected: %v", p.Result())
		case goja.PromiseStateFulfilled:
			return vm.runtime.ToValue(p.Result()), nil
		default:
			return nil, fmt.Errorf("promise is still pending")
		}
	}
	return value, nil
}
