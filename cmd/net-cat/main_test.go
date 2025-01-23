package main

import (
"bytes"
"fmt"
"io"
"os"
"strings"
"testing"
"time"
)

func TestArgumentValidation(t *testing.T) {
// Save original args and restore after test
oldArgs := os.Args
oldExit := osExit
defer func() {
os.Args = oldArgs
osExit = oldExit
}()

var exitCode int
osExit = func(code int) {
exitCode = code
}

tests := []struct {
name      string
args      []string
wantUsage bool
}{
{
name:      "Valid single argument",
args:      []string{"TCPChat", "2525"},
wantUsage: false,
},
{
name:      "Too many arguments",
args:      []string{"TCPChat", "2525", "localhost"},
wantUsage: true,
},
{
name:      "No arguments (default port)",
args:      []string{"TCPChat"},
wantUsage: false,
},
{
name:      "Invalid port",
args:      []string{"TCPChat", "invalid"},
wantUsage: true,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
exitCode = 0
os.Args = tt.args

// Capture stdout
oldStdout := os.Stdout
r, w, _ := os.Pipe()
os.Stdout = w

// Run main in goroutine as it might exit
go main()

w.Close()
os.Stdout = oldStdout

var buf bytes.Buffer
io.Copy(&buf, r)
output := buf.String()

if tt.wantUsage {
if !strings.Contains(output, "[USAGE]") {
t.Error("Expected usage message not shown")
}
if exitCode != 1 {
t.Errorf("Expected exit code 1, got %d", exitCode)
}
} else {
if exitCode != 0 && !strings.Contains(output, "Starting TCP Chat server") {
t.Errorf("Unexpected exit with code %d", exitCode)
}
}
})
}
}

func TestMultipleArgumentHandling(t *testing.T) {
// Save original args and exit function
oldArgs := os.Args
oldExit := osExit
defer func() {
os.Args = oldArgs
osExit = oldExit
}()

// Create channel to capture exit code
exitCalled := make(chan int, 1)
osExit = func(code int) {
exitCalled <- code
}

// Capture stdout
oldStdout := os.Stdout
r, w, _ := os.Pipe()
os.Stdout = w
defer func() {
os.Stdout = oldStdout
}()

// Set test arguments
os.Args = []string{"TCPChat", "2525", "localhost"}

// Run main (in a goroutine since it might exit)
go main()

// Close writer to get output
w.Close()

// Read captured output
var buf bytes.Buffer
io.Copy(&buf, r)
output := buf.String()

// Check exit code
select {
case code := <-exitCalled:
if code != 1 {
t.Errorf("Expected exit code 1, got %d", code)
}
case <-time.After(time.Second):
t.Error("Program did not exit as expected")
}

// Verify usage message
expectedUsage := "[USAGE]: ./TCPChat $port"
if !strings.Contains(output, expectedUsage) {
t.Errorf("Expected output to contain %q, got %q", expectedUsage, output)
}
}

func TestDefaultPort(t *testing.T) {
// Save original args and exit function
oldArgs := os.Args
oldExit := osExit
defer func() {
os.Args = oldArgs
osExit = oldExit
}()

osExit = func(code int) {
// Override exit for testing
}

// Capture stdout
oldStdout := os.Stdout
r, w, _ := os.Pipe()
os.Stdout = w
defer func() {
os.Stdout = oldStdout
}()

// Test with no arguments
os.Args = []string{"TCPChat"}

// Run main in goroutine as it starts a server
go main()

// Give it time to start
time.Sleep(100 * time.Millisecond)

w.Close()

var buf bytes.Buffer
io.Copy(&buf, r)
output := buf.String()

// Verify default port is used
expectedPort := "8989"
if !strings.Contains(output, fmt.Sprintf("Starting TCP Chat server on port %s", expectedPort)) {
t.Errorf("Expected to use default port %s, got output: %s", expectedPort, output)
}
}
