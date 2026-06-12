package errdefs

import (
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type ResponseError struct {
	Status  int      `json:"status"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
}

func (e *ResponseError) Error() string {
	return e.Message
}

func (e *ResponseError) GetStatus() int {
	return e.Status
}

var HumaErrorFunc = func(status int, message string, errs ...error) huma.StatusError {
	details := make([]string, len(errs))
	for i, err := range errs {
		details[i] = err.Error()
	}
	return &ResponseError{
		Status:  status,
		Message: message,
		Details: details,
	}
}

type ErrorType int

const (
	ErrTypeInvalidInput ErrorType = iota
	ErrTypeNotFound
	ErrTypeConflict
	ErrTypeUnauthorized
	ErrTypeProvider
	ErrTypeUnsupported
)

var errorTypeStrings = map[ErrorType]string{
	ErrTypeInvalidInput: "ErrInvalidInput",
	ErrTypeNotFound:     "ErrNotFound",
	ErrTypeConflict:     "ErrConflict",
	ErrTypeUnauthorized: "ErrUnauthorized",
	ErrTypeProvider:     "ErrProvider",
	ErrTypeUnsupported:  "ErrUnsupported",
}

func (e ErrorType) String() string {
	if s, ok := errorTypeStrings[e]; ok {
		return s
	}
	return "ErrUnknown"
}

type CustomError struct {
	Type    ErrorType
	Message string
}

func (e *CustomError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type.String(), e.Message)
}

func (e *CustomError) GetStatus() int {
	switch e.Type {
	case ErrTypeInvalidInput:
		return http.StatusBadRequest
	case ErrTypeNotFound:
		return http.StatusNotFound
	case ErrTypeConflict:
		return http.StatusConflict
	case ErrTypeUnauthorized:
		return http.StatusUnauthorized
	case ErrTypeUnsupported:
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}

func (e *CustomError) Is(target error) bool {
	t, ok := target.(*CustomError)
	if !ok {
		return false
	}
	return e.Type == t.Type
}

func NewCustomError(t ErrorType, message string) *CustomError {
	return &CustomError{Type: t, Message: message}
}

var (
	ErrInvalidInput = NewCustomError(ErrTypeInvalidInput, "")
	ErrNotFound     = NewCustomError(ErrTypeNotFound, "")
	ErrConflict     = NewCustomError(ErrTypeConflict, "")
	ErrUnauthorized = NewCustomError(ErrTypeUnauthorized, "")
	ErrProvider     = NewCustomError(ErrTypeProvider, "")
	ErrUnsupported  = NewCustomError(ErrTypeUnsupported, "")
)
