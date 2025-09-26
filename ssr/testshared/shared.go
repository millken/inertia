package testshared

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/millken/inertia/ssr"
)

// RunCommonVMTests runs a set of shared tests against a VM factory.
func RunCommonVMTests(t *testing.T, newVM func(...ssr.Option) (ssr.VM, error), ssrRenderPath string) {
	if os.Getenv("ssr_render_path") != "" {
		ssrRenderPath = os.Getenv("ssr_render_path")
	}

	vm, err := newVM(ssr.WithBundlerFile(ssrRenderPath))
	if err != nil {
		t.Fatalf("failed to create VM: %v", err)
	}
	defer vm.Close()

	t.Run("RenderTemplate", func(t *testing.T) {
		result, err := vm.RenderTemplate(`<div>{{ msg }}</div>`, map[string]any{"msg": "hello"})
		if err != nil {
			t.Fatalf("RenderTemplate error: %v", err)
		}
		if result != `<div>hello</div>` {
			t.Fatalf("unexpected result: %s", result)
		}
	})

	t.Run("RenderComponent", func(t *testing.T) {
		result, err := vm.RenderComponent(`Tiny`, map[string]any{"msg": "hello"})
		if err == nil {
			t.Fatalf("expected error for Tiny component")
		}
		result, err = vm.RenderComponent(`tiny`, map[string]any{"msg": "hello"})
		if err != nil {
			t.Fatalf("RenderComponent tiny error: %v", err)
		}
		if result != `<div>Hello, hello.</div>` {
			t.Fatalf("unexpected result: %s", result)
		}
	})

	t.Run("Style", func(t *testing.T) {
		result, err := vm.RenderComponent(`index/index`, map[string]any{
			"text": "My Page",
			"posts": []map[string]any{
				{"id": 6076008594505252933, "name": "RMHK", "body": "SpmvIpm"},
				{"id": 2112069284341772391, "name": "hRKc", "body": "GoJglxJiGo"},
				{"id": 8410961337293083680, "name": "bBfi", "body": "INnVnlYO"},
				{"id": 2568248320646248211, "name": "gmEwO", "body": "iaafuXcq"},
				{"id": 5723807497442001082, "name": "nQTEmObp", "body": "NyQDKEUl"},
			},
		})
		if err != nil {
			t.Fatalf("Style render error: %v", err)
		}
		_ = result
	})

	t.Run("Concurrent", func(t *testing.T) {
		ch := make(chan struct{})
		n := 10
		for i := 0; i < n; i++ {
			go func() {
				<-ch
				result, err := vm.RenderTemplate(`<div>{{ msg }}</div>`, map[string]any{"msg": "hello"})
				if err != nil {
					t.Errorf("concurrent render error: %v", err)
					return
				}
				if result != `<div>hello</div>` {
					t.Errorf("concurrent unexpected result: %s", result)
				}
			}()
		}
		close(ch)
	})

	t.Run("ConcurrentBenchmarkLike", func(t *testing.T) {
		ch := make(chan struct{})
		n := 10
		var wg sync.WaitGroup
		wg.Add(n)
		errs := make(chan error, n)
		for i := 0; i < n; i++ {
			go func() {
				defer wg.Done()
				<-ch
				for j := 0; j < 10; j++ {
					result, err := vm.RenderTemplate(`<div>{{ msg }}</div>`, map[string]any{"msg": "hello"})
					if err != nil {
						errs <- err
						return
					}
					if result != `<div>hello</div>` {
						errs <- fmt.Errorf("unexpected result: %s", result)
						return
					}
				}
			}()
		}
		close(ch)
		wg.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				t.Fatal(err)
			}
		}
	})
}
