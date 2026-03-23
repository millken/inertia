package main

import (
	"log"

	"github.com/millken/inertia/web"
)

func main() {
	engine, err := web.New(
		web.WithAddr(":443"),
		web.WithAutoTLSCacheDir(".autocert"),
	)
	if err != nil {
		log.Fatal(err)
	}

	engine.GET("/", func(c *web.Context) error {
		return c.String("hello over autossl")
	})

	engine.GET("/healthz", func(c *web.Context) error {
		return c.JSON(map[string]any{"ok": true, "tls": true})
	})

	log.Printf("listening on %s with autossl", engine.Addr())
	log.Fatal(engine.RunAutoTLSRedirect(":80", "example.com", "www.example.com"))
}