package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/jbrady42/h2chat"
	"github.com/r3labs/sse"
)

type h2Server struct {
	sse      *sse.Server
	channels []string
}

func main() {

	channels := []string{"default", "other"}
	sseServer := sse.New()

	server := h2Server{
		sseServer,
		channels,
	}

	for _, a := range channels {
		sseServer.CreateStream(a)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/events", sseServer.HTTPHandler)
	mux.HandleFunc("/topics", http.HandlerFunc(server.handleListChannels))
	mux.HandleFunc("/messages", http.HandlerFunc(server.handleMessage))
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

func (t *h2Server) handleListChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.NotFound(w, r)
		return
	}
	res, err := json.Marshal(t.channels)
	if err != nil {
		log.Fatalf("Failed parsing message body: %s", err)
	}
	w.Write(res)
}

func (t *h2Server) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" && r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatalf("Failed reading message body: %s", err)
	}

	var msg h2chat.Message
	err = json.Unmarshal(body, &msg)
	if err != nil {
		log.Fatalf("Failed parsing message body: %s", err)
	}

	// log.Println(msg)
	// // Log the request protocol
	// log.Printf("Got connection: %s", r.Proto)
	// Send a message back to the client
	w.Write([]byte("OK"))

	if t.channelExists(msg.Topic) {
		t.sse.Publish(msg.Topic, &sse.Event{
			Data: body,
		})
	}
}

func (t *h2Server) channelExists(str string) bool {
	for _, a := range t.channels {
		if a == str {
			return true
		}
	}
	return false
}
