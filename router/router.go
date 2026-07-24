package router

import (
	"errors"
	"strings"
)

const Methods = "GET|POST|DELETE|PUT|PATCH|OPTIONS|HEAD"

var ErrMethodNotAllowed = errors.New("method not allowed")

// ErrDuplicateRoute is returned by Add when the same (method, path) is
// registered twice. The radix tree would otherwise silently overwrite the
// earlier handler, so registering a duplicate is treated as a programmer error
// and surfaced instead of hidden.
var ErrDuplicateRoute = errors.New("duplicate route")

// Router is a high-performance router.
type Router[T any] struct {
	get     Tree[T]
	post    Tree[T]
	delete  Tree[T]
	put     Tree[T]
	patch   Tree[T]
	options Tree[T]
	head    Tree[T]

	// registered tracks "method normalizedPath" keys already added, so a
	// duplicate registration can be rejected before it silently overwrites a
	// handler in the tree.
	registered map[string]struct{}
}

// New creates a new router containing trees for every HTTP method.
func New[T any]() *Router[T] {
	return &Router[T]{registered: make(map[string]struct{})}
}

// Add registers a new handler for the given method and path. It returns
// ErrMethodNotAllowed for an unknown method and ErrDuplicateRoute if the same
// (method, path) shape was already registered.
func (router *Router[T]) Add(method string, path string, handler T) error {
	tree := router.selectTree(method)
	if tree == nil {
		return ErrMethodNotAllowed
	}

	if router.registered == nil {
		router.registered = make(map[string]struct{})
	}
	key := method + " " + normalizePattern(path)
	if _, dup := router.registered[key]; dup {
		return ErrDuplicateRoute
	}
	router.registered[key] = struct{}{}

	tree.Add(path, handler)
	return nil
}

// normalizePattern collapses parameter (":id") and wildcard ("*rest") segments
// to their kind marker (":" / "*") so two routes that differ only by parameter
// name — which occupy the same position in the radix tree and therefore
// collide — compare equal. A static segment and a parameter segment at the same
// position stay distinct (they legitimately coexist).
func normalizePattern(path string) string {
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		if seg == "" {
			continue
		}
		switch seg[0] {
		case parameter:
			segments[i] = string(parameter)
		case wildcard:
			segments[i] = string(wildcard)
		}
	}
	return strings.Join(segments, "/")
}

// Lookup finds the handler and parameters for the given route.
func (router *Router[T]) Lookup(method string, path string) (bool, T, []Parameter) {
	if method == "" || path == "" {
		var empty T
		return false, empty, nil
	}
	tree := router.selectTree(method)
	if tree == nil {
		var empty T
		return false, empty, nil
	}
	return tree.Lookup(path)
}

// LookupNoAlloc finds the handler and parameters for the given route without using any memory allocations.
func (router *Router[T]) LookupNoAlloc(method string, path string, addParameter func(string, string)) (bool, T) {
	var empty T
	if method == "" || path == "" {
		return false, empty
	}
	tree := router.selectTree(method)
	if tree == nil {
		return false, empty
	}
	return tree.LookupNoAlloc(path, addParameter)
}

// Map traverses all trees and calls the given function on every node.
func (router *Router[T]) Map(transform func(T) T) {
	router.get.Map(transform)
	router.post.Map(transform)
	router.delete.Map(transform)
	router.put.Map(transform)
	router.patch.Map(transform)
	router.options.Map(transform)
	router.head.Map(transform)
}

// selectTree returns the tree by the given HTTP method.
func (router *Router[T]) selectTree(method string) *Tree[T] {
	switch method {
	case "GET":
		return &router.get
	case "POST":
		return &router.post
	case "DELETE":
		return &router.delete
	case "PUT":
		return &router.put
	case "PATCH":
		return &router.patch
	case "OPTIONS":
		return &router.options
	case "HEAD":
		return &router.head
	default:
		return nil
	}
}
