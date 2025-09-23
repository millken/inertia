package router

// Parameter represents a URL parameter.
type Parameter struct {
	Key   string
	Value string
}

// flow tells the main loop what it should do next.
type flow int

// Control flow values.
const (
	flowStop flow = iota
	flowBegin
	flowNext
)
