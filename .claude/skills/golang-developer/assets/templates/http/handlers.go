package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func (s *Server) handleIndex() http.HandlerFunc {
	type response struct {
		Message string `json:"message"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		s.respond(w, r, response{Message: "Welcome to the API"}, http.StatusOK)
	}
}

func (s *Server) handleHealth() http.HandlerFunc {
	type response struct {
		Status string `json:"status"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		s.respond(w, r, response{Status: "ok"}, http.StatusOK)
	}
}

// Helper methods

func (s *Server) respond(w http.ResponseWriter, r *http.Request, data any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			s.logger.Error("failed to encode response",
				slog.String("error", err.Error()),
			)
		}
	}
}

func (s *Server) decode(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

type errorResponse struct {
	Error string `json:"error"`
}

func (s *Server) respondError(w http.ResponseWriter, r *http.Request, err error, status int) {
	s.respond(w, r, errorResponse{Error: err.Error()}, status)
}
