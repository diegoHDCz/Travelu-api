package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func serve(t *testing.T, mux *http.ServeMux, req *http.Request) map[string]any {
	t.Helper()
	var buf bytes.Buffer
	h := Middleware(Config{
		Logger:    slog.New(slog.NewJSONHandler(&buf, nil)),
		RequestID: func(context.Context) string { return "req-1" },
		ClientIP:  func(*http.Request) string { return "203.0.113.7" },
		Skip:      func(r *http.Request) bool { return r.URL.Path == "/health" },
	})(mux)

	func() {
		defer func() { _ = recover() }()
		h.ServeHTTP(httptest.NewRecorder(), req)
	}()

	if buf.Len() == 0 {
		return nil
	}
	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("decode audit line %q: %v", buf.String(), err)
	}
	return line
}

func TestMiddleware_RecordsActionActorAndResource(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("PUT /things/{id}", Action("thing.update", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetActor(r.Context(), "user-1")
		Add(r.Context(), slog.String("reason", "because"))
		w.WriteHeader(http.StatusOK)
	})))

	line := serve(t, mux, httptest.NewRequest(http.MethodPut, "/things/abc", nil))

	want := map[string]any{
		"msg": "audit", "log_type": "audit", "action": "thing.update", "outcome": "success",
		"status": float64(200), "actor_id": "user-1", "resource_id": "abc", "route": "PUT /things/{id}",
		"ip": "203.0.113.7", "request_id": "req-1", "reason": "because",
	}
	for k, v := range want {
		if line[k] != v {
			t.Errorf("%s = %v, want %v", k, line[k], v)
		}
	}
}

func TestMiddleware_DeniedAndErrorCode(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("GET /secret", Action("secret.get", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetErrorCode(r.Context(), "unauthorized")
		w.WriteHeader(http.StatusUnauthorized)
	})))

	line := serve(t, mux, httptest.NewRequest(http.MethodGet, "/secret", nil))
	if line["outcome"] != "denied" || line["error_code"] != "unauthorized" {
		t.Errorf("unexpected line: %v", line)
	}
}

func TestMiddleware_PanicStillAudited(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("GET /boom", Action("boom", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})))

	line := serve(t, mux, httptest.NewRequest(http.MethodGet, "/boom", nil))
	if line["status"] != float64(500) || line["outcome"] != "failure" {
		t.Errorf("unexpected line: %v", line)
	}
}

func TestMiddleware_SkipAndUnmatched(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(http.ResponseWriter, *http.Request) {})

	if line := serve(t, mux, httptest.NewRequest(http.MethodGet, "/health", nil)); line != nil {
		t.Errorf("health should not be audited, got %v", line)
	}
	if line := serve(t, mux, httptest.NewRequest(http.MethodGet, "/nope", nil)); line["action"] != "unmatched_route" || line["status"] != float64(404) {
		t.Errorf("unexpected line: %v", line)
	}
}
