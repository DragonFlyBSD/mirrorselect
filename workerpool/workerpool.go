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
	"sync"
)

// Encapsulate a task in a work pool.
type Task struct {
	Err error

	f    func(data any) error
	data any
}

func NewTask(f func(data any) error, data any) *Task {
	return &Task{f: f, data: data}
}

func (t *Task) Run() {
	t.Err = t.f(t.data)
}

// A worker handles the received tasks.
type Worker struct {
	ID int

	tasks    chan *Task
	quit     chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

func NewWorker(id int, tasks chan *Task) *Worker {
	return &Worker{
		ID:    id,
		tasks: tasks,
		quit:  make(chan struct{}),
	}
}

// Run the worker, process the existing tasks and wait for new tasks.
func (w *Worker) Run() {
	w.wg.Add(1)
	defer w.wg.Done()

	for {
		select {
		case task, ok := <-w.tasks:
			if !ok {
				return
			}
			task.Run()
		case <-w.quit:
			return
		}
	}
}

// Stop the worker.
func (w *Worker) Stop() {
	w.stopOnce.Do(func() {
		close(w.quit)
	})
	w.wg.Wait()
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
func (p *Pool) AddTask(task *Task) bool {
	select {
	case p.collector <- task:
		return true
	case <-p.quit: // don't block after stop
	}
	return false
}

// Run the worker pool to process tasks.
func (p *Pool) Run() {
	for _, worker := range p.workers {
		p.wg.Add(1)
		go func(w *Worker) {
			defer p.wg.Done()
			w.Run()
		}(worker)
	}

	for _, t := range p.tasks {
		p.AddTask(t)
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
