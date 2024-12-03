package controller

import (
	"net/http"

	"github.com/millken/inertia/server"
)

type API struct {
}

func NewAPI() *API {
	return &API{}
}

func (a *API) Health(w http.ResponseWriter, r *http.Request) {
	JSON(w, H{"status": "ok"})
}

func (a *API) Router(r server.Router) {
	r.HandleFunc("GET /healthz", a.Health)
}
