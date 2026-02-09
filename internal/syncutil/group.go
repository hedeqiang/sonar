// Package syncutil provides concurrency utilities.
package syncutil

import (
	"context"
	"sync"
)

// Group manages a set of goroutines that should be started and stopped together.
type Group struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewGroup creates a new Group derived from the given context.
func NewGroup(ctx context.Context) *Group {
	ctx, cancel := context.WithCancel(ctx)
	return &Group{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Context returns the group's context.
func (g *Group) Context() context.Context {
	return g.ctx
}

// Go launches a goroutine within the group.
// The function receives the group context and should return when the context is cancelled.
func (g *Group) Go(fn func(ctx context.Context)) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		fn(g.ctx)
	}()
}

// Stop cancels the group context and waits for all goroutines to finish.
func (g *Group) Stop() {
	g.cancel()
	g.wg.Wait()
}
