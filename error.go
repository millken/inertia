package inertia

import (
	"fmt"
	"log/slog"
	"net/http"
)

// Error logs the error and sends an internal server error response.
func Error(w http.ResponseWriter, err error, HTTPStatus int) {
	slog.Error(err.Error())

	http.Error(w, err.Error(), HTTPStatus)
}

type errorHandlerFn func(w http.ResponseWriter, r *http.Request, err error)

var (
	ErrorHandlerMap = map[int]errorHandlerFn{
		http.StatusNotFound: func(w http.ResponseWriter, r *http.Request, err error) {
			http.NotFound(w, r)
		},

		http.StatusInternalServerError: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.Error(err.Error())
			fmt.Fprintln(w, err)
		},
	}
)

// defaultCatchAllHandler to log and return a 404 for all routes except the root route.
var defaultCatchAllHandler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	ErrorHandlerMap[http.StatusNotFound](w, r, nil)
})
