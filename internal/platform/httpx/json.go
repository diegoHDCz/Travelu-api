package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// WriteJSON writes v as a JSON response with the given status code. A nil v
// writes only the status line, with no body and no Content-Type header
// (useful for 204 No Content responses).
func WriteJSON(w http.ResponseWriter, status int, v any) {
	if v == nil {
		w.WriteHeader(status)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode json response", "error", err)
	}
}
