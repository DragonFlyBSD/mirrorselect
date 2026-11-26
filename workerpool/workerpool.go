// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2021-2026 The DragonFly Project.
//
// A simple worker pool that runs tasks.
//
// Credit:
// https://brandur.org/go-worker-pool
// https://hackernoon.com/concurrency-in-golang-and-workerpool-part-2-l3w31q7
//

package workerpool

import (
	"context"
	"sync"
)

// Encapsulate a task in a work pool.
type Task struct {
	Err error

	f    func(ctx context.Context, data any) error
	data any
}

func NewTask(f func(ctx context.Context, data any) error, data any) *Task {
	return &Task{f: f, data: data}
}

func (t *Task) Run(ctx context.Context) {
	t.Err = t.f(ctx, t.data)
}

// A worker handles the received tasks.
type Worker struct {
	ID int

	tasks     chan *Task
	quit      chan struct{}
	started   chan struct{}
	done      chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
}

func NewWorker(id int, tasks chan *Task) *Worker {
	return &Worker{
		ID:      id,
		tasks:   tasks,
		quit:    make(chan struct{}),
		started: make(chan struct{}),
		done:    make(chan struct{}),
	}
}

// Run the worker, process the existing tasks and wait for new tasks.
func (w *Worker) Run(ctx context.Context) {
	w.startOnce.Do(func() {
		close(w.started)
	})
	defer close(w.done)

	for {
		select {
		case task, ok := <-w.tasks:
			if !ok {
				return
			}
			task.Run(ctx)
		case <-w.quit:
			return
		case <-ctx.Done():
			return
		}
	}
}

// Stop the worker.
func (w *Worker) Stop() {
	w.stopOnce.Do(func() {
		close(w.quit)
	})
	select {
	case <-w.started:
		<-w.done
	default:
	}
}

// A worker group that runs tasks at a configured concurrency.
type Pool struct {
	tasks     []*Task
	workers   []*Worker
	collector chan *Task
	quit      chan struct{}
	stopOnce  sync.Once
	wg        sync.WaitGroup
}

func NewPool(tasks []*Task, concurrency int) *Pool {
	collector := make(chan *Task)
	workers := make([]*Worker, concurrency)
	for i := range concurrency {
		workers[i] = NewWorker(i, collector)
	}

	return &Pool{
		tasks:     tasks,
		workers:   workers,
		collector: collector,
		quit:      make(chan struct{}),
	}
}

// Add a task to the pool.
// It returns false if the context is canceled or the pool is stopped.
func (p *Pool) AddTask(ctx context.Context, task *Task) bool {
	select {
	case <-ctx.Done():
		return false
	case <-p.quit: // don't block after stop
		return false
	default:
	}

	select {
	case p.collector <- task:
		return true
	case <-ctx.Done():
	case <-p.quit: // don't block after stop
	}
	return false
}

// Run the worker pool to process tasks.
func (p *Pool) Run(ctx context.Context) {
	for _, worker := range p.workers {
		p.wg.Add(1)
		go func(w *Worker) {
			defer p.wg.Done()
			w.Run(ctx)
		}(worker)
	}

	for _, t := range p.tasks {
		if !p.AddTask(ctx, t) {
			break
		}
	}

	p.wg.Wait()
}

// Stop all workers.
func (p *Pool) Stop() {
	p.stopOnce.Do(func() {
		close(p.quit)
		for _, worker := range p.workers {
			worker.Stop()
		}
	})
	p.wg.Wait()
}
