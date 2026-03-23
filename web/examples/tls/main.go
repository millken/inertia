package main

import (
	"log"

	"github.com/millken/inertia/web"
)

func main() {
	engine, err := web.New(web.WithAddr(":443"))
	if err != nil {
		log.Fatal(err)
	}

	engine.GET("/", func(c *web.Context) error {
		return c.String("hello over tls")
	})

	engine.GET("/healthz", func(c *web.Context) error {
		return c.JSON(map[string]any{"ok": true})
	})

	log.Printf("listening on %s with manual tls", engine.Addr())
	log.Fatal(engine.RunTLS("server.crt", "server.key"))
}
