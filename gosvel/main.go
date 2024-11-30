package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/millken/inertia/gosvel/runtime/es"
)

var (
	flagInput  = flag.String("input", "", "input file, multiple files are separated by comma")
	flagOutput = flag.String("output", "", "output directory")
	flagClean  = flag.Bool("clean", false, "clean output directory before generate")
	flagMinify = flag.Bool("minify", false, "minify output")
	flagEmbed  = flag.Bool("embed", false, "embed output")
	flagSSR    = flag.Bool("ssr", false, "server side rendering")
	flagDebug  = flag.Bool("debug", false, "debug mode")
)

func log(v ...interface{}) {
	if *flagDebug {
		fmt.Println(v...)
	}
}

func main() {
	flag.Parse()
	if *flagInput == "" {
		flag.PrintDefaults()
		return
	}
	if *flagOutput == "" {
		flag.PrintDefaults()
		return
	}
	log("input:", *flagInput)
	log("output:", *flagOutput)
	log("clean:", *flagClean)
	log("minify:", *flagMinify)
	log("embed:", *flagEmbed)
	log("ssr:", *flagSSR)
	builder := es.New()
	flag := new(es.Flag)
	flag.Minify = *flagMinify
	flag.Embed = *flagEmbed
	var esbuildOption api.BuildOptions
	if *flagSSR {
		esbuildOption = es.SSR(flag)
	} else {
		esbuildOption = es.DOM(flag)
	}
	files, err := builder.Bundle(esbuildOption, strings.Split(*flagInput, ",")...)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	log("generate", len(files), "files")
	if *flagClean {
		log("clean output directory")
		if err := os.RemoveAll(*flagOutput); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		log("create output directory")
		if err := os.MkdirAll(*flagOutput, 0755); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	for _, file := range files {
		log("file:", file.Path)
		cwd, _ := os.Getwd()
		name := filepath.Clean(*flagOutput + strings.TrimPrefix(file.Path, cwd))
		log("write", name)
		if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if err := os.WriteFile(name, file.Contents, 0644); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	fmt.Println("ok")
	os.Exit(0)
}
