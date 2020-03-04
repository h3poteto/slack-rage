package server

import (
	"fmt"
	"log"
	"net/http"
)

type Server struct {
	Channel string
}

func (s *Server) Serve() error {
	http.HandleFunc("/", s.ServeHTTP)
	log.Println("Listening on :9090")
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		return fmt.Errorf("Failed to start server: %s", err)
	}

	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	event, err := DecodeJSON(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch event.Type() {
	case "url_verification":
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(event.String("challenge")))
		return
	default:
		w.WriteHeader(http.StatusOK)
		return
	}

}
