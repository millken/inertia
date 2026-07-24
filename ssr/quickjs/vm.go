package quickjs

import (
	"context"
	"fmt"

	"github.com/buke/quickjs-go"
	"github.com/millken/inertia/ssr"
)

var _ ssr.VM = (*VM)(nil)

type renderRequest struct {
	script string
	result chan<- renderResult
}

type renderResult struct {
	value string
	err   error
}

type VM struct {
	*ssr.BaseVM
	reqCh  chan renderRequest
	doneCh chan struct{}
}

func NewVM(options ...ssr.Option) (*VM, error) {
	baseVM, err := ssr.NewBaseVM(options...)
	if err != nil {
		return nil, err
	}

	vm := &VM{
		BaseVM: baseVM,
		reqCh:  make(chan renderRequest),
		doneCh: make(chan struct{}),
	}

	initErr := make(chan error, 1)
	go vm.run(initErr)
	if err := <-initErr; err != nil {
		return nil, err
	}
	return vm, nil
}

func (vm *VM) run(initErr chan<- error) {
	runtime := quickjs.NewRuntime()
	defer func() {
		runtime.Close()
		close(vm.doneCh)
	}()

	ctx := runtime.NewContext()
	defer ctx.Close()

	if vm.Options.BundlerJS != "" {
		script := fmt.Sprintf("var module = { exports: {} }; var exports = module.exports; %s;", vm.Options.BundlerJS)
		ret := ctx.Eval(script)
		if ret == nil {
			initErr <- fmt.Errorf("failed to run bundler script: context invalid")
			return
		}
		if err := ret.ToError(); err != nil {
			ret.Free()
			initErr <- fmt.Errorf("failed to run bundler script: %w", err)
			return
		}
		ret.Free()
	}

	initErr <- nil

	for req := range vm.reqCh {
		ret := ctx.Eval(req.script)
		if ret == nil {
			req.result <- renderResult{err: fmt.Errorf("quickjs: context invalid")}
			continue
		}
		if ret.IsException() {
			ex := ctx.Exception()
			ret.Free()
			req.result <- renderResult{err: ex}
			continue
		}

		awaited := ret.Await()
		ret.Free()

		if awaited == nil {
			req.result <- renderResult{err: fmt.Errorf("quickjs: Await returned nil")}
			continue
		}
		if awaited.IsException() {
			ex := ctx.Exception()
			awaited.Free()
			req.result <- renderResult{err: ex}
			continue
		}
		if err := awaited.ToError(); err != nil {
			awaited.Free()
			req.result <- renderResult{err: err}
			continue
		}
		result := awaited.ToString()
		awaited.Free()
		req.result <- renderResult{value: result}
	}
}

func (vm *VM) Close() {
	close(vm.reqCh)
	<-vm.doneCh
}

// withTimeout returns ctx unchanged when it already has a deadline; otherwise it
// derives a child bounded by the VM's configured Timeout. Note: a blocked C eval
// cannot be hard-interrupted, so on timeout the caller unblocks but the worker
// goroutine may finish the in-flight eval late (a genuinely hung worker poisons
// that VM until Close; subsequent renders then fail fast on their own timeout).
func (vm *VM) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	if vm.Options.Timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, vm.Options.Timeout)
}

func (vm *VM) RenderTemplate(ctx context.Context, tpl string, data map[string]any) (string, error) {
	cacheKey := vm.GenerateCacheKey(tpl, data)
	if cached, found := vm.TryCache(cacheKey); found {
		return cached, nil
	}

	buf, err := ssr.JSONMarshal(data)
	if err != nil {
		return "", err
	}

	ctx, cancel := vm.withTimeout(ctx)
	defer cancel()
	script := fmt.Sprintf(`module.exports.inertiaRenderTemplate(%q, %q)`, tpl, buf)
	result, err := vm.dispatch(ctx, script)
	if err != nil {
		return "", err
	}
	vm.SetCache(cacheKey, result)
	return result, nil
}

func (vm *VM) RenderComponent(ctx context.Context, name string, data map[string]any) (string, error) {
	cacheKey := vm.GenerateCacheKey(name, data)
	if cached, found := vm.TryCache(cacheKey); found {
		return cached, nil
	}

	buf, err := ssr.JSONMarshal(data)
	if err != nil {
		return "", err
	}

	ctx, cancel := vm.withTimeout(ctx)
	defer cancel()
	script := fmt.Sprintf(`module.exports.inertiaRenderComponent(%q, %q)`, name, buf)
	result, err := vm.dispatch(ctx, script)
	if err != nil {
		return "", err
	}
	vm.SetCache(cacheKey, result)
	return result, nil
}

func (vm *VM) dispatch(ctx context.Context, script string) (string, error) {
	ch := make(chan renderResult, 1)
	select {
	case vm.reqCh <- renderRequest{script: script, result: ch}:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	select {
	case res := <-ch:
		return res.value, res.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}
