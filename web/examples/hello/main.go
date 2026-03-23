package main

import (
	"log"

	"github.com/millken/inertia/web"
)

func main() {
	engine, err := web.New(web.WithAddr(":8080"))
	if err != nil {
		log.Fatal(err)
	}

	engine.Use(func(c *web.Context) error {
		c.Header("X-Powered-By", "inertia/web")
		return c.Next()
	})

	engine.GET("/", func(c *web.Context) error {
		return c.String("hello, web")
	})

	engine.GET("/users/:id", func(c *web.Context) error {
		return c.JSON(map[string]any{
			"id":   c.Request.Param("id"),
			"tag":  c.Query("tag"),
			"addr": c.ClientIP(),
		})
	})

	log.Printf("listening on %s", engine.Addr())
	log.Fatal(engine.Run())
}