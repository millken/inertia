package es

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/evanw/esbuild/pkg/api"
	esbuild "github.com/evanw/esbuild/pkg/api"
	"github.com/millken/inertia/gosvel/runtime/js"
	"github.com/millken/inertia/gosvel/runtime/svelte"
)

func httpPlugin() esbuild.Plugin {
	return esbuild.Plugin{
		Name: "bud-http",
		Setup: func(epb esbuild.PluginBuild) {
			epb.OnResolve(esbuild.OnResolveOptions{Filter: `^http[s]?://`}, func(args esbuild.OnResolveArgs) (result esbuild.OnResolveResult, err error) {
				result.Namespace = "http"
				result.Path = args.Path
				return result, nil
			})
			epb.OnLoad(esbuild.OnLoadOptions{Filter: `.*`, Namespace: `http`}, func(args esbuild.OnLoadArgs) (result esbuild.OnLoadResult, err error) {
				res, err := http.Get(args.Path)
				if err != nil {
					return result, err
				}
				defer res.Body.Close()
				body, err := io.ReadAll(res.Body)
				if err != nil {
					return result, err
				}
				contents := string(body)
				result.Contents = &contents
				result.Loader = esbuild.LoaderJS
				return result, nil
			})
		},
	}
}

func npmPlugin() esbuild.Plugin {
	return esbuild.Plugin{
		Name: "bud-npm",
		Setup: func(epb esbuild.PluginBuild) {
			epb.OnResolve(esbuild.OnResolveOptions{Filter: `^[a-z0-9@]`}, func(args esbuild.OnResolveArgs) (result esbuild.OnResolveResult, err error) {
				result.Path = filepath.Join("bud", "npm", args.Path)
				result.External = true
				return result, nil
			})
		},
	}
}

// 定义 Svelte Loader 插件
func SvelteLoader() esbuild.Plugin {
	return esbuild.Plugin{
		Name: "svelte-loader",
		Setup: func(build esbuild.PluginBuild) {
			build.OnEnd(func(result *esbuild.BuildResult) (r esbuild.OnEndResult, err error) {
				// 将 svelte 编译器写入文件
				return
			})
			build.OnLoad(esbuild.OnLoadOptions{Filter: `\.svelte$`}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				//TODO: 缓存，避免重复加载
				// 使用 svelte 编译器编译 Svelte 文件
				js, err := js.LoadV8(&js.Console{
					Log:   os.Stdout,
					Error: os.Stderr,
				})
				if err != nil {
					return esbuild.OnLoadResult{}, fmt.Errorf("failed to load V8: %s", err)
				}
				ctx := context.Background()
				compiler, err := svelte.Load(ctx, js)
				if err != nil {
					return esbuild.OnLoadResult{}, fmt.Errorf("failed to load svelte compiler: %s", err)
				}
				content, err := os.ReadFile(args.Path)
				if err != nil {
					return esbuild.OnLoadResult{}, fmt.Errorf("failed to read svelte file: %s", err)
				}
				client, err := compiler.Client(ctx, args.Path, content)
				if err != nil {
					return esbuild.OnLoadResult{}, fmt.Errorf("failed to compile svelte file: %s", err)
				}

				contents := string(client.JS)
				return api.OnLoadResult{
					Contents: &contents,
					Loader:   esbuild.LoaderJS,
				}, nil
			})
		},
	}
}
