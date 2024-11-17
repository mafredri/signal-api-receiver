package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kalbasit/signal-receiver/signalapireceiver"
)

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

	msgs := s.sarc.Flush()
	if err := json.NewEncoder(w).Encode(msgs); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
