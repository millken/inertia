package main

import (
	"cmp"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/millken/inertia"
	"github.com/millken/inertia/controller"
	"github.com/millken/inertia/middleware"
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
	render.SetSvelteFS(os.DirFS("/workspace/Codes/github.com/millken/inertia/view"))
	render.SetCompileFS(os.DirFS("/workspace/Codes/github.com/millken/inertia/public/assets"))
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
	opts := []inertia.Option{
		inertia.WithErrorHandler(http.StatusNotFound, func(w http.ResponseWriter, r *http.Request, _ error) {

			http.Error(w, "Custom 404 Not Found: ", http.StatusNotFound)
		}),
		// inertia.WithErrorHandler(http.StatusInternalServerError, func(w http.ResponseWriter, r *http.Request, err error) {
		// 	slog.Error(err.Error())
		// 	http.Error(w, "Custom 500 Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		// }),
	}
	// opts = append(opts, inertia.WithRootTemplateHTML("<!DOCTYPE html><html><head><title>Inertia App</title></head><body>{{.PageHTML}}</body></html>"))
	// Create Inertia instance
	i, _ := inertia.New(opts...)
	i.DevMode = true

	// Add specific routes first
	i.Get("/", controller.Hello)
	i.Get("/panic", controller.Panic)

	// Add static asset routes last (wildcard routes should be last)
	i.ServeAsset("/assets/", os.DirFS("/workspace/Codes/github.com/millken/inertia/public/assets/"))
	// Add middleware first
	i.Use(middleware.Recovery(), middleware.Gzip())
	slog.Info(fmt.Sprintf("> Starting at http://%s", i.Addr()))
	err := i.Serve()
	if err != nil {
		slog.Error(fmt.Sprintf("Server terminated: %v", err.Error()))
	}
}
