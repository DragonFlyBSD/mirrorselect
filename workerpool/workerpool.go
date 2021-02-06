//
// A simple worker pool that runs tasks.
//
// Credit:
// https://brandur.org/go-worker-pool
//

package workerpool

import (
	"sync"
)


// Encapsulate a task in a work pool.
//
type Task struct {
	Err	error

	f	func() error
}

func NewTask(f func() error) *Task {
	return &Task{ f: f }
}

func (t *Task) Run(wg *sync.WaitGroup) {
	t.Err = t.f()
	wg.Done()
}


// A worker group that runs tasks at a configured concurrency.
//
type Pool struct {
	Tasks		[]*Task

	concurrency	int
	taskChan	chan *Task
	wg		sync.WaitGroup
}

func NewPool(tasks []*Task, concurrency int) *Pool {
	return &Pool{
		Tasks: tasks,
		concurrency: concurrency,
		taskChan: make(chan *Task),
	}
}

// Run all tasks in the pool and blocks until it's finished.
//
func (p *Pool) Run() {
	for i := 0; i < p.concurrency; i++ {
		go p.work()
	}

	p.wg.Add(len(p.Tasks))
	for _, task := range p.Tasks {
		p.taskChan <- task
	}

	close(p.taskChan)

	p.wg.Wait()
}

// The work loop for any single goroutine.
//
func (p *Pool) work() {
	for task := range p.taskChan {
		task.Run(&p.wg)
	}
}
