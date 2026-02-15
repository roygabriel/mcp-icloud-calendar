// Package health provides HTTP health check and readiness endpoints for
// Kubernetes-style liveness and readiness probes.
package health

import (
	"net/http"
	"sync/atomic"
)

// Server provides HTTP health check endpoints.
type Server struct {
	ready atomic.Bool
	mux   *http.ServeMux
}

// NewServer creates a health server with /healthz and /readyz endpoints.
func NewServer() *Server {
	s := &Server{
		mux: http.NewServeMux(),
	}

	s.mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	s.mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if s.ready.Load() {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not ready"))
		}
	})

	return s
}

// SetReady marks the server as ready to serve traffic.
func (s *Server) SetReady(ready bool) {
	s.ready.Store(ready)
}

// Mux returns the underlying ServeMux for adding extra handlers (e.g. /metrics).
func (s *Server) Mux() *http.ServeMux {
	return s.mux
}
