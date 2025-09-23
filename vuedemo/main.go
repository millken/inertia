package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/millken/inertia"
	"github.com/millken/inertia/controller"
	middleware "github.com/millken/inertia/middlware"
)

func main() {
	opts := []inertia.Option{
		inertia.WithErrorHandler(http.StatusNotFound, func(w http.ResponseWriter, r *http.Request, _ error) {

			http.Error(w, "Custom 404 Not Found: ", http.StatusNotFound)
		}),
		inertia.WithErrorHandler(http.StatusInternalServerError, func(w http.ResponseWriter, r *http.Request, err error) {
			slog.Error(err.Error())
			http.Error(w, "Custom 500 Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		}),
	}
	// opts = append(opts, inertia.WithRootTemplateHTML("<!DOCTYPE html><html><head><title>Inertia App</title></head><body>{{.PageHTML}}</body></html>"))
	// Create Inertia instance
	i, _ := inertia.New(opts...)
	i.DevMode = true

	// Add specific routes first
	i.Get("/", controller.Index)
	i.Get("/post/:id", controller.ShowPost)
	i.Get("/post/:id/edit", controller.EditPost)
	i.Get("/panic", controller.Panic)

	// Add static asset routes last (wildcard routes should be last)
	i.ServeAsset("/", os.DirFS("/workspace/Codes/github.com/millken/inertia/public"))
	// Add middleware first
	i.Use(middleware.Gzip(),
		middleware.AccessLog(), middleware.Recovery())
	slog.Info(fmt.Sprintf("> Starting at http://%s", i.Addr()))
	err := i.Serve()
	if err != nil {
		slog.Error(fmt.Sprintf("Server terminated: %v", err.Error()))
	}
}
