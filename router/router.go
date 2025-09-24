package router

import (
	"errors"
	"strings"
)

const Methods = "GET|POST|DELETE|PUT|PATCH|OPTIONS|HEAD"

var ErrMethodNotAllowed = errors.New("method not allowed")

// Router is a high-performance router.
type Router[T any] struct {
	get     Tree[T]
	post    Tree[T]
	delete  Tree[T]
	put     Tree[T]
	patch   Tree[T]
	options Tree[T]
	head    Tree[T]
}

// New creates a new router containing trees for every HTTP method.
func New[T any]() *Router[T] {
	return &Router[T]{}
}

// Add registers a new handler for the given method and path.
func (router *Router[T]) Add(method string, path string, handler T) error {
	if !strings.Contains(Methods, method) {
		return ErrMethodNotAllowed
	}
	tree := router.selectTree(method)
	tree.Add(path, handler)
	return nil
}

// Lookup finds the handler and parameters for the given route.
func (router *Router[T]) Lookup(method string, path string) (bool, T, []Parameter) {
	if method == "" || path == "" || !strings.Contains(Methods, method) {
		var empty T
		return false, empty, nil
	}
	if method[0] == 'G' {
		return router.get.Lookup(path)
	}

	tree := router.selectTree(method)
	return tree.Lookup(path)
}

// LookupNoAlloc finds the handler and parameters for the given route without using any memory allocations.
func (router *Router[T]) LookupNoAlloc(method string, path string, addParameter func(string, string)) (bool, T) {
	var empty T
	if method == "" || path == "" || !strings.Contains(Methods, method) {
		return false, empty
	}
	if method[0] == 'G' {
		return router.get.LookupNoAlloc(path, addParameter)
	}

	tree := router.selectTree(method)
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
