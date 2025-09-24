package router

import (
	"strconv"
)

// Parameter represents a URL parameter.
type Parameter struct {
	Key   string
	Value string
}

func (p Parameter) Int() (int, bool) {
	v, err := strconv.Atoi(p.Value)
	if err != nil {
		return 0, false
	}
	return v, true
}

func (p Parameter) Int64() (int64, bool) {
	v, err := strconv.ParseInt(p.Value, 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func (p Parameter) Uint() (uint, bool) {
	v, err := strconv.ParseUint(p.Value, 10, 0)
	if err != nil {
		return 0, false
	}
	return uint(v), true
}

func (p Parameter) Uint64() (uint64, bool) {
	v, err := strconv.ParseUint(p.Value, 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func (p Parameter) Float32() (float32, bool) {
	v, err := strconv.ParseFloat(p.Value, 32)
	if err != nil {
		return 0, false
	}
	return float32(v), true
}

func (p Parameter) Float64() (float64, bool) {
	v, err := strconv.ParseFloat(p.Value, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// flow tells the main loop what it should do next.
type flow int

// Control flow values.
const (
	flowStop flow = iota
	flowBegin
	flowNext
)
