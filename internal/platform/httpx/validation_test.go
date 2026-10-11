package httpx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type validationInput struct {
	Name      string    `json:"name" validate:"required,min=2"`
	Email     string    `json:"email" validate:"omitempty,email,max=10"`
	Kind      string    `json:"kind" validate:"omitempty,oneof=A B"`
	Count     int       `json:"count" validate:"omitempty,gt=0"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date" validate:"omitempty,gtfield=StartDate"`
}

func decodeAndWrite(t *testing.T, body string) (int, errorEnvelope) {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	w := httptest.NewRecorder()
	var in validationInput
	if err := DecodeAndValidate(r, &in); err != nil {
		WriteError(context.Background(), w, err)
	}
	var env errorEnvelope
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return w.Code, env
}

func TestDecodeAndValidate_FieldErrors(t *testing.T) {
	status, env := decodeAndWrite(t, `{"email":"averylongemail@example.com","kind":"C","count":0,
		"start_date":"2026-01-02T00:00:00Z","end_date":"2026-01-01T00:00:00Z"}`)

	if status != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", status)
	}
	if env.Error.Code != "validation_error" {
		t.Fatalf("code = %q", env.Error.Code)
	}

	want := map[string]FieldError{
		"name":     {Field: "name", Rule: "required", Message: "name is required"},
		"email":    {Field: "email", Rule: "max", Param: "10", Message: "email must be at most 10 characters long"},
		"kind":     {Field: "kind", Rule: "oneof", Param: "A B", Message: "kind must be one of: A, B"},
		"end_date": {Field: "end_date", Rule: "gtfield", Param: "start_date", Message: "end_date must be after start_date"},
	}
	got := map[string]FieldError{}
	for _, f := range env.Error.Fields {
		got[f.Field] = f
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("field %s = %+v, want %+v", k, got[k], w)
		}
	}
	if env.Error.Message != env.Error.Fields[0].Message {
		t.Errorf("message = %q, want first field message", env.Error.Message)
	}
}

func TestDecodeAndValidate_TypeError(t *testing.T) {
	status, env := decodeAndWrite(t, `{"name":"ok","count":"three"}`)
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", status)
	}
	if env.Error.Message != "count must be of type int" || len(env.Error.Fields) != 1 || env.Error.Fields[0].Field != "count" {
		t.Errorf("unexpected error: %+v", env.Error)
	}
}

func TestDecodeAndValidate_EmptyBody(t *testing.T) {
	status, env := decodeAndWrite(t, ``)
	if status != http.StatusBadRequest || env.Error.Message != "request body is empty" {
		t.Errorf("got %d %+v", status, env.Error)
	}
}
