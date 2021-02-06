package workerpool

import "testing"


// Credit: https://brandur.org/go-worker-pool
func TestWorkerPool(t *testing.T) {
	tasks := []*Task{
		NewTask(func() error { return nil }),
		NewTask(func() error { return nil }),
		NewTask(func() error { return nil }),
	}

	p := NewPool(tasks, 3)
	p.Run()

	for _, task := range p.Tasks {
		if task.Err != nil {
			t.Error(task.Err)
		}
	}
}
