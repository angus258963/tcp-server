package pool

import (
	"fmt"
	"time"

	"github.com/angus258963/tcp-server/ratelimit"
)

type Pool struct {
	Jobs           chan Job
	UnfinishedJobs chan struct{}
}
type Worker struct {
	Handler     func(chan []byte, Job)
	Rate        *ratelimit.Ratelimit
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
func New(queue, concurrency int, rate *ratelimit.Ratelimit, externalHandler func(chan []byte, Job)) *Pool {
	jobs := make(chan Job, queue)
	unfinishedJobs := make(chan struct{}, concurrency+queue)
	pool := &Pool{
		Jobs:           jobs,
		UnfinishedJobs: unfinishedJobs,
	}

	// create concurrency workers
	for i := 0; i < concurrency; i++ {
		worker := &Worker{
			Handler:     externalHandler,
			Rate:        rate,
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
		w.handleJob(job)
		<-w.Pool.UnfinishedJobs
	}
}

func (pool *Pool) Send(job Job) {
	pool.Jobs <- job
	fmt.Println("Number of pending jobs: ", len(pool.Jobs))
	pool.UnfinishedJobs <- struct{}{}
	fmt.Println("Number of jobs: ", len(pool.UnfinishedJobs))
}

func (pool *Pool) GetNumberOfJobs() int {
	return len(pool.UnfinishedJobs)
}
func (pool *Pool) GetNumberOfRunningJobs() int {
	return len(pool.UnfinishedJobs) - len(pool.Jobs)
}

func (w *Worker) handleJob(job Job) {
	if !w.Rate.Acquire() {
		// if reach ratelimit, wait one token time and resend job to queue
		time.Sleep(time.Millisecond * time.Duration(w.Rate.Limit()/w.Rate.Bucket()))
		w.Pool.Send(job)
		fmt.Println("Resend job")
		return
	}

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
