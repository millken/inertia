package v8go

import (
	"testing"

	"github.com/millken/inertia/ssr"
	"github.com/millken/inertia/ssr/testshared"
)

func TestV8goVM(t *testing.T) {
	newVM := func(opts ...ssr.Option) (ssr.VM, error) {
		return NewVM(opts...)
	}
	testshared.RunCommonVMTests(t, newVM, "ssr-render.js")
}
