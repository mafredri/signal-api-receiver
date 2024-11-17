package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kalbasit/signal-receiver/signalapireceiver"
)

const usage = `
GET /receive/pop   => Return the oldest message
GET /receive/flush => Return all messages
`

type Server struct {
	sarc *signalapireceiver.Client
}

func New(sarc *signalapireceiver.Client) *Server {
	return &Server{sarc: sarc}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, "GET is the only allowed verb")
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
	} else if r.URL.Path == "/receive/flush" {
		msgs := s.sarc.Flush()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(msgs); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("ERROR! GET %s is not supported. The supported paths are below:", r.URL.Path) + usage))
	}
}
