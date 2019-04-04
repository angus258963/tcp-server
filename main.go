package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/tcp-server/dcard"
	poolServ "github.com/tcp-server/pool"
)

const (
	CONN_HOST           = "localhost"
	CONN_PORT           = "3333"
	CONN_TYPE           = "tcp"
	RequestTimeousSecs  = 1
	RequestLimit        = 1
	RequestOffset       = 0
	NumberOfConcurrency = 10
)

func main() {
	// Listen for incoming connections.
	l, err := net.Listen(CONN_TYPE, CONN_HOST+":"+CONN_PORT)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("Listening on " + CONN_HOST + ":" + CONN_PORT)

	// new external and handler
	external := dcard.New(RequestTimeousSecs, RequestLimit, RequestOffset)
	externalHandler := func(finished chan bool, job poolServ.Job) {
		posts, err := external.Search(job.Query)
		if err != nil {
			job.Result <- []byte("Error req.Search:" + err.Error())
			finished <- false
			return
		}
		bytes, err := json.Marshal(posts)
		if err != nil {
			job.Result <- []byte("Error josn.Marshal" + err.Error())
			finished <- false
			return
		}

		bytes = append(bytes)
		job.Result <- bytes
		finished <- true
	}

	// init job queue
	jobs := make(chan poolServ.Job, 100)
	pool := &poolServ.Pool{
		Jobs: jobs,
	}
	// create concurrency workers
	for i := 0; i < NumberOfConcurrency; i++ {
		worker := &poolServ.Worker{
			Handler:     externalHandler,
			Pool:        pool,
			TimeoutSecs: 10,
		}
		go func() {
			worker.Listen()
		}()
	}

	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		go handleRequest(conn, pool)
	}
}

// Handles incoming requests.
func handleRequest(conn net.Conn, pool *poolServ.Pool) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	quit := make(chan struct{})
	result := make(chan []byte)
	go func() {
		for {
			conn.Write(<-result)
			conn.Write([]byte("\n"))
		}
	}()
	for scanner.Scan() {
		text := scanner.Text()
		if text == "quit" {
			// TODO: Gracefully shut down
			close(quit)
			conn.Write([]byte("End of connect"))
			return
		}

		// send to worker pool
		pool.Send(poolServ.Job{
			Query:  text,
			Quit:   quit,
			Result: result,
		})
		conn.Write([]byte("Message Received\n"))
	}
}
