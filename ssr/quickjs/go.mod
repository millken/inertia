module github.com/millken/inertia/ssr/quickjs

go 1.24.0

toolchain go1.24.4

require (
	github.com/buke/quickjs-go v0.6.0
	github.com/dnsoa/go/assert v1.1.2
	github.com/millken/inertia/ssr v0.0.0
)

replace github.com/millken/inertia/ssr => ../
