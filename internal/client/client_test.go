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
}

func newMockConn() *mockConn {
return &mockConn{
readData:  make(chan []byte, 100),
writeData: make(chan []byte, 100),
}
}

func (c *mockConn) Read(b []byte) (n int, err error) {
data := <-c.readData
copy(b, data)
return len(data), nil
}

func (c *mockConn) Write(b []byte) (n int, err error) {
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
}

func TestClientSend(t *testing.T) {
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
}

func TestChangeName(t *testing.T) {
client := New(newMockConn())
original := "original"
client.ChangeName(original)

if client.Name() != original {
t.Errorf("Expected name %q, got %q", original, client.Name())
}

newName := "newname"
client.ChangeName(newName)

if client.Name() != newName {
t.Errorf("Expected name %q, got %q", newName, client.Name())
}

if len(client.nameHistory) != 1 {
t.Errorf("Expected 1 name in history, got %d", len(client.nameHistory))
}

if client.nameHistory[0] != original {
t.Errorf("Expected history to contain %q, got %q", original, client.nameHistory[0])
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
