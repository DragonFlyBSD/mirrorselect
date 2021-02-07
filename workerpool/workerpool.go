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
	quit	chan bool
}

func NewWorker(id int, tasks chan *Task) *Worker {
	return &Worker{
		ID: id,
		tasks: tasks,
		quit: make(chan bool),
	}
}

// Start the worker and do the tasks.
func (w *Worker) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for task := range w.tasks {
			task.Run()
		}
	}()
}

// Start the worker in background, waiting for tasks.
func (w *Worker) StartBackground() {
	for {
		select {
		case task := <-w.tasks:
			task.Run()
		case <-w.quit:
			return
		}
	}
}

// Stop the background worker.
func (w *Worker) Stop() {
	go func() {
		w.quit <- true
	}()
}


// A worker group that runs tasks at a configured concurrency.
//
type Pool struct {
	Tasks		[]*Task
	Workers		[]*Worker

	concurrency	int
	collector	chan *Task
	quit		chan bool
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

// Add a task to the pool.
func (p *Pool) AddTask(task *Task) {
	p.collector <- task
}

// Run the workers in background.
func (p *Pool) RunBackground() {
	for i := 0; i < p.concurrency; i++ {
		worker := NewWorker(i, p.collector)
		p.Workers = append(p.Workers, worker)
		go worker.StartBackground()
	}

	for i := range p.Tasks {
		p.collector <- p.Tasks[i]
	}

	p.quit = make(chan bool)
	<-p.quit
}

// Stop the background workers.
func (p *Pool) Stop() {
	for i := range p.Workers {
		p.Workers[i].Stop()
	}
	p.quit <- true
}
