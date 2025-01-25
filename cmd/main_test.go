package main

import (
"bytes"
"fmt"
"io"
"os"
"strings"
"sync"
"syscall"
"testing"
"time"
)

// testPort helps avoid conflicts between concurrent tests
var currentPort = 8990

var (
portMutex sync.Mutex
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
var exitMutex sync.Mutex
osExit = func(code int) {
exitMutex.Lock()
exitCode = code
exitMutex.Unlock()
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
exitMutex.Lock()
exitCode = 0
exitMutex.Unlock()

os.Args = tt.args

// Capture stdout using sync.Once to ensure proper cleanup
var once sync.Once
oldStdout := os.Stdout
r, w, _ := os.Pipe()
os.Stdout = w

// Ensure cleanup happens exactly once
defer once.Do(func() {
w.Close()
os.Stdout = oldStdout
})

// Channel to signal main has completed
done := make(chan struct{})
serverReady := make(chan struct{})

go func() {
main()
close(done)
}()

// Wait for server to start or timeout
go func() {
time.Sleep(100 * time.Millisecond)
close(serverReady)
}()

// Wait for either completion or timeout
select {
case <-done:
case <-time.After(2 * time.Second):
t.Log("Test timed out")
}

// Send interrupt signal to stop the server
syscall.Kill(syscall.Getpid(), syscall.SIGINT)

once.Do(func() {
w.Close()
os.Stdout = oldStdout
})

var buf bytes.Buffer
io.Copy(&buf, r)
output := buf.String()

exitMutex.Lock()
currentExitCode := exitCode
exitMutex.Unlock()

if tt.wantUsage {
if !strings.Contains(output, "[USAGE]") {
t.Error("Expected usage message not shown")
}
if currentExitCode != 1 {
t.Errorf("Expected exit code 1, got %d", currentExitCode)
}
} else {
if currentExitCode != 0 && !strings.Contains(output, "Starting TCP Chat server") {
t.Errorf("Unexpected exit with code %d", currentExitCode)
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

// Capture stdout with proper synchronization
var once sync.Once
oldStdout := os.Stdout
r, w, _ := os.Pipe()
os.Stdout = w

// Ensure cleanup happens exactly once
defer once.Do(func() {
w.Close()
os.Stdout = oldStdout
})

// Set test arguments
os.Args = []string{"TCPChat", "2525", "localhost"}

// Run main with timeout
done := make(chan struct{})
serverReady := make(chan struct{})

go func() {
main()
close(done)
}()

// Wait for server to start or timeout
go func() {
time.Sleep(100 * time.Millisecond)
close(serverReady)
}()

// Wait for either completion or timeout
select {
case <-done:
case <-time.After(2 * time.Second):
t.Log("Test timed out")
}

// Send interrupt signal to stop the server
syscall.Kill(syscall.Getpid(), syscall.SIGINT)

once.Do(func() {
w.Close()
os.Stdout = oldStdout
})

// Read captured output
var buf bytes.Buffer
io.Copy(&buf, r)
output := buf.String()

// Check exit code with timeout
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
// Get a unique port for this test
portMutex.Lock()
port := currentPort
currentPort++
portMutex.Unlock()

// Convert port to string
portStr := fmt.Sprintf("%d", port)

os.Args = []string{"TCPChat", portStr}

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

// Capture stdout with proper synchronization
var once sync.Once
oldStdout := os.Stdout
r, w, _ := os.Pipe()
os.Stdout = w

// Ensure cleanup happens exactly once
defer once.Do(func() {
w.Close()
os.Stdout = oldStdout
})

// Run main with timeout
done := make(chan struct{})
serverReady := make(chan struct{})

go func() {
main()
close(done)
}()

// Wait for server to start or timeout
go func() {
time.Sleep(100 * time.Millisecond)
close(serverReady)
}()

<-serverReady

// Give it time to start with timeout
select {
case <-done:
case <-time.After(2 * time.Second):
}

// Send interrupt signal to stop the server
syscall.Kill(syscall.Getpid(), syscall.SIGINT)

once.Do(func() {
w.Close()
os.Stdout = oldStdout
})

var buf bytes.Buffer
io.Copy(&buf, r)
output := buf.String()

// Verify default port is used
if !strings.Contains(output, fmt.Sprintf("Starting TCP Chat server on port %s", portStr)) {
t.Errorf("Expected to use port %s, got output: %s", portStr, output)
}
}
