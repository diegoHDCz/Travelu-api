package httpx

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

// FieldError describes a single field that failed validation, so clients can
// highlight the exact input and branch on Rule without parsing Message.
type FieldError struct {
	Field   string `json:"field"`
	Rule    string `json:"rule"`
	Param   string `json:"param,omitempty"`
	Message string `json:"message"`
}

var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	// Report fields by their JSON name ("email") instead of the Go one
	// ("Email") so the client sees the same key it sent.
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" {
			return ""
		}
		if name == "" {
			return f.Name
		}
		return name
	})
	return v
}

// newValidationError converts validator errors into the 422 *Error sent to
// clients, with one FieldError per failing field. The top-level Message is
// the first field's message so simple clients can still show something
// useful.
func newValidationError(errs validator.ValidationErrors) *Error {
	fields := make([]FieldError, 0, len(errs))
	for _, fe := range errs {
		fields = append(fields, FieldError{
			Field:   fe.Field(),
			Rule:    fe.Tag(),
			Param:   fieldParam(fe),
			Message: fieldMessage(fe),
		})
	}

	msg := "request validation failed"
	if len(fields) > 0 {
		msg = fields[0].Message
	}
	return &Error{
		Status:  http.StatusUnprocessableEntity,
		Code:    "validation_error",
		Message: msg,
		Fields:  fields,
	}
}

func fieldParam(fe validator.FieldError) string {
	switch fe.Tag() {
	case "gtfield", "gtefield", "ltfield", "ltefield", "eqfield", "nefield", "required_without", "required_with":
		return toSnake(fe.Param())
	}
	return fe.Param()
}

func fieldMessage(fe validator.FieldError) string {
	field, param := fe.Field(), fieldParam(fe)
	isString := fe.Kind() == reflect.String
	isCollection := fe.Kind() == reflect.Slice || fe.Kind() == reflect.Map || fe.Kind() == reflect.Array

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "required_without":
		return fmt.Sprintf("%s is required when %s is not provided", field, param)
	case "required_with":
		return fmt.Sprintf("%s is required when %s is provided", field, param)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "uuid", "uuid4":
		return fmt.Sprintf("%s must be a valid UUID", field)
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, strings.Join(strings.Fields(param), ", "))
	case "len":
		if isString {
			return fmt.Sprintf("%s must be exactly %s characters long", field, param)
		}
		if isCollection {
			return fmt.Sprintf("%s must contain exactly %s items", field, param)
		}
		return fmt.Sprintf("%s must be equal to %s", field, param)
	case "min":
		if isString {
			return fmt.Sprintf("%s must be at least %s characters long", field, param)
		}
		if isCollection {
			return fmt.Sprintf("%s must contain at least %s items", field, param)
		}
		return fmt.Sprintf("%s must be greater than or equal to %s", field, param)
	case "max":
		if isString {
			return fmt.Sprintf("%s must be at most %s characters long", field, param)
		}
		if isCollection {
			return fmt.Sprintf("%s must contain at most %s items", field, param)
		}
		return fmt.Sprintf("%s must be less than or equal to %s", field, param)
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", field, param)
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", field, param)
	case "lt":
		return fmt.Sprintf("%s must be less than %s", field, param)
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", field, param)
	case "gtfield":
		return fmt.Sprintf("%s must be after %s", field, param)
	case "gtefield":
		return fmt.Sprintf("%s must be after or equal to %s", field, param)
	case "ltfield":
		return fmt.Sprintf("%s must be before %s", field, param)
	case "ltefield":
		return fmt.Sprintf("%s must be before or equal to %s", field, param)
	}
	if param != "" {
		return fmt.Sprintf("%s failed the %s=%s rule", field, fe.Tag(), param)
	}
	return fmt.Sprintf("%s failed the %s rule", field, fe.Tag())
}

// toSnake converts a Go field name used as a validator param (e.g.
// "StartDate") to the snake_case JSON key used across the API.
func toSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			r = unicode.ToLower(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}

// asValidationErrors reports whether err carries validator errors.
func asValidationErrors(err error) (validator.ValidationErrors, bool) {
	var ve validator.ValidationErrors
	ok := errors.As(err, &ve)
	return ve, ok
}
