package svelte

import (
	"cmp"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	"github.com/millken/inertia/gosvel/runtime/js"
)

//go:generate go run github.com/evanw/esbuild/cmd/esbuild compiler.ts --format=iife --global-name=__svelte__ --bundle --platform=node --inject:compiler_shim.ts --external:url --outfile=compiler.js --log-level=warning --minify

// compiler.js is used to compile .svelte files into JS & CSS
//
//go:embed compiler.js
var compiler string

func Load(ctx context.Context, js js.VM) (*Compiler, error) {
	if _, err := js.Evaluate(ctx, "svelte/compiler.js", compiler); err != nil {
		return nil, err
	}
	return &Compiler{js, false, ""}, nil
}

type Compiler struct {
	VM      js.VM
	Dev     bool
	RootDir string
}

type SSR struct {
	JS  string
	CSS string
}

// Compile server-rendered code
func (c *Compiler) Server(ctx context.Context, path string, code []byte) (*SSR, error) {
	cwd, _ := os.Getwd()
	expr := fmt.Sprintf(`;__svelte__.compile({ "path": %q, "code": %q, "target": "server", "dev": %t, "css": "external", "rootDir": %q })`, path, code, c.Dev, cmp.Or(c.RootDir, cwd))
	result, err := c.VM.Evaluate(ctx, path, expr)
	if err != nil {
		return nil, err
	}
	out := new(SSR)
	if err := json.Unmarshal([]byte(result), out); err != nil {
		return nil, err
	}
	return out, nil
}

type Client struct {
	JS  string
	CSS string
}

// Compile Client code
func (c *Compiler) Client(ctx context.Context, path string, code []byte) (*Client, error) {
	cwd, _ := os.Getwd()
	expr := fmt.Sprintf(`;__svelte__.compile({ "path": %q, "code": %q, "target": "client", "dev": %t, "css": "injected","rootDir": %q })`, path, code, c.Dev, cmp.Or(c.RootDir, cwd))
	result, err := c.VM.Evaluate(ctx, path, expr)
	if err != nil {
		return nil, err
	}
	out := new(Client)
	if err := json.Unmarshal([]byte(result), out); err != nil {
		return nil, err
	}
	return out, nil
}
