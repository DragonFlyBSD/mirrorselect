// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2021-2026 The DragonFly Project.

package workerpool

import (
	"sync"
	"testing"
	"time"
)

// Credit: https://brandur.org/go-worker-pool
func TestWorkerPool1(t *testing.T) {
	tasks := []*Task{
		NewTask(func(data interface{}) error { return nil }, nil),
		NewTask(func(data interface{}) error { return nil }, nil),
		NewTask(func(data interface{}) error { return nil }, nil),
	}

	p := NewPool(tasks, 3)
	p.Run()

	for _, task := range p.Tasks {
		if task.Err != nil {
			t.Error(task.Err)
		}
	}
}

// https://hackernoon.com/concurrency-in-golang-and-workerpool-part-2-l3w31q7
func TestWorkerPool2(t *testing.T) {
	var allTask []*Task
	for i := 0; i < 100; i++ {
		task := NewTask(func(data interface{}) error {
			time.Sleep(time.Millisecond)
			return nil
		}, i)
		allTask = append(allTask, task)
	}

	pool := NewPool(allTask, 5)
	pool.Run()

	for _, task := range pool.Tasks {
		if task.Err != nil {
			t.Error(task.Err)
		}
	}
}

func TestWorkerPool3(t *testing.T) {
	const taskCount = 20

	tasks := make([]*Task, 0, taskCount)
	pool := NewPool(nil, 5)
	done := make(chan struct{})
	go func() {
		pool.RunBackground()
		close(done)
	}()

	var wg sync.WaitGroup
	for i := 0; i < taskCount; i++ {
		task := NewTask(func(data interface{}) error {
			defer wg.Done()
			time.Sleep(time.Millisecond)
			return nil
		}, i)
		tasks = append(tasks, task)
		pool.AddTask(task)
		wg.Add(1)
	}

	wg.Wait()
	pool.Stop()
	pool.Stop() // Stopping the pool must be idempotent.

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("RunBackground did not stop")
	}

	for _, task := range tasks {
		if task.Err != nil {
			t.Error(task.Err)
		}
	}
}
