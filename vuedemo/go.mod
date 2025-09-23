module vuedemo

go 1.24.4

replace github.com/millken/inertia => ../

replace github.com/millken/inertia/middleware => ../middleware

require (
	github.com/millken/inertia v0.0.0-00010101000000-000000000000
	github.com/millken/inertia/middleware v0.0.0-00010101000000-000000000000
)

require github.com/brianvoe/gofakeit/v7 v7.1.2 // indirect
