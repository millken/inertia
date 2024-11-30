package es_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/millken/inertia/gosvel/runtime/es"
	"github.com/millken/inertia/gosvel/runtime/js"
	"github.com/millken/inertia/gosvel/runtime/virtual"
	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	is := require.New(t)
	builder := es.New()
	flag := new(es.Flag)
	flag.Minify = true
	flag.Embed = true
	ssr := es.DOM(flag)

	file, err := builder.Serve(ssr, "../../view/common.js")
	is.NoError(err)
	os.WriteFile("_esbuild.js", file.Contents, 0644)
	t.Logf("%s", file.Contents)
}

func TestBundle(t *testing.T) {
	is := require.New(t)
	builder := es.New()
	flag := new(es.Flag)
	flag.Minify = true
	flag.Embed = true
	client := es.DOM(flag)

	files, err := builder.Bundle(client, "./common.ts", "../../view/*.svelte")
	is.NoError(err)
	for _, file := range files {
		name := filepath.Base(file.Path)
		t.Log("write", name)
		os.WriteFile(name, file.Contents, 0644)
	}
}

func TestServeSSR(t *testing.T) {
	is := require.New(t)
	fsys := virtual.Tree{
		"node_modules/uid/index.js": &virtual.File{
			Data: []byte(`export default function uid() { return "uid" }`),
		},
		"node_modules/react-dom/server.js": &virtual.File{
			Data: []byte(`
				import uid from 'uid'
				export function renderToString() { return "<h1>hello</h1>" + uid() }
			`),
		},
		"node_modules/react/index.js": &virtual.File{
			Data: []byte(`export function createElement() { return {} }`),
		},
		"node_modules/@pkg/slugify/index.mjs": &virtual.File{
			Data: []byte(`export default function slugify(title) { return title }`),
		},
		"view/Header.jsx": &virtual.File{
			Data: []byte(`export default (props) => <h1>{props.title}</h1>`),
		},
		"view/index.jsx": &virtual.File{
			Data: []byte(`
				import { renderToString } from 'react-dom/server'
				import slugify from '@pkg/slugify'
				import * as React from 'react'
				import Header from './Header.jsx'
				export function render (props) {
					return renderToString(<Header title={slugify(props.title)} />)
				}
			`),
		},
	}
	builder := es.New()
	flag := new(es.Flag)
	ssr := es.SSR(flag)
	ssr.GlobalName = "bud"
	ssr.Plugins = append(ssr.Plugins, es.FS(fsys, "virtual"))
	ssr.Loader[".jsx"] = api.LoaderJSX
	ssr.Loader[".mjs"] = api.LoaderJS
	ssr.Loader[".js"] = api.LoaderJS
	file, err := builder.Serve(ssr, "./view/index.jsx")
	is.NoError(err)
	vm, err := js.LoadV8(&js.Console{})
	is.NoError(err)
	defer vm.Close()
	result, err := vm.Evaluate(context.Background(), "view/index.jsx", fmt.Sprintf(`%s; bud.render({ title: "hello" })`, string(file.Contents)))
	is.NoError(err)
	is.Equal(result, `<h1>hello</h1>uid`)
}
