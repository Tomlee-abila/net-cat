package errors

import "fmt"

// ErrorType represents different categories of errors that can occur
type ErrorType string

const (
	ErrConnection  ErrorType = "connection"
	ErrValidation  ErrorType = "validation"
	ErrBroadcast   ErrorType = "broadcast"
	ErrConcurrency ErrorType = "concurrency"
)

// ClientError represents a structured error with context
type ClientError struct {
	Type    ErrorType
	Message string
	Client  interface{} // Using interface{} to avoid import cycles
}

func (e *ClientError) Error() string {
	return fmt.Sprintf("%s error: %s", e.Type, e.Message)
}

// New creates a new ClientError
func New(errType ErrorType, message string, client interface{}) *ClientError {
	return &ClientError{
		Type:    errType,
		Message: message,
		Client:  client,
	}
}
