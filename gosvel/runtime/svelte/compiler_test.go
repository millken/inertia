package svelte_test

import (
	"context"
	"os"
	"testing"

	"github.com/millken/inertia/gosvel/runtime/js"
	"github.com/millken/inertia/gosvel/runtime/svelte"
	"github.com/stretchr/testify/require"
)

func TestSSR(t *testing.T) {
	is := require.New(t)
	js, err := js.LoadV8(&js.Console{
		Log:   os.Stdout,
		Error: os.Stderr,
	})
	is.NoError(err)
	ctx := context.Background()
	compiler, err := svelte.Load(ctx, js)
	is.NoError(err)
	ssr, err := compiler.Server(ctx, "test.svelte", []byte(`<h1>hi world!</h1>`))
	is.NoError(err)
	is.Contains(ssr.JS, `import * as $ from "svelte/internal/server";`)
	is.Contains(ssr.JS, `hi world!</h1>`)
}

func TestSSRRecovery(t *testing.T) {
	is := require.New(t)
	js, err := js.LoadV8(&js.Console{
		Log:   os.Stdout,
		Error: os.Stderr,
	})
	is.NoError(err)
	ctx := context.Background()
	compiler, err := svelte.Load(ctx, js)
	is.NoError(err)
	ssr, err := compiler.Server(ctx, "test.svelte", []byte(`<h1>hi world!</h1></h1>`))
	is.Error(err)
	is.Contains(err.Error(), `attempted to close an element that was not open`)
	is.Contains(err.Error(), `<h1>hi world!</h1></h1`)
	is.Nil(ssr)
	_, err = compiler.Server(ctx, "test.svelte", []byte(`<h1>hi world!</h1>`))
	is.NoError(err)
}

func TestClient(t *testing.T) {
	is := require.New(t)
	js, err := js.LoadV8(&js.Console{
		Log:   os.Stdout,
		Error: os.Stderr,
	})
	is.NoError(err)
	ctx := context.Background()
	compiler, err := svelte.Load(ctx, js)
	is.NoError(err)
	dom, err := compiler.Client(ctx, "test.svelte", []byte(`<h1>hi world!</h1>`))
	is.NoError(err)
	is.Contains(dom.JS, `from "svelte/internal/client"`)
	is.Contains(dom.JS, `<h1>hi world!</h1>`)
}

func TestDOMRecovery(t *testing.T) {
	is := require.New(t)
	js, err := js.LoadV8(&js.Console{
		Log:   os.Stdout,
		Error: os.Stderr,
	})
	is.NoError(err)
	ctx := context.Background()
	compiler, err := svelte.Load(ctx, js)
	is.NoError(err)
	dom, err := compiler.Client(ctx, "test.svelte", []byte(`<h1>hi world!</h1></h1>`))
	is.Error(err)
	is.Nil(dom)
	dom, err = compiler.Client(ctx, "test.svelte", []byte(`<h1>hi world!</h1>`))
	is.NoError(err)
	is.NotNil(dom)
}
