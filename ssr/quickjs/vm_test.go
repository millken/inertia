package quickjs

import (
	"os"
	"sync"
	"testing"

	"github.com/dnsoa/go/assert"
	"github.com/millken/inertia/ssr"
)

var (
	ssrRenderPath = "/workspace/Codes/github.com/millken/inertia-vue-template/dist/ssr-render-cjs.js"
)

// resolveSSRBundle returns the path to the SSR bundle used by these tests,
// honoring the ssr_render_path env var. The bundle is produced by an external
// front-end project and is not part of this repo, so when it is unavailable
// (e.g. in CI) the test is skipped rather than failed.
func resolveSSRBundle(t *testing.T) string {
	t.Helper()
	p := ssrRenderPath
	if env := os.Getenv("ssr_render_path"); env != "" {
		p = env
	}
	if _, err := os.Stat(p); err != nil {
		t.Skipf("SSR bundle not available (%s); set ssr_render_path to run", p)
	}
	return p
}

func TestVMRender(t *testing.T) {
	ssrRenderPath = resolveSSRBundle(t)
	r := assert.New(t)
	vm, err := NewVM(ssr.WithBundlerFile(ssrRenderPath))
	r.NoError(err)
	// defer vm.Close()
	t.Run("RenderTemplate", func(t *testing.T) {
		result, err := vm.RenderTemplate(`<div>{{ msg }}</div>`, map[string]any{"msg": "hello"})
		r.NoError(err)
		r.Equal(`<div>hello</div>`, result)
	})
	t.Run("RenderComponent", func(t *testing.T) {
		result, err := vm.RenderComponent(`Tiny`, map[string]any{"msg": "hello"})
		r.Error(err)
		r.Empty(result)
		result, err = vm.RenderComponent(`tiny`, map[string]any{"msg": "hello"})
		r.NoError(err)
		// fmt.Println("result:", result)
		r.Equal(`<div>Hello, hello.</div>`, result)
	})

}

func TestStyle(t *testing.T) {
	ssrRenderPath = resolveSSRBundle(t)
	r := assert.New(t)
	vm, err := NewVM(ssr.WithBundlerFile(ssrRenderPath))
	r.NoError(err)
	defer vm.Close()
	result, err := vm.RenderComponent(`index/index`, map[string]any{
		"text": "My Page",
		"posts": []map[string]any{
			{
				"id":   6076008594505252933,
				"name": "RMHK",
				"body": "SpmvIpm",
			},
			{
				"id":   2112069284341772391,
				"name": "hRKc",
				"body": "GoJglxJiGo",
			},
			{
				"id":   8410961337293083680,
				"name": "bBfi",
				"body": "INnVnlYO",
			},
			{
				"id":   2568248320646248211,
				"name": "gmEwO",
				"body": "iaafuXcq",
			},
			{
				"id":   5723807497442001082,
				"name": "nQTEmObp",
				"body": "NyQDKEUl",
			},
		},
	})
	r.NoError(err)
	println(result)
}
func BenchmarkVMRender(b *testing.B) {
	vm, err := NewVM(ssr.WithBundlerFile(ssrRenderPath))
	if err != nil {
		b.Fatal(err)
	}
	defer vm.Close()
	b.Run("RenderTemplate", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := vm.RenderTemplate(`<div>{{ msg }}</div>`, map[string]any{"msg": "hello"})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("RenderComponent", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := vm.RenderComponent(`index/index`, map[string]any{
				"text": "My Page",
				"posts": []map[string]any{
					{
						"id":   6076008594505252933,
						"name": "RMHK",
						"body": "SpmvIpm",
					},
				}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func TestVMRender_Concurrent(t *testing.T) {
	ssrRenderPath = resolveSSRBundle(t)
	r := assert.New(t)
	vm, err := NewVM(ssr.WithBundlerFile(ssrRenderPath))
	r.NoError(err)
	defer vm.Close()
	ch := make(chan struct{})
	n := 10
	for i := 0; i < n; i++ {
		go func() {
			<-ch
			result, err := vm.RenderTemplate(`<div>{{ msg }}</div>`, map[string]any{"msg": "hello"})
			r.NoError(err)
			r.Equal(`<div>hello</div>`, result)
		}()
	}
	close(ch)
}

func BenchmarkVMRender_Concurrent(b *testing.B) {
	options := []ssr.Option{
		ssr.WithDefaultCache(128),
		ssr.WithBundlerFile(ssrRenderPath),
	}
	vm, err := NewVM(options...)
	if err != nil {
		b.Fatal(err)
	}
	defer vm.Close()
	ch := make(chan struct{})
	n := 10

	var wg sync.WaitGroup
	wg.Add(n)
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			<-ch
			for j := 0; j < b.N; j++ {
				result, err := vm.RenderTemplate(`<div>{{ msg }}</div>`, map[string]any{"msg": "hello"})
				if err != nil {
					errs <- err
					return
				}
				if result != `<div>hello</div>` {
					errs <- err
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
			b.Fatal(err)
		}
	}
}
