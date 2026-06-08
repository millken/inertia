module github.com/millken/inertia/ssr/v8go

go 1.26.2

require (
	github.com/dnsoa/go/assert v1.1.2
	github.com/millken/inertia/ssr v1.0.4
	github.com/tommie/v8go v0.34.0
)

require (
	github.com/tommie/v8go/deps/android_amd64 v0.0.0-20251007175045-97bcf8e7d6ed // indirect
	github.com/tommie/v8go/deps/android_arm64 v0.0.0-20251007175045-97bcf8e7d6ed // indirect
	github.com/tommie/v8go/deps/darwin_amd64 v0.0.0-20251007175045-97bcf8e7d6ed // indirect
	github.com/tommie/v8go/deps/darwin_arm64 v0.0.0-20251007175045-97bcf8e7d6ed // indirect
	github.com/tommie/v8go/deps/linux_amd64 v0.0.0-20251007175045-97bcf8e7d6ed // indirect
	github.com/tommie/v8go/deps/linux_arm64 v0.0.0-20251007175045-97bcf8e7d6ed // indirect
)

replace github.com/millken/inertia/ssr => ../
