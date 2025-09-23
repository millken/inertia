package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/millken/inertia"
)

func Hello(c *inertia.Context) {
	c.Write([]byte("Hello, World!"))
}

func Panic(c *inertia.Context) {
	panic("Something went wrong!")
}
func Index(c *inertia.Context) {

	var posts []*Post
	gofakeit.Slice(&posts)
	c.Set("text", "Hello, World!")
	c.Set("posts", posts)
	if err := c.Render("index/index"); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}
}

func ShowPost(c *inertia.Context) {
	id := c.Params.Get("id")
	idInt, _ := strconv.Atoi(id)
	post := Post{ID: idInt, Name: "Post " + id, Body: "This is the body of post " + id}
	c.Meta.SetTitle(post.Name).SetDescription(post.Body).SetKeywords("post,example,blog").SetAuthor("Admin")
	fmt.Println(c.Meta.ToHTML())
	c.Set("post", post)
	if err := c.Render("index/show"); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}
}

func EditPost(c *inertia.Context) {
	id := c.Params.Get("id")
	idInt, _ := strconv.Atoi(id)
	post := Post{ID: idInt, Name: "Post " + id, Body: "This is the body of post " + id}
	c.Set("post", post)
	if err := c.Render("index/edit"); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}
}
