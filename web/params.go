package web

import "github.com/millken/inertia/router"

type Params []router.Parameter

func (p Params) Get(key string) string {
	for _, param := range p {
		if param.Key == key {
			return param.Value
		}
	}
	return ""
}