package task2

import "sync"

type Task interface {
	Name() string
	Context() *Context

	Start()
	Wait() error

	Status() string
	Summary() string
}

type TaskFunc struct {
	name string
	f    func(*Context) error
	w    sync.WaitGroup
	err  error

	x *Context

	started, done bool
}

func NewTaskFunc(name string, parent *Context, f func(*Context) error) Task {
	return &TaskFunc{
		name: name,
		f:    f,
		x:    parent.Child(),
	}
}

func (t *TaskFunc) Name() string { return t.name }

func (t *TaskFunc) Context() *Context { return t.x }

func (t *TaskFunc) Start() {
	if t.started {
		panic("simpleTask: already started")
	}
	t.started = true
	t.w.Add(1)
	go func() {
		defer t.w.Done()
		t.err = t.f(t.x)
		t.done = true
	}()
}

func (t *TaskFunc) Wait() error {
	if !t.started {
		panic("simpleTask: not yet started")
	}
	t.w.Wait()
	return t.err
}

func (t *TaskFunc) Status() string {
	if t.done {
		return "done"
	} else if t.started {
		return "working..."
	}
	return "pending"
}

func (t *TaskFunc) Summary() string { return "" }
