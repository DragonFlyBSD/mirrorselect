// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2021-2026 The DragonFly Project.

package workerpool

import (
	"context"
	"errors"
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
		pool.Run(context.Background())
		close(done)
	}()

	var wg sync.WaitGroup
	for i := 0; i < taskCount; i++ {
		task := NewTask(func(_ context.Context, _ any) error {
			defer wg.Done()
			time.Sleep(time.Millisecond)
			return nil
		}, i)
		tasks = append(tasks, task)
		pool.AddTask(context.Background(), task)
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
		pool.AddTask(context.Background(),
			NewTask(func(_ context.Context, _ any) error { return nil }, nil))
		close(added)
	}()

	select {
	case <-added:
	case <-time.After(time.Second):
		t.Fatal("AddTask blocked after Stop")
	}
}

func TestPoolCancelTask(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan struct{})
	pool := NewPool(nil, 2)
	done := make(chan struct{})
	go func() {
		pool.Run(ctx)
		close(done)
	}()

	task := NewTask(func(ctx context.Context, _ any) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}, nil)
	if !pool.AddTask(ctx, task) {
		t.Fatal("AddTask failed")
	}

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("Task did not start")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not stop after context cancellation")
	}

	if !errors.Is(task.Err, context.Canceled) {
		t.Errorf("Task error = %v; want %v", task.Err, context.Canceled)
	}

	// Adding a task with a canceled context must not block.
	if pool.AddTask(ctx, NewTask(func(context.Context, any) error {
		return nil
	}, nil)) {
		t.Error("AddTask succeeded after context cancellation")
	}
}

func TestAddTaskCanceledWhileBlocked(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	pool := NewPool(nil, 0)

	added := make(chan bool)
	go func() {
		added <- pool.AddTask(ctx, NewTask(func(context.Context, any) error {
			return nil
		}, nil))
	}()

	select {
	case <-added:
		t.Fatal("AddTask did not block without workers")
	case <-time.After(50 * time.Millisecond):
	}

	cancel()
	select {
	case ok := <-added:
		if ok {
			t.Error("AddTask succeeded after context cancellation")
		}
	case <-time.After(time.Second):
		t.Fatal("AddTask did not return after context cancellation")
	}
}

func TestRunCanceledInitialTasks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	tasks := []*Task{
		NewTask(func(context.Context, any) error { return nil }, nil),
	}
	pool := NewPool(tasks, 0)

	done := make(chan struct{})
	go func() {
		pool.Run(ctx)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not stop after context cancellation")
	}
}

func TestStopBeforeRun(t *testing.T) {
	pool := NewPool(nil, 2)
	pool.Stop()

	done := make(chan struct{})
	go func() {
		pool.Run(context.Background())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not stop after Stop")
	}
}
