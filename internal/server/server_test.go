package server

import (
"fmt"
"net"
"testing"
"time"

"net-cat/internal/client"
"net-cat/internal/config"
"net-cat/internal/protocol"
)

// Helper function to create a test server with custom config
func createTestServer(cfg *config.Config) (*Server, error) {
if cfg == nil {
cfg = config.DefaultConfig().WithListenAddr(":0") // Let OS assign port
}
return New(cfg), nil
}

func TestServerLifecycle(t *testing.T) {
srv, err := createTestServer(nil)
if err != nil {
t.Fatalf("Failed to create test server: %v", err)
}

errChan := make(chan error, 1)
go func() {
errChan <- srv.Start()
}()

// Give server time to start
time.Sleep(100 * time.Millisecond)

// Try to connect
addr := srv.ln.Addr().String()
conn, err := net.Dial("tcp", addr)
if err != nil {
t.Fatalf("Failed to connect to server: %v", err)
}
defer conn.Close()

// Verify server accepts connection
time.Sleep(100 * time.Millisecond)

// Stop server
if err := srv.Stop(); err != nil {
t.Fatalf("Failed to stop server: %v", err)
}

// Verify server has stopped
select {
case err := <-errChan:
if err != nil {
t.Errorf("Server.Start() returned unexpected error: %v", err)
}
case <-time.After(time.Second):
t.Error("Timeout waiting for server to stop")
}
}

func TestServerClientLimit(t *testing.T) {
maxClients := 2
cfg := config.DefaultConfig().
WithListenAddr(":0").
WithMaxClients(maxClients)

srv, err := createTestServer(cfg)
if err != nil {
t.Fatalf("Failed to create test server: %v", err)
}

// Start server
go srv.Start()
defer srv.Stop()

// Wait for server to start
time.Sleep(100 * time.Millisecond)

addr := srv.ln.Addr().String()
conns := make([]net.Conn, 0, maxClients+1)

// Try to connect more clients than allowed
for i := 0; i < maxClients+1; i++ {
conn, err := net.Dial("tcp", addr)
if err != nil {
t.Fatalf("Failed to connect to server: %v", err)
}
defer conn.Close()
conns = append(conns, conn)

// Give server time to process connection
time.Sleep(100 * time.Millisecond)
}

// Verify client count
srv.clientsMu.RLock()
clientCount := len(srv.clients)
srv.clientsMu.RUnlock()

if clientCount > maxClients {
t.Errorf("Server accepted more clients than maximum: got %d, want <= %d", clientCount, maxClients)
}
}

func TestBroadcastMessage(t *testing.T) {
srv, err := createTestServer(nil)
if err != nil {
t.Fatalf("Failed to create test server: %v", err)
}

// Create test client
c := client.New(&mockConn{})
c.ChangeName("test-user")
c.SetState(protocol.StateActive)

srv.clientsMu.Lock()
srv.clients[c.Name()] = c
srv.clientsMu.Unlock()

// Start broadcast loop
go srv.broadcastLoop()

// Send test message
testMsg := "Hello, world!"
msg := protocol.NewMessage("system", testMsg)
srv.broadcast <- msg

// Give time for message to be processed
time.Sleep(100 * time.Millisecond)

// Verify message was stored
srv.messagesMu.RLock()
msgCount := len(srv.messages)
lastMsg := srv.messages[msgCount-1]
srv.messagesMu.RUnlock()

if msgCount == 0 {
t.Error("No messages stored")
}

if lastMsg.Content != testMsg {
t.Errorf("Last message content = %q, want %q", lastMsg.Content, testMsg)
}
}

func TestClientNameConflict(t *testing.T) {
srv, err := createTestServer(nil)
if err != nil {
t.Fatalf("Failed to create test server: %v", err)
}

testName := "test-user"

// Register first client
c1 := client.New(&mockConn{})
if err := srv.registerClient(c1, testName); err != nil {
t.Fatalf("Failed to register first client: %v", err)
}

// Try to register second client with same name
c2 := client.New(&mockConn{})
if err := srv.registerClient(c2, testName); err == nil {
t.Error("Expected error when registering client with duplicate name")
}
}

// mockConn implementation for testing
type mockConn struct {
net.Conn
closed bool
}

func (c *mockConn) Close() error {
c.closed = true
return nil
}

func (c *mockConn) RemoteAddr() net.Addr {
return &net.TCPAddr{IP: net.ParseIP("127.0.0.1")}
}

func (c *mockConn) Write(b []byte) (n int, err error) {
return len(b), nil
}

func (c *mockConn) Read(b []byte) (n int, err error) {
return 0, fmt.Errorf("mock read")
}

func (c *mockConn) SetDeadline(t time.Time) error {
return nil
}
