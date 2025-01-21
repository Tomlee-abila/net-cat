package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// Mock a simple client for testing
func mockClient(name string, ip string, conn net.Conn) Client {
	return Client{
		conn:  conn,
		ipAdd: ip,
		name:  name,
	}
}

// Create a mock connection for testing
type mockConn struct {
	net.Conn
	readData  chan []byte
	writeData chan []byte
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
	return nil
}

func (c *mockConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1")}
}

// Test username validation
func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Valid username", "Alice", false},
		{"Empty username", "", true},
		{"Too long username", strings.Repeat("a", 33), true},
		{"Username with newline", "Alice\n", true},
		{"Username with tab", "Alice\t", true},
		{"Username with spaces", "Alice Smith", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUsername(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateUsername() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test the addClient method with connection limits
func TestAddClient(t *testing.T) {
	server := NewServer(":8989")

	// Test normal addition
	client1 := mockClient("Alice", "192.168.1.1", nil)
	if err := server.addClient(client1); err != nil {
		t.Errorf("Failed to add first client: %v", err)
	}

	// Test duplicate username
	client2 := mockClient("Alice", "192.168.1.2", nil)
	if err := server.addClient(client2); err == nil {
		t.Error("Expected error for duplicate username, got none")
	}

	// Test connection limit
	for i := 0; i < 10; i++ {
		client := mockClient(fmt.Sprintf("User%d", i), fmt.Sprintf("192.168.1.%d", i+3), nil)
		server.addClient(client)
	}

	// Try to add one more client beyond the limit
	clientExtra := mockClient("Extra", "192.168.1.13", nil)
	if err := server.addClient(clientExtra); err == nil {
		t.Error("Expected error when exceeding connection limit")
	}
}

// Test concurrent client connections
func TestConcurrentConnections(t *testing.T) {
	server := NewServer(":8989")
	var wg sync.WaitGroup

	// Try to add 15 clients concurrently (more than max limit)
	for i := 0; i < 15; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			client := mockClient(fmt.Sprintf("User%d", i), fmt.Sprintf("192.168.1.%d", i), nil)
			server.addClient(client)
		}(i)
	}

	wg.Wait()

	// Verify we don't exceed max clients
	if len(server.clients) > server.maxClients {
		t.Errorf("Server exceeded max client limit: got %d, want <= %d", len(server.clients), server.maxClients)
	}
}

// Test message broadcasting
func TestMessageBroadcast(t *testing.T) {
	server := NewServer(":8989")
	conn1 := newMockConn()
	conn2 := newMockConn()

	client1 := mockClient("Alice", "192.168.1.1", conn1)
	client2 := mockClient("Bob", "192.168.1.2", conn2)

	server.addClient(client1)
	server.addClient(client2)

	// Test message broadcast
	timestamp := "[" + time.Now().Format("02-01-2006 15:04:05") + "]"
	server.messageClients(client1, "Hello", timestamp)

	// Check if client2 received the message
	select {
	case data := <-conn2.writeData:
		if !strings.Contains(string(data), "Hello") {
			t.Error("Client 2 didn't receive the broadcast message")
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for broadcast message")
	}
}

// Test client removal
func TestRemoveClient(t *testing.T) {
	server := NewServer(":8989")
	client1 := mockClient("Alice", "192.168.1.1", nil)
	client2 := mockClient("Bob", "192.168.1.2", nil)

	server.addClient(client1)
	server.addClient(client2)

	server.removeClient(client1)

	if len(server.clients) != 1 {
		t.Errorf("Expected 1 client after removal, got %d", len(server.clients))
	}

	if server.clients[0].name != "Bob" {
		t.Error("Wrong client was removed")
	}

	// Verify username is available again
	if server.activeNames["Alice"] {
		t.Error("Username was not freed after client removal")
	}
}

// Test server initialization
func TestNewServer(t *testing.T) {
	server := NewServer(":8989")

	if server == nil {
		t.Fatal("Server initialization failed")
	}

	if server.maxClients != 10 {
		t.Errorf("Expected maxClients to be 10, got %d", server.maxClients)
	}

	if server.activeNames == nil {
		t.Error("activeNames map was not initialized")
	}

	if server.messages != "" {
		t.Error("messages should be empty on initialization")
	}
}

// Test the main function with invalid arguments
func TestMainWithInvalidArgs(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// Test with too many arguments
	os.Args = []string{"cmd", "8989", "extra"}
	main()

	// Test with invalid port
	os.Args = []string{"cmd", "invalid"}
	main()
}
