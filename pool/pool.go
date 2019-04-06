package pool

import (
	"fmt"
	"time"
)

type Pool struct {
	Jobs chan Job
}
type Worker struct {
	Handler     func(chan []byte, Job)
	Pool        *Pool
	TimeoutSecs int
}

type Job struct {
	Query  string
	Result chan []byte
}

// New creates a worker pool
// queue: number of job queue
// concurrency: number of workers
// externalHandler: handler func
func New(queue, concurrency int, externalHandler func(chan []byte, Job)) *Pool {
	jobs := make(chan Job, queue)
	pool := &Pool{
		Jobs: jobs,
	}

	// create concurrency workers
	for i := 0; i < concurrency; i++ {
		worker := &Worker{
			Handler:     externalHandler,
			Pool:        pool,
			TimeoutSecs: 10,
		}
		go func() {
			worker.Listen()
		}()
	}
	return pool
}

func (w *Worker) Listen() {
	for job := range w.Pool.Jobs {
		finished := make(chan []byte)
		go w.Handler(finished, job)

		select {
		case <-time.After(time.Second * time.Duration(w.TimeoutSecs)):
			fmt.Println("timeout")
			job.Result <- []byte("timeout query: " + job.Query)
		case res := <-finished:
			fmt.Println("finished")
			job.Result <- res
		}
	}
}

func (pool *Pool) Send(job Job) {
	pool.Jobs <- job
	fmt.Println("Number of jobs: ", len(pool.Jobs))
}
