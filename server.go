package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/angus258963/tcp-server/dcard"
	poolServ "github.com/angus258963/tcp-server/pool"
	"github.com/angus258963/tcp-server/ratelimit"
)

var host = flag.String("host", "localhost", "The host to listen to; default is localhost")
var port = flag.Int("port", 3333, "The port to listen on; default is 3333.")

const (
	CONN_TYPE           = "tcp"
	RequestTimeousSecs  = 1
	RequestLimit        = 1
	RequestOffset       = 0
	NumberOfConcurrency = 10
	ConnTimeoutSecs     = 60
	RateLimitBuckets    = 30
	RateLimitMs         = 1000
	ConnectionLimit     = 100
)

func main() {
	flag.Parse()

	// Listen for incoming connections.
	l, err := net.Listen(CONN_TYPE, *host+":"+strconv.Itoa(*port))
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("Listening on " + *host + ":" + strconv.Itoa(*port))

	// new external and handler
	external := dcard.New(RequestTimeousSecs, RequestLimit, RequestOffset)
	externalHandler := func(finished chan []byte, job poolServ.Job) {
		posts, err := external.Search(job.Query)
		if err != nil {
			finished <- []byte("Error req.Search:" + err.Error())
			return
		}
		bytes, err := json.Marshal(posts)
		if err != nil {
			finished <- []byte("Error josn.Marshal" + err.Error())
			return
		}
		finished <- bytes
	}

	// init connect queue
	connQueue := make(chan struct{}, ConnectionLimit)

	// init pool to handle external requests
	rate := ratelimit.New(RateLimitBuckets, RateLimitMs)
	pool := poolServ.New(100, NumberOfConcurrency, rate, externalHandler)

	// listen for http requests
	go httpServer(pool, connQueue)

	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}

		// send one conn to connQueue, if queue is full then close the connection
		select {
		case connQueue <- struct{}{}:
			conn.Write([]byte("Connected \n"))
		default:
			conn.Write([]byte("Server overloading come back later \n"))
			conn.Close()
			continue
		}
		// Handle connections in a new goroutine.
		conn.SetReadDeadline(time.Now().Add(time.Second * ConnTimeoutSecs))
		go handleRequest(conn, pool, connQueue)
	}
}

// Handles incoming requests.
func handleRequest(conn net.Conn, pool *poolServ.Pool, connQueue chan struct{}) {
	defer func() {
		conn.Close()
		<-connQueue // pop one from conn queue
	}()

	scanner := bufio.NewScanner(conn)
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
			conn.Write([]byte("start to end the connection ...\n"))
			break
		}

		// send to worker pool
		pool.Send(poolServ.Job{
			Query:  text,
			Result: result,
		})
		conn.Write([]byte("Message Received\n"))
	}

	if err := scanner.Err(); err != nil {
		conn.Write([]byte("Scanner error:" + err.Error() + "\n"))
	}
	conn.Write([]byte("Gracefully shut down\n"))
	time.Sleep(time.Second * 10)
	conn.Write([]byte("End of connect\n"))
}

func httpServer(pool *poolServ.Pool, connQueue chan struct{}) {
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		res := fmt.Sprintf("server status:\n")
		res += fmt.Sprintf("number of connections: %d\n", len(connQueue))
		res += fmt.Sprintf("number of processing jobs: %d\n", pool.GetNumberOfRunningJobs())
		res += fmt.Sprintf("number of remaing jobs: %d\n", pool.GetNumberOfJobs())
		fmt.Fprintf(w, res)
	})

	http.ListenAndServe(":80", nil)
}
