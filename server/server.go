package main

import (
	"log"
	"net/http"

	_ "github.com/jbrady42/h2chat"
	"github.com/r3labs/sse"
)

func main() {

	server := sse.New()
	server.CreateStream("messages")

	// go func(s *sse.Server) {
	// 	var c = 0
	// 	for {
	// 		time.Sleep(time.Second)
	// 		server.Publish("messages", &sse.Event{
	// 			Data: []byte(fmt.Sprintf("tick %d", c)),
	// 		})
	// 		c += 1
	// 	}
	// }(server)

	mux := http.NewServeMux()
	mux.HandleFunc("/events", server.HTTPHandler)
	mux.HandleFunc("/messages", http.HandlerFunc(handleMessage))
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

func handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		http.NotFound(w, r)
		return
	}

	m := h2chat.Message{}
	// Log the request protocol
	log.Printf("Got connection: %s", r.Proto)
	// Send a message back to the client
	w.Write([]byte("Hello"))
}
