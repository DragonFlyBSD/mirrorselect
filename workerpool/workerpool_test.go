// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2021-2026 The DragonFly Project.

package workerpool

import (
	"sync"
	"testing"
	"time"
)

func TestWorkerPool1(t *testing.T) {
	const taskCount = 20

	tasks := make([]*Task, 0, taskCount)
	pool := NewPool(nil, 5)
	done := make(chan struct{})
	go func() {
		pool.Run()
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
		t.Fatal("Run did not stop")
	}

	for _, task := range tasks {
		if task.Err != nil {
			t.Error(task.Err)
		}
	}

	// Adding a task after stopping must not block.
	added := make(chan struct{})
	go func() {
		pool.AddTask(NewTask(func(data interface{}) error { return nil }, nil))
		close(added)
	}()

	select {
	case <-added:
	case <-time.After(time.Second):
		t.Fatal("AddTask blocked after Stop")
	}
}
