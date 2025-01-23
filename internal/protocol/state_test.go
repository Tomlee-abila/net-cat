package protocol

import (
"fmt"
"testing"
)

func TestConnectionState(t *testing.T) {
tests := []struct {
name  string
state ConnectionState
want  string
}{
{
name:  "Connecting state",
state: StateConnecting,
want:  "Connecting",
},
{
name:  "Authenticated state",
state: StateAuthenticated,
want:  "Authenticated",
},
{
name:  "Active state",
state: StateActive,
want:  "Active",
},
{
name:  "Disconnecting state",
state: StateDisconnecting,
want:  "Disconnecting",
},
{
name:  "Invalid state",
state: ConnectionState(99),
want:  "Unknown",
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
if got := tt.state.String(); got != tt.want {
t.Errorf("ConnectionState.String() = %v, want %v", got, tt.want)
}
})
}
}

func TestConnectionStateOrder(t *testing.T) {
// Test that states are properly ordered
if StateConnecting >= StateAuthenticated {
t.Error("StateConnecting should be less than StateAuthenticated")
}

if StateAuthenticated >= StateActive {
t.Error("StateAuthenticated should be less than StateActive")
}

if StateActive >= StateDisconnecting {
t.Error("StateActive should be less than StateDisconnecting")
}
}

func TestConnectionStateTransitions(t *testing.T) {
transitions := []struct {
from ConnectionState
to   ConnectionState
valid bool
}{
{StateConnecting, StateAuthenticated, true},
{StateAuthenticated, StateActive, true},
{StateActive, StateDisconnecting, true},
{StateConnecting, StateActive, false},     // Can't skip authentication
{StateAuthenticated, StateConnecting, false}, // Can't go backwards
{StateActive, StateConnecting, false},      // Can't go backwards
{StateDisconnecting, StateActive, false},   // Can't reactivate
}

for _, tt := range transitions {
t.Run(fmt.Sprintf("%s->%s", tt.from, tt.to), func(t *testing.T) {
// Test if transition follows expected rules
isValid := isValidTransition(tt.from, tt.to)
if isValid != tt.valid {
t.Errorf("Transition from %s to %s: got validity %v, want %v",
tt.from, tt.to, isValid, tt.valid)
}
})
}
}

// Helper function to validate state transitions
func isValidTransition(from, to ConnectionState) bool {
switch from {
case StateConnecting:
return to == StateAuthenticated
case StateAuthenticated:
return to == StateActive
case StateActive:
return to == StateDisconnecting
default:
return false
}
}
