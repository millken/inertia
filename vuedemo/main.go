package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"vuedemo/controller"

	"github.com/millken/inertia"
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
		inertia.WithDevMode(true),
	}
	// Create Inertia instance
	iner, _ := inertia.New(opts...)
	defaultMeta := inertia.NewMeta().SetTitle("My Site").SetDescription("Welcome to my site").SetKeywords("inertia,go,vuejs")

	// Add specific routes first
	iner.GET("/", controller.Index)
	iner.GET("/post/:id", controller.ShowPost)
	iner.GET("/post/:id/edit", controller.EditPost)
	iner.GET("/panic", controller.Panic)

	// Add static asset routes last (wildcard routes should be last)
	iner.StaticFS("/", os.DirFS("./view/dist"))
	// Add middleware first
	fileLog, _ := os.OpenFile("access.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer fileLog.Close()
	iner.Use(middleware.AccessLog(middleware.WithAccessLogOutput(fileLog)))
	// Add other middleware
	iner.Use(middleware.Gzip(), middleware.Recovery(), inertia.UseMeta(defaultMeta))
	slog.Info(fmt.Sprintf("> Starting at http://%s", iner.Addr()))
	err := iner.Serve()
	if err != nil {
		slog.Error(fmt.Sprintf("Server terminated: %v", err.Error()))
	}
}
