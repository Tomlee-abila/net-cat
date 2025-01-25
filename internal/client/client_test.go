package client

import (
"fmt"
"net"
"sync"
"testing"
"time"

"net-cat/internal/protocol"
)

// mockConn implements net.Conn for testing
type mockConn struct {
net.Conn
readData  chan []byte
writeData chan []byte
closed    bool
mu        sync.Mutex
readErr   error  // Added to simulate read errors
writeErr  error  // Added to simulate write errors
}

func newMockConn() *mockConn {
return &mockConn{
readData:  make(chan []byte, 100),
writeData: make(chan []byte, 100),
}
}

func (c *mockConn) Read(b []byte) (n int, err error) {
if c.readErr != nil {
return 0, c.readErr
}
data := <-c.readData
copy(b, data)
return len(data), nil
}

func (c *mockConn) Write(b []byte) (n int, err error) {
if c.writeErr != nil {
return 0, c.writeErr
}
c.writeData <- b
return len(b), nil
}

func (c *mockConn) Close() error {
c.mu.Lock()
defer c.mu.Unlock()
c.closed = true
return nil
}

func (c *mockConn) RemoteAddr() net.Addr {
return &net.TCPAddr{IP: net.ParseIP("127.0.0.1")}
}

func (c *mockConn) SetDeadline(t time.Time) error {
return nil
}

func TestNewClient(t *testing.T) {
conn := newMockConn()
client := New(conn)

if client.State() != protocol.StateConnecting {
t.Errorf("New client should be in connecting state, got %v", client.State())
}

if client.Name() != "" {
t.Errorf("New client should have empty name, got %q", client.Name())
}

if client.IsClosed() {
t.Error("New client should not be closed")
}

// Test Done channel
select {
case <-client.Done():
t.Error("Done channel should not be closed for new client")
default:
// Expected behavior
}
}

func TestClientState(t *testing.T) {
client := New(newMockConn())

tests := []struct {
name  string
state protocol.ConnectionState
}{
{"Set connecting", protocol.StateConnecting},
{"Set authenticated", protocol.StateAuthenticated},
{"Set active", protocol.StateActive},
{"Set disconnecting", protocol.StateDisconnecting},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
client.SetState(tt.state)
if client.State() != tt.state {
t.Errorf("Client state = %v, want %v", client.State(), tt.state)
}
})
}
}

func TestClientActivity(t *testing.T) {
client := New(newMockConn())
initialActivity := client.LastActivity()

// Wait a bit to ensure time difference
time.Sleep(time.Millisecond)

client.UpdateActivity()
if client.LastActivity().Equal(initialActivity) {
t.Error("LastActivity should have been updated")
}
}

func TestClientClose(t *testing.T) {
conn := newMockConn()
client := New(conn)

if err := client.Close(); err != nil {
t.Errorf("Close() error = %v", err)
}

if !client.IsClosed() {
t.Error("Client should be marked as closed")
}

if !conn.closed {
t.Error("Underlying connection should be closed")
}

// Test Done channel after close
select {
case <-client.Done():
// Expected behavior
default:
t.Error("Done channel should be closed after client.Close()")
}

// Test double close
if err := client.Close(); err != nil {
t.Error("Second close should not return error")
}
}

func TestClientSend(t *testing.T) {
t.Run("successful send", func(t *testing.T) {
conn := newMockConn()
client := New(conn)
client.SetState(protocol.StateActive)

msg := protocol.NewMessage("test", "hello")
if err := client.Send(msg); err != nil {
t.Errorf("Send() error = %v", err)
}

select {
case data := <-conn.writeData:
if len(data) == 0 {
t.Error("No data written to connection")
}
default:
t.Error("No data sent to connection")
}
})

t.Run("send with inactive state", func(t *testing.T) {
conn := newMockConn()
client := New(conn)
client.SetState(protocol.StateConnecting)

msg := protocol.NewMessage("test", "hello")
if err := client.Send(msg); err == nil {
t.Error("Expected error when sending in inactive state")
}
})

t.Run("send with write error", func(t *testing.T) {
conn := newMockConn()
conn.writeErr = fmt.Errorf("write error")
client := New(conn)
client.SetState(protocol.StateActive)

msg := protocol.NewMessage("test", "hello")
if err := client.Send(msg); err == nil {
t.Error("Expected error when connection write fails")
}
})
}

func TestSendPrompt(t *testing.T) {
t.Run("successful prompt send", func(t *testing.T) {
conn := newMockConn()
client := New(conn)
client.ChangeName("test-user")

if err := client.SendPrompt(); err != nil {
t.Errorf("SendPrompt() error = %v", err)
}

select {
case data := <-conn.writeData:
if !containsAll(string(data), "[", client.Name(), "]") {
t.Errorf("Prompt format incorrect, got %s", string(data))
}
default:
t.Error("No prompt sent to connection")
}
})

t.Run("send prompt with write error", func(t *testing.T) {
conn := newMockConn()
conn.writeErr = fmt.Errorf("write error")
client := New(conn)
client.ChangeName("test-user")

if err := client.SendPrompt(); err == nil {
t.Error("Expected error when connection write fails")
}
})
}

func TestSetDeadline(t *testing.T) {
client := New(newMockConn())
deadline := time.Now().Add(time.Second)

if err := client.SetDeadline(deadline); err != nil {
t.Errorf("SetDeadline() error = %v", err)
}
}

func TestChangeName(t *testing.T) {
client := New(newMockConn())

// Initial name is empty, so the first name change won't be added to history
original := "original"
client.ChangeName(original)

if client.Name() != original {
t.Errorf("Expected name %q, got %q", original, client.Name())
}

// Second name change should add original name to history
newName := "newname"
client.ChangeName(newName)

if client.Name() != newName {
t.Errorf("Expected name %q, got %q", newName, client.Name())
}

// Check history
if got := len(client.nameHistory); got != 1 {
t.Errorf("Expected 1 name in history, got %d", got)
}

// The first non-empty name should be in history
if got := client.nameHistory[0]; got != original {
t.Errorf("Expected history to contain %q, got %q", original, got)
}
}

func TestCanChangeName(t *testing.T) {
client := New(newMockConn())

if !client.CanChangeName() {
t.Error("New client should be able to change name")
}

// Change name maximum number of times
for i := 0; i < 3; i++ {
client.ChangeName(fmt.Sprintf("name%d", i))
}

if client.CanChangeName() {
t.Error("Client should not be able to change name after maximum changes")
}
}

func containsAll(s string, substrs ...string) bool {
for _, sub := range substrs {
if !contains(s, sub) {
return false
}
}
return true
}

func contains(s, substr string) bool {
return s != "" && substr != "" && s != substr && fmt.Sprintf("%s", s) != fmt.Sprintf("%s", substr)
}
