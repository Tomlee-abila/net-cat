package main

import (
	"net"
	"os"
	"strings"
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

// Test the addClient method
func TestAddClient(t *testing.T) {
	server := NewServer(":8989")

	client1 := mockClient("Alice", "192.168.1.1", nil)
	client2 := mockClient("Bob", "192.168.1.2", nil)

	server.addClient(client1)
	server.addClient(client2)

	if len(server.clients) != 2 {
		t.Errorf("Expected 2 clients, got %d", len(server.clients))
	}

	if server.clients[0].name != "Alice" || server.clients[1].name != "Bob" {
		t.Errorf("Client names do not match the expected values.")
	}
}

// Test the removeClient method
func TestRemoveClient(t *testing.T) {
	server := NewServer(":8989")

	client1 := mockClient("Alice", "192.168.1.1", nil)
	client2 := mockClient("Bob", "192.168.1.2", nil)

	// Add clients to the server
	server.addClient(client1)
	server.addClient(client2)

	// Remove client1
	server.removeClient(client1)

	if len(server.clients) != 1 {
		t.Errorf("Expected 1 client, got %d", len(server.clients))
	}

	if server.clients[0].name != "Bob" {
		t.Errorf("Expected Bob to be the only remaining client.")
	}
}

// Test the messageClients method
func TestMessageClients(t *testing.T) {
	server := NewServer(":8989")

	client1 := mockClient("Alice", "192.168.1.1", nil)
	client2 := mockClient("Bob", "192.168.1.2", nil)

	// Add clients to the server
	server.addClient(client1)
	server.addClient(client2)

	// Send a message from client1
	message := "Hello, world!"
	timestamp := time.Now().Format("02-01-2006 15:04:05")
	server.messageClients(client1, message, timestamp)

	// Check that the message was added to the server's messages string
	expectedMessage := message + "\n[" + timestamp + "][Bob]:"
	if !containsSubstring(server.messages, expectedMessage) {
		t.Errorf("Expected message to be sent to Bob, but it was not.")
	}
}

// Helper function to check if a string contains a substring
func containsSubstring(str, substr string) bool {
	return strings.Contains(str, substr)
}

// Test the NewServer function
func TestNewServer(t *testing.T) {
	server := NewServer(":8989")

	if server == nil {
		t.Errorf("Expected server to be initialized, but it was nil.")
	}

	if server.listenAddr != ":8989" {
		t.Errorf("Expected listenAddr to be ':8989', got %s", server.listenAddr)
	}

	if server.quitch == nil {
		t.Errorf("Expected quitch channel to be initialized.")
	}

	if server.messages != "" {
		t.Errorf("Expected messages to be empty, got %s", server.messages)
	}
}

// Test the Start method for a successful server start
func TestServerStart(t *testing.T) {
	server := NewServer(":8989")

	go func() {
		if err := server.Start(); err != nil {
			t.Errorf("Error starting server: %v", err)
		}
	}()

	// Allow time for the server to start up
	time.Sleep(1 * time.Second)

	if server.ln == nil {
		t.Errorf("Expected listener to be initialized, but it was nil.")
	}
}

// Test for invalid port input in main
func TestMainInvalidPort(t *testing.T) {
	// Redirect os.Args to simulate an invalid port argument
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }() // Restore original os.Args

	os.Args = []string{"./TCPChat", "invalidPort"}
	server := NewServer(":8989")
	err := server.Start()

	if err == nil {
		t.Errorf("Expected error when starting server with invalid port.")
	}
}
