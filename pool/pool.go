package pool

import (
	"fmt"
	"time"
)

type Pool struct {
	Jobs chan Job
}
type Worker struct {
	Handler     func(chan bool, Job)
	Pool        *Pool
	TimeoutSecs int
}

type Job struct {
	Query  string
	Quit   chan struct{}
	Result chan []byte
}

func (w *Worker) Listen() {
	for job := range w.Pool.Jobs {
		finished := make(chan bool)
		go w.Handler(finished, job)

		select {
		case <-time.After(time.Second * time.Duration(w.TimeoutSecs)):
			fmt.Println("timeout")
		case ok := <-finished:
			fmt.Println("finished: ", ok)
		case <-job.Quit:
			fmt.Println("quit")
		}
	}
}

func (pool *Pool) Send(job Job) {
	pool.Jobs <- job
	fmt.Println("Number of jobs: ", len(pool.Jobs))
}
