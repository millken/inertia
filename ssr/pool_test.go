package ssr

import (
	"context"
	"errors"
	"testing"
)

// fakeVM is a deterministic, in-memory ssr.VM for testing the Pool without a
// real JS runtime. kind 0 = template, 1 = component.
type fakeVM struct {
	renderfn func(ctx context.Context, kind int, key string, data map[string]any) (string, error)
}

func (f *fakeVM) RenderTemplate(ctx context.Context, tpl string, data map[string]any) (string, error) {
	return f.renderfn(ctx, 0, tpl, data)
}
func (f *fakeVM) RenderComponent(ctx context.Context, name string, data map[string]any) (string, error) {
	return f.renderfn(ctx, 1, name, data)
}
func (f *fakeVM) Close() {}

func newFakePool(t *testing.T, size int, renderfn func(ctx context.Context, kind int, key string, data map[string]any) (string, error)) *Pool {
	t.Helper()
	p, err := NewPool(size, func() (VM, error) { return &fakeVM{renderfn: renderfn}, nil })
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	t.Cleanup(p.Close)
	return p
}

func TestPoolRender(t *testing.T) {
	p := newFakePool(t, 2, func(ctx context.Context, kind int, key string, data map[string]any) (string, error) {
		if kind == 1 {
			return "<c>" + key + "</c>", nil
		}
		return "<t>" + key + "</t>", nil
	})
	if got, err := p.RenderComponent(context.Background(), "Home", nil); err != nil || got != "<c>Home</c>" {
		t.Errorf("RenderComponent = %q,%v, want <c>Home</c>", got, err)
	}
	if got, err := p.RenderTemplate(context.Background(), "tpl", nil); err != nil || got != "<t>tpl</t>" {
		t.Errorf("RenderTemplate = %q,%v, want <t>tpl</t>", got, err)
	}
}

// TestPoolRecover ensures a panicking worker returns an error and the Pool
// stays usable, rather than deadlocking the caller on <-task.res.
func TestPoolRecover(t *testing.T) {
	p := newFakePool(t, 1, func(ctx context.Context, kind int, key string, data map[string]any) (string, error) {
		panic("boom")
	})
	if _, err := p.RenderComponent(context.Background(), "Home", nil); err == nil {
		t.Fatal("expected error from panicked render, got nil")
	}
	// Pool must still be usable after a recovered panic (no deadlock).
	if _, err := p.RenderTemplate(context.Background(), "tpl", nil); err == nil {
		t.Fatal("expected error from second panicked render, got nil")
	}
}

// TestPoolCtxCancel ensures a cancelled context unblocks the caller.
func TestPoolCtxCancel(t *testing.T) {
	p := newFakePool(t, 1, func(ctx context.Context, kind int, key string, data map[string]any) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := p.RenderComponent(ctx, "Home", nil); !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestPoolClosed(t *testing.T) {
	p := newFakePool(t, 1, func(ctx context.Context, kind int, key string, data map[string]any) (string, error) {
		return "ok", nil
	})
	p.Close()
	if _, err := p.RenderComponent(context.Background(), "Home", nil); err == nil {
		t.Fatal("expected error rendering on closed pool")
	}
}
