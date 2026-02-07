package health

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	s := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	s.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("healthz status = %d, want 200", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Errorf("healthz body = %q, want ok", w.Body.String())
	}
}

func TestReadyz_NotReady(t *testing.T) {
	s := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	s.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("readyz status = %d, want 503", w.Code)
	}
}

func TestReadyz_Ready(t *testing.T) {
	s := NewServer()
	s.SetReady(true)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	s.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("readyz status = %d, want 200", w.Code)
	}
}
