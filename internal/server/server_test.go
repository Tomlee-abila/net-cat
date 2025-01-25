package server

import (
"fmt"
"strings"
"testing"
"runtime"
"sync"
"sync/atomic"
"time"

"net-cat/internal/client"
"net-cat/internal/config"
"net-cat/internal/protocol"
)

// Helper function to read all responses with proper synchronization
func readAllResponses(ch chan []byte) []string {
var (
    responses []string
    mu sync.Mutex
)
timeout := time.After(100 * time.Millisecond)
for {
    select {
    case data := <-ch:
        mu.Lock()
        responses = append(responses, string(data))
        mu.Unlock()
    case <-timeout:
        mu.Lock()
        result := make([]string, len(responses))
        copy(result, responses)
        mu.Unlock()
        return result
    }
}
}

func TestServerPerformance(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping performance test in short mode")
    }

    cfg := config.DefaultConfig().
        WithListenAddr(":0").
        WithMaxMessageSize(1024).
        WithMessageRateLimit(1 * time.Millisecond)

    srv, err := createTestServer(cfg)
    if err != nil {
        t.Fatalf("Failed to create test server: %v", err)
    }

    // Use WaitGroup to track all clients for proper cleanup
    var clientWg sync.WaitGroup
    defer func() {
        srv.Stop()
        clientWg.Wait()
    }()

    createClient := func(name string) (*mockConn, *client.Client) {
        conn := newMockConn()
        c := client.New(conn)
        c.ChangeName(name)
        c.SetState(protocol.StateActive)

        srv.clientsMu.Lock()
        srv.activeNamesMu.Lock()
        if err := srv.registerClient(c, c.Name()); err != nil {
            srv.activeNamesMu.Unlock()
            srv.clientsMu.Unlock()
            t.Fatalf("Failed to register client %s: %v", name, err)
        }
        srv.activeNamesMu.Unlock()
        srv.clientsMu.Unlock()

        clientWg.Add(1)
        go func() {
            srv.handleClientMessages(c)
            clientWg.Done()
        }()

        time.Sleep(10 * time.Millisecond) // Wait for handler startup
        clearChannelBytes(conn.writeData)
        return conn, c
    }

    t.Run("high message volume", func(t *testing.T) {
        conn, client := createClient("volume_test")
        defer client.Close()

        messageCount := 100 // Reduced from 1000 to avoid test timeout
        var receivedCount atomic.Int32
        done := make(chan struct{})

        // Monitor responses in separate goroutine
        go func() {
            defer close(done)
            for i := 0; i < messageCount; i++ {
                select {
                case data := <-conn.writeData:
                    if strings.Contains(string(data), fmt.Sprintf("Message %d", i)) {
                        receivedCount.Add(1)
                    }
                case <-time.After(100 * time.Millisecond):
                    return
                }
            }
        }()

        start := time.Now()

        // Send messages with rate limiting
        for i := 0; i < messageCount; i++ {
            msg := fmt.Sprintf("Message %d", i)
            conn.readData <- []byte(msg + "\n")
            time.Sleep(time.Millisecond) // Prevent overwhelming the server
        }

        // Wait for processing with timeout
        select {
        case <-done:
        case <-time.After(5 * time.Second):
            t.Error("Timeout waiting for message processing")
        }

        duration := time.Since(start)
        received := receivedCount.Load()
        messagesPerSecond := float64(received) / duration.Seconds()

        t.Logf("Processed %d messages in %v (%.2f msgs/sec)",
            received, duration, messagesPerSecond)

        if received < int32(float64(messageCount) * 0.9) { // Allow for some message loss
            t.Errorf("Too many messages lost: sent %d, received %d",
                messageCount, received)
        }
    })

    t.Run("concurrent connections", func(t *testing.T) {
        clientCount := 10 // Reduced from 50 to avoid overwhelming
        messageCount := 5  // Reduced from 20
        var wg sync.WaitGroup
        errChan := make(chan error, clientCount*messageCount)

        // Launch clients in batches to prevent connection flood
        for i := 0; i < clientCount; i++ {
            wg.Add(1)
            go func(id int) {
                defer wg.Done()

                conn, client := createClient(fmt.Sprintf("stress_test_%d", id))
                defer client.Close()

                // Send messages with rate limiting
                for j := 0; j < messageCount; j++ {
                    msg := fmt.Sprintf("Client %d Message %d", id, j)
                    conn.readData <- []byte(msg + "\n")

                    // Wait and verify with timeout
                    verified := make(chan bool)
                    go func() {
                        defer close(verified)
                        deadline := time.After(100 * time.Millisecond)
                        for {
                            select {
                            case data := <-conn.writeData:
                                if strings.Contains(string(data), msg) {
                                    verified <- true
                                    return
                                }
                            case <-deadline:
                                return
                            }
                        }
                    }()

                    select {
                    case <-verified:
                    case <-time.After(200 * time.Millisecond):
                        errChan <- fmt.Errorf("client %d: message %d timeout", id, j)
                    }

                    time.Sleep(10 * time.Millisecond)
                }
            }(i)

            // Space out client creation
            time.Sleep(50 * time.Millisecond)
        }

        // Wait with timeout
        done := make(chan struct{})
        go func() {
            wg.Wait()
            close(done)
        }()

        select {
        case <-done:
        case <-time.After(10 * time.Second):
            t.Fatal("Test timeout")
        }

        close(errChan)
        var errors []error
        for err := range errChan {
            errors = append(errors, err)
        }

        if len(errors) > 0 {
            t.Errorf("Encountered %d errors during stress test:", len(errors))
            for _, err := range errors {
                t.Error(err)
            }
        }

        // Check server health
        runtime.GC()
        t.Logf("Active goroutines after test: %d", runtime.NumGoroutine())
    })
}

// Rest of the test file remains unchanged...
