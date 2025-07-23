package probe

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	ErrorTypeValidation    ErrorType = "validation"
	ErrorTypeExecution     ErrorType = "execution"
	ErrorTypeConfiguration ErrorType = "configuration"
	ErrorTypeNetwork       ErrorType = "network"
	ErrorTypeFile          ErrorType = "file"
	ErrorTypeAction        ErrorType = "action"
	ErrorTypeDependency    ErrorType = "dependency"
)

// ProbeError is the base error type with context information
type ProbeError struct {
	Type      ErrorType
	Operation string
	Message   string
	Cause     error
	Context   map[string]interface{}
}

func (e *ProbeError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s error in %s: %s (caused by: %v)", e.Type, e.Operation, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s error in %s: %s", e.Type, e.Operation, e.Message)
}

func (e *ProbeError) Unwrap() error {
	return e.Cause
}

func (e *ProbeError) Is(target error) bool {
	if t, ok := target.(*ProbeError); ok {
		return e.Type == t.Type
	}
	return false
}

// NewProbeError creates a new ProbeError
func NewProbeError(errorType ErrorType, operation, message string, cause error) *ProbeError {
	return &ProbeError{
		Type:      errorType,
		Operation: operation,
		Message:   message,
		Cause:     cause,
		Context:   make(map[string]interface{}),
	}
}

// WithContext adds context information to the error
func (e *ProbeError) WithContext(key string, value interface{}) *ProbeError {
	e.Context[key] = value
	return e
}

// ValidationError for validation-specific errors
type ValidationError struct {
	messages []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error:\n%s", strings.Join(e.messages, "\n"))
}

func (e *ValidationError) HasError() bool {
	return len(e.messages) > 0
}

func (e *ValidationError) AddMessage(s string) {
	e.messages = append(e.messages, s)
}

// Convenience functions for creating specific error types
func NewValidationError(operation, message string, cause error) *ProbeError {
	return NewProbeError(ErrorTypeValidation, operation, message, cause)
}

func NewExecutionError(operation, message string, cause error) *ProbeError {
	return NewProbeError(ErrorTypeExecution, operation, message, cause)
}

func NewConfigurationError(operation, message string, cause error) *ProbeError {
	return NewProbeError(ErrorTypeConfiguration, operation, message, cause)
}

func NewNetworkError(operation, message string, cause error) *ProbeError {
	return NewProbeError(ErrorTypeNetwork, operation, message, cause)
}

func NewFileError(operation, message string, cause error) *ProbeError {
	return NewProbeError(ErrorTypeFile, operation, message, cause)
}

func NewActionError(operation, message string, cause error) *ProbeError {
	return NewProbeError(ErrorTypeAction, operation, message, cause)
}

func NewDependencyError(operation, message string, cause error) *ProbeError {
	return NewProbeError(ErrorTypeDependency, operation, message, cause)
}

// IsErrorType checks if an error is of a specific type
func IsErrorType(err error, errorType ErrorType) bool {
	var probeErr *ProbeError
	if errors.As(err, &probeErr) {
		return probeErr.Type == errorType
	}
	return false
}
