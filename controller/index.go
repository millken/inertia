package controller

import (
	"net/http"
	"strconv"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/millken/inertia/pkg/render"
)

func Index(w http.ResponseWriter, r *http.Request) {
	render := render.FromContext(r.Context())

	var posts []*Post
	gofakeit.Slice(&posts)
	render.Set("text", "Hello, World!")
	render.Set("posts", posts)
	if err := render.Render("Post/Index"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func ShowPost(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	render := render.FromContext(r.Context())
	post := Post{ID: id, Name: "Post " + r.PathValue("id"), Body: "This is the body of post " + r.PathValue("id")}
	render.Set("post", post)
	if err := render.Render("Post/show"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func EditPost(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	render := render.FromContext(r.Context())
	post := Post{ID: id, Name: "Post " + r.PathValue("id"), Body: "This is the body of post " + r.PathValue("id")}
	render.Set("post", post)
	if err := render.Render("Post/edit"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
