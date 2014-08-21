package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

type Broker struct {
	clients  map[chan []byte]bool
	messages chan []byte
}

// Processing should be called concurrently,
// leveraging the messages channel
func (b *Broker) Process() {
	for {
		select {
		case msg := <-b.messages:
			for s, _ := range b.clients {
				s <- msg
			}

			log.Printf("Broadcasted to %d clients...", len(b.clients))
		}
	}
}

// Add a new client to clients map
func (b *Broker) Add(c chan []byte) {
	b.clients[c] = true
	log.Println("Added new client:", c)
}

func (b *Broker) Remove(c chan []byte) {
	delete(b.clients, c)
	log.Println("Removed client:", c)
}

// Send msg to all clients
func (b *Broker) Send(msg []byte) {
	b.messages <- msg
}

// http handler for /events/
func (b *Broker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f, ok := w.(http.Flusher)

	if !ok {
		http.Error(w, "Flush is not supported", http.StatusInternalServerError)
	}

	n, ok := w.(http.CloseNotifier)

	if !ok {
		http.Error(w, "Close Notifier not supported", http.StatusInternalServerError)
	}

	c := make(chan []byte)
	b.Add(c)

	defer b.Remove(c)

	// Set all associated headers for sse
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/event-stream")

	t := time.Now()
	br := false

	for {
		select {
		case <-n.CloseNotify():
			br = true

		case msg := <-c:
			fmt.Fprintf(w, "data: %s\n\n", msg)

			f.Flush()

		default:
			if time.Since(t).Seconds() > 30 {
				t = time.Now()
				msg := []byte(strconv.FormatInt(t.Unix(), 10))

				b.Send(msg)
			}
		}

		if br {
			break
		}
	}
}
