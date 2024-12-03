package main

import (
	"cmp"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/millken/inertia/controller"
	"github.com/millken/inertia/pkg/render"
	"github.com/millken/inertia/public"
	"github.com/millken/inertia/server"
)

type Server interface {
	Addr() string
	Handler() http.Handler
}

func NewServer() Server {
	r := server.New(
		server.WithHost(cmp.Or(os.Getenv("HOST"), "127.0.0.1")),
		server.WithPort(cmp.Or(os.Getenv("PORT"), "5000")),
	)

	// r.Use(server.Gzip, server.Logger, server.Recovery)

	render := render.New()
	render.SetSvelteFS(os.DirFS("/media/millken/code01-data/Codes/github.com/millken/inertia/view"))
	render.SetCompileFS(os.DirFS("/media/millken/code01-data/Codes/github.com/millken/inertia/public/assets"))
	b, err := public.Files.ReadFile("index.html")
	if err != nil {
		slog.Error(fmt.Sprintf("Error reading index.html: %v", err.Error()))
	}
	render.SetRootTemplateHTML(b)
	r.Use(render.Middleware(), server.HttpDump)
	r.Folder("/", public.Files)
	r.HandleFunc("GET /{$}", controller.Index)
	r.HandleFunc("GET /{id}", controller.ShowPost)
	r.HandleFunc("GET /{id}/edit", controller.EditPost)
	return r
}

func main() {
	server := NewServer()
	slog.Info(fmt.Sprintf("> Starting at http://%s", server.Addr()))
	err := http.ListenAndServe(server.Addr(), server.Handler())
	if err != nil {
		slog.Error(fmt.Sprintf("Server terminated: %v", err.Error()))
	}
}
