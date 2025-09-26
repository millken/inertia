package ssr

import (
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
		switch task.kind {
		case 0:
			html, err := vm.RenderTemplate(task.tpl, task.data)
			task.res <- renderResult{html: html, err: err}
		case 1:
			html, err := vm.RenderComponent(task.name, task.data)
			task.res <- renderResult{html: html, err: err}
		default:
			task.res <- renderResult{html: "", err: fmt.Errorf("unknown task kind: %d", task.kind)}
		}
	}
}

// RenderTemplate dispatches a template render to the pool and waits for result.
func (p *Pool) RenderTemplate(tpl string, data map[string]any) (string, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return "", fmt.Errorf("pool closed")
	}
	p.mu.Unlock()

	task := &renderTask{kind: 0, tpl: tpl, data: data, res: make(chan renderResult, 1)}
	p.tasks <- task
	r := <-task.res
	return r.html, r.err
}

// RenderComponent dispatches a component render to the pool and waits for result.
func (p *Pool) RenderComponent(name string, data map[string]any) (string, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return "", fmt.Errorf("pool closed")
	}
	p.mu.Unlock()

	task := &renderTask{kind: 1, name: name, data: data, res: make(chan renderResult, 1)}
	p.tasks <- task
	r := <-task.res
	return r.html, r.err
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
