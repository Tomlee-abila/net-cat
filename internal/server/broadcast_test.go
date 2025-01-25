package server

import (
"context"
"fmt"
"os"
"path/filepath"
"strings"
"sync"
"testing"
"time"

"net-cat/internal/client"
"net-cat/internal/config"
"net-cat/internal/protocol"
)

func TestLogMessage(t *testing.T) {
tmpDir := t.TempDir()
logFile := filepath.Join(tmpDir, "test.log")

cfg := config.DefaultConfig().
WithListenAddr(":0").
WithLogFile(logFile)

srv, err := createTestServer(cfg)
if err != nil {
t.Fatalf("Failed to create server: %v", err)
}

// Create channel to track message logged
messageDone := make(chan struct{})

testMsg := protocol.Message{
From:      "test-user",
Content:   "test message",
Timestamp: time.Now(),
}

// Start message logging in goroutine
go func() {
srv.logMessage(testMsg)
messageDone <- struct{}{}
}()

// Wait for message to be logged with increased timeout
timeout := time.After(5 * time.Second)
var content []byte

waitLoop:
for {
select {
case <-timeout:
t.Fatal("Timeout waiting for log file to be written")
case <-messageDone:
// Give filesystem a moment to sync
time.Sleep(100 * time.Millisecond)
content, err = os.ReadFile(logFile)
if err == nil && len(content) > 0 {
break waitLoop
}
case <-time.After(100 * time.Millisecond):
content, err = os.ReadFile(logFile)
if err == nil && len(content) > 0 {
break waitLoop
}
}
}

// Clean shutdown after test
defer func() {
if err := srv.Stop(); err != nil {
t.Errorf("Failed to stop server: %v", err)
}
}()

logStr := string(content)
if !strings.Contains(logStr, testMsg.From) || !strings.Contains(logStr, testMsg.Content) {
t.Error("Log file doesn't contain message content")
}
}

func TestLogMessageErrors(t *testing.T) {
// Use a nil file for error case testing
cfg := config.DefaultConfig().
WithListenAddr(":0").
WithLogFile("")

srv, err := createTestServer(cfg)
if err != nil {
t.Fatalf("Failed to create server: %v", err)
}
defer srv.Stop()

testMsg := protocol.Message{
From:      "test-user",
Content:   "test message",
Timestamp: time.Now(),
}

// Channel to track completion
done := make(chan struct{})
go func() {
    srv.logMessage(testMsg)
    close(done)
}()

select {
case <-done:
case <-time.After(time.Second):
    t.Error("logMessage took too long to complete")
}
}

func TestBroadcastLoopErrorHandling(t *testing.T) {
srv, err := createTestServer(nil)
if err != nil {
t.Fatalf("Failed to create server: %v", err)
}
defer srv.Stop()

clients := make([]*client.Client, 3)
conns := make([]*mockConn, 3)
for i := 0; i < 3; i++ {
conns[i] = newMockConn()
clients[i] = client.New(conns[i])
clients[i].ChangeName(fmt.Sprintf("user%d", i))
clients[i].SetState(protocol.StateActive)
if err := srv.registerClient(clients[i], clients[i].Name()); err != nil {
t.Fatalf("Failed to register client %d: %v", i, err)
}
}

conns[1].writeErr = fmt.Errorf("simulated write error")
msg := protocol.NewMessage("user0", "test message")
srv.broadcast <- msg

// Use channel to signal when broadcast is processed
broadcastDone := make(chan struct{})
go func() {
    for {
        srv.clientsMu.RLock()
        count := len(srv.clients)
        srv.clientsMu.RUnlock()
        if count == 2 {
            close(broadcastDone)
            return
        }
        time.Sleep(10 * time.Millisecond)
    }
}()

select {
case <-broadcastDone:
case <-time.After(time.Second):
    t.Fatal("Timeout waiting for broadcast to complete")
}

srv.clientsMu.RLock()
clientCount := len(srv.clients)
srv.clientsMu.RUnlock()
if clientCount != 2 {
t.Errorf("Expected 2 clients after send failure, got %d", clientCount)
}

for i := 0; i < cap(srv.broadcast)+1; i++ {
select {
case srv.broadcast <- msg:
case <-time.After(100 * time.Millisecond):
return
}
}
}

func TestBroadcastRateLimit(t *testing.T) {
cfg := config.DefaultConfig().
WithMessageRateLimit(100 * time.Millisecond).
WithListenAddr(":0")

srv, err := createTestServer(cfg)
if err != nil {
t.Fatalf("Failed to create server: %v", err)
}
defer srv.Stop()

conn := newMockConn()
c := client.New(conn)
c.ChangeName("test-user")
c.SetState(protocol.StateActive)

if err := srv.registerClient(c, c.Name()); err != nil {
t.Fatalf("Failed to register client: %v", err)
}

// Simulate sending messages through handleClientMessages
go func() {
for i := 0; i < 5; i++ {
msg := fmt.Sprintf("test message %d\n", i)
conn.readData <- []byte(msg)
}
}()

// Start message handling
go srv.handleClientMessages(c)

// Look for rate limit message
deadline := time.After(time.Second)
rateLimitFound := false
messageLoop:
for {
select {
case data := <-conn.writeData:
if strings.Contains(string(data), "wait before sending") {
rateLimitFound = true
break messageLoop
}
case <-deadline:
break messageLoop
}
}

if !rateLimitFound {
t.Error("Expected to receive rate limit message")
}
}

func TestConcurrentNameChanges(t *testing.T) {
srv, err := createTestServer(nil)
if err != nil {
t.Fatalf("Failed to create server: %v", err)
}
defer srv.Stop()

clientCount := 5
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

var wg sync.WaitGroup
nameChanges := make(map[string]bool)
var nameChangesMu sync.Mutex

for i := 0; i < clientCount; i++ {
wg.Add(1)
go func(idx int) {
defer wg.Done()

conn := newMockConn()
c := client.New(conn)
initialName := fmt.Sprintf("user%d", idx)
c.ChangeName(initialName)
c.SetState(protocol.StateActive)

if err := srv.registerClient(c, c.Name()); err != nil {
t.Errorf("Failed to register client %d: %v", idx, err)
return
}

for j := 0; j < 3; j++ {
    select {
    case <-ctx.Done():
        t.Error("Test timed out")
        return
    default:
        newName := fmt.Sprintf("user%d-%d", idx, j)
        err := srv.handleNameChange(c, newName)
        if err == nil {
            nameChangesMu.Lock()
            if nameChanges[newName] {
                t.Errorf("Name %s was already taken", newName)
            }
            nameChanges[newName] = true
            nameChangesMu.Unlock()
        }
    }
}
}(i)
}

wg.Wait()

srv.clientsMu.RLock()
finalClientCount := len(srv.clients)
srv.clientsMu.RUnlock()

if finalClientCount != clientCount {
t.Errorf("Expected %d clients, got %d", clientCount, finalClientCount)
}
}
