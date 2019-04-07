package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/tcp-server/dcard"
	poolServ "github.com/tcp-server/pool"
	"github.com/tcp-server/ratelimit"
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

	// init pool to handle external requests
	rate := ratelimit.New(RateLimitBuckets, RateLimitMs)
	pool := poolServ.New(100, NumberOfConcurrency, rate, externalHandler)
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}

		conn.SetReadDeadline(time.Now().Add(time.Second * ConnTimeoutSecs))
		// Handle connections in a new goroutine.
		go handleRequest(conn, pool)
	}
}

// Handles incoming requests.
func handleRequest(conn net.Conn, pool *poolServ.Pool) {
	defer conn.Close()

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
