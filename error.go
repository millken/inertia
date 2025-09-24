package inertia

import (
	"log/slog"
	"net/http"
)

// Error logs the error and sends an internal server error response.
func Error(w http.ResponseWriter, err error, HTTPStatus int) {
	slog.Error(err.Error(), slog.Int("status", HTTPStatus))

	http.Error(w, err.Error(), HTTPStatus)
}

type ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request, err error)

var (
	ErrorHandlerMap = map[int]ErrorHandlerFunc{
		http.StatusNotFound: func(w http.ResponseWriter, r *http.Request, err error) {
			http.NotFound(w, r)
		},
		http.StatusForbidden: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
		},

		http.StatusInternalServerError: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.Error(err.Error(), slog.Int("status", http.StatusInternalServerError), slog.String("method", r.Method), slog.String("path", r.URL.Path))
			http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		},
	}
)

// defaultCatchAllHandler to log and return a 404 for all routes except the root route.
var defaultCatchAllHandler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	ErrorHandlerMap[http.StatusNotFound](w, r, nil)
})
