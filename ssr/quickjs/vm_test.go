package quickjs

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/millken/inertia/ssr"
)

// testBundle is a hermetic, dependency-free SSR bundle: it exports the two
// inertiaRender* functions returning Promises (the runtime awaits them),
// without requiring Vue or any external front-end build. This lets the quickjs
// VM run in CI.
const testBundle = `
module.exports.inertiaRenderComponent = function(name, propsJson) {
  var p = propsJson ? JSON.parse(propsJson) : {};
  return Promise.resolve("<c>" + name + ":" + (p.msg || "") + "</c>");
};
module.exports.inertiaRenderTemplate = function(tpl, propsJson) {
  var p = propsJson ? JSON.parse(propsJson) : {};
  return Promise.resolve("<t>" + (p.msg || "") + "</t>");
};
`

func TestQuickjsRender(t *testing.T) {
	vm, err := NewVM(ssr.WithBundlerJS(testBundle))
	if err != nil {
		t.Fatalf("NewVM: %v", err)
	}
	defer vm.Close()

	got, err := vm.RenderComponent(context.Background(), "Home", map[string]any{"msg": "hi"})
	if err != nil {
		t.Fatalf("RenderComponent: %v", err)
	}
	if want := "<c>Home:hi</c>"; got != want {
		t.Errorf("RenderComponent = %q, want %q", got, want)
	}

	got, err = vm.RenderTemplate(context.Background(), "tpl", map[string]any{"msg": "hi"})
	if err != nil {
		t.Fatalf("RenderTemplate: %v", err)
	}
	if want := "<t>hi</t>"; got != want {
		t.Errorf("RenderTemplate = %q, want %q", got, want)
	}
}

func TestQuickjsCache(t *testing.T) {
	vm, err := NewVM(ssr.WithBundlerJS(testBundle), ssr.WithDefaultCache(8))
	if err != nil {
		t.Fatal(err)
	}
	defer vm.Close()
	got1, err := vm.RenderComponent(context.Background(), "Home", map[string]any{"msg": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	got2, err := vm.RenderComponent(context.Background(), "Home", map[string]any{"msg": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if got1 != got2 {
		t.Errorf("cached render mismatch: %q vs %q", got1, got2)
	}
}

// TestQuickjsCtxCancel ensures a cancelled context unblocks the caller rather
// than waiting on the single-threaded worker.
func TestQuickjsCtxCancel(t *testing.T) {
	vm, err := NewVM(ssr.WithBundlerJS(testBundle))
	if err != nil {
		t.Fatal(err)
	}
	defer vm.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := vm.RenderComponent(ctx, "Home", nil); !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestQuickjsRenderConcurrent(t *testing.T) {
	vm, err := NewVM(ssr.WithBundlerJS(testBundle))
	if err != nil {
		t.Fatal(err)
	}
	defer vm.Close()

	const n = 10
	var wg sync.WaitGroup
	wg.Add(n)
	errs := make(chan error, n)
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			<-start
			got, err := vm.RenderComponent(context.Background(), "Home", map[string]any{"msg": "hi"})
			if err != nil {
				errs <- err
				return
			}
			if want := "<c>Home:hi</c>"; got != want {
				errs <- errors.New("mismatch: " + got)
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

func BenchmarkQuickjsRender(b *testing.B) {
	vm, err := NewVM(ssr.WithBundlerJS(testBundle))
	if err != nil {
		b.Fatal(err)
	}
	defer vm.Close()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := vm.RenderComponent(context.Background(), "Home", map[string]any{"msg": "hi"}); err != nil {
			b.Fatal(err)
		}
	}
}
