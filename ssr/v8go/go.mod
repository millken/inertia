module github.com/millken/inertia/ssr/v8go

go 1.24.0

toolchain go1.24.4

require (
	github.com/dnsoa/go/assert v1.1.2
	github.com/millken/inertia/ssr v0.0.0
	github.com/tommie/v8go v0.33.0
)

require (
	github.com/tommie/v8go/deps/android_amd64 v0.0.0-20250515043113-5dcc98077472 // indirect
	github.com/tommie/v8go/deps/android_arm64 v0.0.0-20250515043113-5dcc98077472 // indirect
	github.com/tommie/v8go/deps/darwin_amd64 v0.0.0-20250515043113-5dcc98077472 // indirect
	github.com/tommie/v8go/deps/darwin_arm64 v0.0.0-20250515043113-5dcc98077472 // indirect
	github.com/tommie/v8go/deps/linux_amd64 v0.0.0-20250515043113-5dcc98077472 // indirect
	github.com/tommie/v8go/deps/linux_arm64 v0.0.0-20250515043113-5dcc98077472 // indirect
)

replace github.com/millken/inertia/ssr => ../
