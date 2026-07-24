package ssr

import (
	"context"
	"fmt"
	"sync"
)

var _ VM = (*Pool)(nil)

// Pool manages a set of VM workers to safely run VMs in parallel.
type Pool struct {
	workers []VM
	tasks   chan *renderTask
	wg      sync.WaitGroup
	mu      sync.Mutex
	closed  bool
}

type renderTask struct {
	ctx  context.Context
	kind int // 0: template, 1: component
	tpl  string
	name string
	data map[string]any
	res  chan renderResult
}

type renderResult struct {
	html string
	err  error
}

// NewPool creates a pool with the given size. vmFactory is a function that
// returns a fresh VM instance for each worker.
func NewPool(size int, vmFactory func() (VM, error)) (*Pool, error) {
	if size <= 0 {
		return nil, fmt.Errorf("pool size must be > 0")
	}
	p := &Pool{
		workers: make([]VM, 0, size),
		tasks:   make(chan *renderTask, size*4),
	}

	for i := 0; i < size; i++ {
		vm, err := vmFactory()
		if err != nil {
			// clean up already created VMs
			for _, w := range p.workers {
				w.Close()
			}
			return nil, err
		}
		p.workers = append(p.workers, vm)
		p.wg.Add(1)
		go p.runWorker(vm)
	}

	return p, nil
}

func (p *Pool) runWorker(vm VM) {
	defer p.wg.Done()
	for task := range p.tasks {
		if task == nil {
			continue
		}
		p.execTask(vm, task)
	}
}

// execTask runs one render with panic recovery so a panicking VM can never
// deadlock the caller on <-task.res.
func (p *Pool) execTask(vm VM, task *renderTask) {
	defer func() {
		if r := recover(); r != nil {
			task.res <- renderResult{err: fmt.Errorf("ssr: render panic recovered: %v", r)}
		}
	}()
	var (
		html string
		err  error
	)
	switch task.kind {
	case 0:
		html, err = vm.RenderTemplate(task.ctx, task.tpl, task.data)
	case 1:
		html, err = vm.RenderComponent(task.ctx, task.name, task.data)
	default:
		err = fmt.Errorf("unknown task kind: %d", task.kind)
	}
	task.res <- renderResult{html: html, err: err}
}

// RenderTemplate dispatches a template render to the pool and waits for result.
// Both the dispatch and the wait honor ctx: a cancelled/deadlined context
// unblocks the caller even if the pool is saturated or a worker is stuck.
func (p *Pool) RenderTemplate(ctx context.Context, tpl string, data map[string]any) (string, error) {
	p.mu.Lock()
	closed := p.closed
	p.mu.Unlock()
	if closed {
		return "", fmt.Errorf("pool closed")
	}

	task := &renderTask{ctx: ctx, kind: 0, tpl: tpl, data: data, res: make(chan renderResult, 1)}
	select {
	case p.tasks <- task:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	select {
	case r := <-task.res:
		return r.html, r.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// RenderComponent dispatches a component render to the pool and waits for result.
// See RenderTemplate for ctx semantics.
func (p *Pool) RenderComponent(ctx context.Context, name string, data map[string]any) (string, error) {
	p.mu.Lock()
	closed := p.closed
	p.mu.Unlock()
	if closed {
		return "", fmt.Errorf("pool closed")
	}

	task := &renderTask{ctx: ctx, kind: 1, name: name, data: data, res: make(chan renderResult, 1)}
	select {
	case p.tasks <- task:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	select {
	case r := <-task.res:
		return r.html, r.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// Close shuts down the pool and closes all underlying VMs.
func (p *Pool) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	close(p.tasks)
	p.mu.Unlock()

	p.wg.Wait()
	for _, w := range p.workers {
		w.Close()
	}
}
