package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/andrewburian/eventsource"
)

func main() {

	stream := eventsource.NewStream()

	go func(s *eventsource.Stream) {
		var c = 0
		for {
			time.Sleep(time.Second)
			stream.Broadcast(eventsource.DataEvent(fmt.Sprintf("tick %d", c)))
			c += 1
		}
	}(stream)

	mux := http.NewServeMux()
	mux.Handle("/stream1", stream)
	mux.HandleFunc("/", http.HandlerFunc(handle))

	// Create a server on port 8000
	// Exactly how you would run an HTTP/1.1 server
	srv := &http.Server{Addr: ":8000", Handler: mux}

	// Start the server with TLS, since we are running HTTP/2 it must be
	// run with TLS.
	// Exactly how you would run an HTTP/1.1 server with TLS connection.
	log.Printf("Serving on https://0.0.0.0:8000")
	log.Fatal(srv.ListenAndServeTLS("certs/server.crt", "certs/server.key"))
}

func handle(w http.ResponseWriter, r *http.Request) {
	// Log the request protocol
	log.Printf("Got connection: %s", r.Proto)
	// Send a message back to the client
	w.Write([]byte("Hello"))
}
