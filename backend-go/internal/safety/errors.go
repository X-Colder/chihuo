package safety

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrNotFound          = errors.New("safety resource not found")
	ErrConflict          = errors.New("safety resource conflict")
	ErrInvalidState      = errors.New("invalid safety resource state")
	ErrInvalidTransition = errors.New("invalid safety state transition")
	ErrValidation        = errors.New("safety validation failed")
)

type FieldError struct {
	Field   string
	Message string
}

type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	if e == nil || len(e.Fields) == 0 {
		return ErrValidation.Error()
	}
	parts := make([]string, 0, len(e.Fields))
	for _, field := range e.Fields {
		parts = append(parts, fmt.Sprintf("%s: %s", field.Field, field.Message))
	}
	return strings.Join(parts, "; ")
}

func (e *ValidationError) Unwrap() error {
	return ErrValidation
}

func validationError(fields ...FieldError) error {
	return &ValidationError{Fields: fields}
}

func invalidTransition(from, to string) error {
	return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, from, to)
}
