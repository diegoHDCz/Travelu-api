// Package httpx contains small HTTP building blocks shared across domains:
// JSON helpers, the standard error envelope and generic middlewares.
package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/diegoczajka/travelu-api/internal/platform/audit"
)

// Error is the single error type handlers and services return when they want
// to control the HTTP status and machine-readable code sent to the client.
// Domain packages build sentinel errors with NewError and the resulting
// *Error is mapped to the response by WriteError, which is the only place in
// the codebase that turns a Go error into an HTTP response.
type Error struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
	// Fields lists every invalid input field; only set on validation errors.
	Fields []FieldError `json:"fields,omitempty"`
}

func (e *Error) Error() string {
	return e.Message
}

// NewError builds a domain sentinel error carrying the HTTP status and code
// it must be translated to.
func NewError(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

type errorEnvelope struct {
	Error *Error `json:"error"`
}

// WriteError serializes err into the standard {"error": {"code", "message"}}
// envelope. Known *Error values keep their status/code; validator errors
// become a 422 validation_error; anything else is logged and hidden behind a
// generic 500 so internal details never leak to clients.
func WriteError(ctx context.Context, w http.ResponseWriter, err error) {
	var appErr *Error
	if errors.As(err, &appErr) {
		audit.SetErrorCode(ctx, appErr.Code)
		WriteJSON(w, appErr.Status, errorEnvelope{Error: appErr})
		return
	}

	if validationErrs, ok := asValidationErrors(err); ok {
		audit.SetErrorCode(ctx, "validation_error")
		WriteJSON(w, http.StatusUnprocessableEntity, errorEnvelope{Error: newValidationError(validationErrs)})
		return
	}

	audit.SetErrorCode(ctx, "internal_error")
	slog.ErrorContext(ctx, "unhandled error", "error", err, "request_id", RequestIDFrom(ctx))
	WriteJSON(w, http.StatusInternalServerError, errorEnvelope{Error: &Error{
		Code:    "internal_error",
		Message: "an unexpected error occurred",
	}})
}

// DecodeAndValidate decodes the JSON request body into dst and validates it
// with the package-level validator instance.
func DecodeAndValidate(r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return decodeError(err)
	}
	if err := validate.Struct(dst); err != nil {
		return err
	}
	return nil
}

// decodeError turns a JSON decoding failure into a 400 that tells the client
// what is wrong (wrong type for a field, body too large, malformed JSON).
func decodeError(err error) *Error {
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) && typeErr.Field != "" {
		return &Error{
			Status:  http.StatusBadRequest,
			Code:    "invalid_body",
			Message: fmt.Sprintf("%s must be of type %s", typeErr.Field, typeErr.Type.String()),
			Fields: []FieldError{{
				Field:   typeErr.Field,
				Rule:    "type",
				Param:   typeErr.Type.String(),
				Message: fmt.Sprintf("%s must be of type %s", typeErr.Field, typeErr.Type.String()),
			}},
		}
	}
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return NewError(http.StatusRequestEntityTooLarge, "body_too_large",
			fmt.Sprintf("request body must be at most %d bytes", maxErr.Limit))
	}
	if errors.Is(err, io.EOF) {
		return NewError(http.StatusBadRequest, "invalid_body", "request body is empty")
	}
	return NewError(http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
}
