package js

import (
	"context"

	"go.kuoruan.net/v8go-polyfills/console"
	"go.kuoruan.net/v8go-polyfills/fetch"
	"go.kuoruan.net/v8go-polyfills/timers"
	"go.kuoruan.net/v8go-polyfills/url"
	"rogchap.com/v8go"
)

type Value = v8go.Value
type Error = v8go.JSError

func load(c *Console) (*v8go.Isolate, *v8go.Context, error) {
	isolate := v8go.NewIsolate()
	global := v8go.NewObjectTemplate(isolate)
	// Fetch support
	if err := fetch.InjectTo(isolate, global); err != nil {
		isolate.TerminateExecution()
		isolate.Dispose()
		return nil, nil, err
	}
	// setTimeout & setInterval support
	if err := timers.InjectTo(isolate, global); err != nil {
		isolate.TerminateExecution()
		isolate.Dispose()
		return nil, nil, err
	}
	// Create the context
	context := v8go.NewContext(isolate, global)
	// URL support
	if err := url.InjectTo(context); err != nil {
		context.Close()
		isolate.TerminateExecution()
		isolate.Dispose()
		return nil, nil, err
	}
	// Console support
	if err := console.InjectTo(context, console.WithOutput(c.Error)); err != nil {
		context.Close()
		isolate.TerminateExecution()
		isolate.Dispose()
		return nil, nil, err
	}
	//structuredClone support
	context.RunScript(`globalThis.structuredClone = function (value) {
		return JSON.parse(JSON.stringify(value));
	};`, "structuredClone.js")
	return isolate, context, nil
}

func LoadV8(c *Console) (*V8VM, error) {
	isolate, context, err := load(c)
	if err != nil {
		return nil, err
	}
	return &V8VM{
		isolate: isolate,
		context: context,
	}, nil
}

type V8VM struct {
	isolate *v8go.Isolate
	context *v8go.Context
}

var _ VM = (*V8VM)(nil)

// Compile a script into the context
func (vm *V8VM) Compile(path, code string) error {
	script, err := vm.isolate.CompileUnboundScript(code, path, v8go.CompileOptions{})
	if err != nil {
		return err
	}
	// Bind to the context
	if _, err := script.Run(vm.context); err != nil {
		return err
	}
	return nil
}

func (vm *V8VM) Evaluate(ctx context.Context, path, expr string) (string, error) {
	value, err := vm.context.RunScript(expr, path)
	if err != nil {
		return "", err
	}
	// Handle promises
	if value.IsPromise() {
		prom, err := value.AsPromise()
		if err != nil {
			return "", err
		}
		// TODO: this could run forever
		for prom.State() == v8go.Pending {
			continue
		}
		return prom.Result().String(), nil
	}
	return value.String(), nil
}

func (vm *V8VM) Close() {
	vm.context.Close()
	vm.isolate.TerminateExecution()
	vm.isolate.Dispose()
}
