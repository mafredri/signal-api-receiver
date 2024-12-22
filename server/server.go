package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/kalbasit/signal-api-receiver/receiver"
)

const usage = `
GET /receive/pop   => Return the oldest message
GET /receive/flush => Return all messages
`

// Server represent the HTTP server that exposes the pop/flush routes
type Server struct {
	sarc client
}

type client interface {
	Connect() error
	ReceiveLoop() error
	Pop() *receiver.Message
	Flush() []receiver.Message
}

// New returns a new Server
func New(sarc client) *Server {
	s := &Server{sarc: sarc}
	go s.start()
	return s
}

func (s *Server) start() {
	for {
		if err := s.sarc.ReceiveLoop(); err != nil {
			log.Printf("Error in the receive loop: %v", err)
		}
	Reconnect:
		if err := s.sarc.Connect(); err != nil {
			log.Printf("Error reconnecting: %v", err)
			time.Sleep(time.Second)
			goto Reconnect
		}
	}
}

// ServeHTTP implements the http.Handler interface
//
// /receive/pop
//
//	This returns status 200 and a receiver.Message or status 204 with no body
//
// /receive/flush
//
//	This returns status 200 and a list of receiver.Message
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, "GET is the only allowed verb")
		return
	}

	if r.URL.Path == "/healthz" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.URL.Path == "/receive/pop" {
		msg := s.sarc.Pop()
		if msg == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(msg); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}

		return
	}

	if r.URL.Path == "/receive/flush" {
		msgs := s.sarc.Flush()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(msgs); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(fmt.Sprintf("ERROR! GET %s is not supported. The supported paths are below:", r.URL.Path) + usage))
}
