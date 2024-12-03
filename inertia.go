package inertia

// Option is an option parameter that modifies Inertia.
type Option func(i *Inertia) error

type Inertia struct {
	rootTemplateHTML string
}
