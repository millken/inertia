package inertia

import (
	"github.com/millken/inertia/router"
)

type Params []router.Parameter

func (p Params) Get(key string) string {
	for _, param := range p {
		if param.Key == key {
			return param.Value
		}
	}
	return ""
}

func (p Params) GetInt(key string) (int, bool) {
	for _, param := range p {
		if param.Key == key {
			return param.Int()
		}
	}
	return 0, false
}

func (p Params) GetInt64(key string) (int64, bool) {
	for _, param := range p {
		if param.Key == key {
			return param.Int64()
		}
	}
	return 0, false
}

func (p Params) GetUint(key string) (uint, bool) {
	for _, param := range p {
		if param.Key == key {
			return param.Uint()
		}
	}
	return 0, false
}

func (p Params) GetUint64(key string) (uint64, bool) {
	for _, param := range p {
		if param.Key == key {
			return param.Uint64()
		}
	}
	return 0, false
}

func (p Params) GetFloat32(key string) (float32, bool) {
	for _, param := range p {
		if param.Key == key {
			return param.Float32()
		}
	}
	return 0, false
}

func (p Params) GetFloat64(key string) (float64, bool) {
	for _, param := range p {
		if param.Key == key {
			return param.Float64()
		}
	}
	return 0, false
}
