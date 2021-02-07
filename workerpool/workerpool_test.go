package workerpool

import (
	"fmt"
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
			taskID := data.(int)
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("Task %d processed\n", taskID)
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
