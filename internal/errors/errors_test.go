package errors

import (
"errors"
"fmt"
"testing"
)

type mockClient struct {
name string
}

func TestClientError(t *testing.T) {
tests := []struct {
name        string
errType     ErrorType
message     string
client      interface{}
wantString  string
wantMessage string
}{
{
name:        "Connection error",
errType:     ErrConnection,
message:     "failed to connect",
client:      &mockClient{name: "test"},
wantString:  "connection error: failed to connect",
wantMessage: "failed to connect",
},
{
name:        "Validation error",
errType:     ErrValidation,
message:     "invalid input",
client:      nil,
wantString:  "validation error: invalid input",
wantMessage: "invalid input",
},
{
name:        "Broadcast error",
errType:     ErrBroadcast,
message:     "failed to broadcast",
client:      &mockClient{name: "test"},
wantString:  "broadcast error: failed to broadcast",
wantMessage: "failed to broadcast",
},
{
name:        "Concurrency error",
errType:     ErrConcurrency,
message:     "deadlock detected",
client:      nil,
wantString:  "concurrency error: deadlock detected",
wantMessage: "deadlock detected",
},
{
name:        "Nil client",
errType:     ErrConnection,
message:     "error with nil client",
client:      nil,
wantString:  "connection error: error with nil client",
wantMessage: "error with nil client",
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
err := New(tt.errType, tt.message, tt.client)

// Test Error() string
if got := err.Error(); got != tt.wantString {
t.Errorf("Error() = %v, want %v", got, tt.wantString)
}

// Test Message field
if got := err.Message; got != tt.wantMessage {
t.Errorf("Message = %v, want %v", got, tt.wantMessage)
}

// Test Type field
if got := err.Type; got != tt.errType {
t.Errorf("Type = %v, want %v", got, tt.errType)
}

// Test Client field
if got := err.Client; got != tt.client {
t.Errorf("Client = %v, want %v", got, tt.client)
}
})
}
}

func TestErrorTypes(t *testing.T) {
types := []struct {
errType ErrorType
want    string
}{
{ErrConnection, "connection"},
{ErrValidation, "validation"},
{ErrBroadcast, "broadcast"},
{ErrConcurrency, "concurrency"},
}

for _, tt := range types {
t.Run(string(tt.errType), func(t *testing.T) {
if string(tt.errType) != tt.want {
t.Errorf("ErrorType = %v, want %v", tt.errType, tt.want)
}
})
}
}

func TestErrorWrapping(t *testing.T) {
clientErr := &ClientError{
Type:    ErrConnection,
Message: "wrapped error",
Client:  &mockClient{name: "test"},
}

wrappedErr := fmt.Errorf("outer error: %w", clientErr)

// Test error unwrapping
var unwrapped *ClientError
if !errors.As(wrappedErr, &unwrapped) {
t.Error("Failed to unwrap ClientError")
}

if unwrapped.Type != ErrConnection {
t.Errorf("Unwrapped error type = %v, want %v", unwrapped.Type, ErrConnection)
}
}

func TestErrorFormatting(t *testing.T) {
err := New(ErrConnection, "test error", &mockClient{name: "test"})

// Test %v formatting
want := "connection error: test error"
if got := fmt.Sprintf("%v", err); got != want {
t.Errorf("fmt.Sprintf(%%v) = %q, want %q", got, want)
}

// Test %+v formatting (should include client info)
wantVerbose := "connection error: test error (client: test)"
if got := fmt.Sprintf("%+v", err); got != wantVerbose {
t.Errorf("fmt.Sprintf(%%+v) = %q, want %q", got, wantVerbose)
}
}
