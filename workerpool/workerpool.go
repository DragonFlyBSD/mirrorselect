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
//
type Task struct {
	Err	error

	data	interface{}
	f	func(data interface{}) error
}

func NewTask(f func(data interface{}) error, data interface{}) *Task {
	return &Task{ f: f, data: data }
}

func (t *Task) Run() {
	t.Err = t.f(t.data)
}


// A worker handles the received tasks.
//
type Worker struct {
	ID	int

	tasks	chan *Task
}

func NewWorker(id int, tasks chan *Task) *Worker {
	return &Worker{
		ID: id,
		tasks: tasks,
	}
}

func (w *Worker) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for task := range w.tasks {
			task.Run()
		}
	}()
}


// A worker group that runs tasks at a configured concurrency.
//
type Pool struct {
	Tasks		[]*Task

	concurrency	int
	collector	chan *Task
	wg		sync.WaitGroup
}

func NewPool(tasks []*Task, concurrency int) *Pool {
	return &Pool{
		Tasks: tasks,
		concurrency: concurrency,
		collector: make(chan *Task),
	}
}

// Run all tasks in the pool and blocks until it's finished.
//
func (p *Pool) Run() {
	for i := 0; i < p.concurrency; i++ {
		worker := NewWorker(i, p.collector)
		worker.Start(&p.wg)
	}

	for i := range p.Tasks {
		p.collector <- p.Tasks[i]
	}

	close(p.collector)

	p.wg.Wait()
}
