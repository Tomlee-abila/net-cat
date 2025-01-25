package server

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"net-cat/internal/client"
	"net-cat/internal/config"
	"net-cat/internal/protocol"
)

type Server struct {
	cfg *config.Config
	ln  net.Listener
lnMu sync.Mutex

	done      chan struct{}
	doneMu    sync.Mutex
	isRunning bool

	clients   map[string]*client.Client
	clientsMu sync.RWMutex

	broadcast  chan protocol.Message
	messages   []protocol.Message
	messagesMu sync.RWMutex

	activeNames   map[string]bool
	activeNamesMu sync.RWMutex
}

func New(cfg *config.Config) *Server {
	return &Server{
		cfg:         cfg,
		done:        make(chan struct{}),
		clients:     make(map[string]*client.Client),
		broadcast:   make(chan protocol.Message, 100),
		messages:    make([]protocol.Message, 0),
		activeNames: make(map[string]bool),
	}
}

func (s *Server) Start() error {
	s.lnMu.Lock()
s.doneMu.Lock()
	if s.isRunning {
		s.doneMu.Unlock()
		return fmt.Errorf("server is already running")
	}

	// Reset done channel if it was closed
	if s.done == nil {
		s.done = make(chan struct{})
	}
	s.isRunning = true
	s.doneMu.Unlock()

	// Create listener while holding the lock
listener, err := net.Listen("tcp", s.cfg.ListenAddr)
	if err != nil {
		s.doneMu.Lock()
		s.isRunning = false
		s.doneMu.Unlock()
		return fmt.Errorf("failed to listen on %s: %w", s.cfg.ListenAddr, err)
	}
	s.ln = listener
s.lnMu.Unlock()

	log.Printf("Server listening on %s", s.ln.Addr())

	go s.acceptLoop()
	go s.cleanInactiveConnections()
	go s.broadcastLoop()

	return nil
}

func (s *Server) Stop() error {
// First acquire doneMu to check/update running state
s.doneMu.Lock()
if !s.isRunning {
s.doneMu.Unlock()
return nil
}
s.isRunning = false

// Signal shutdown to all goroutines
if s.done != nil {
close(s.done)
}
s.done = nil
s.doneMu.Unlock()

// Then acquire lnMu to close listener
s.lnMu.Lock()
if s.ln != nil {
err := s.ln.Close()
s.ln = nil
s.lnMu.Unlock()
if err != nil {
return fmt.Errorf("error closing listener: %w", err)
}
} else {
s.lnMu.Unlock()
}

// Create WaitGroup for client cleanup
var wg sync.WaitGroup

// Get snapshot of clients with RLock
s.clientsMu.RLock()
clients := make([]*client.Client, 0, len(s.clients))
for _, c := range s.clients {
clients = append(clients, c)
}
s.clientsMu.RUnlock()

// Close client connections concurrently
for _, c := range clients {
wg.Add(1)
go func(client *client.Client) {
defer wg.Done()
client.SetState(protocol.StateDisconnecting)
_ = client.Send(protocol.SystemMessage("Server shutting down..."))
_ = client.Conn.Close()
}(c)
}

// Wait for all client handlers to finish
wg.Wait()

// Clear maps after all handlers are done
// Acquire locks in consistent order to prevent deadlocks
s.activeNamesMu.Lock()
s.clientsMu.Lock()
s.clients = make(map[string]*client.Client)
s.activeNames = make(map[string]bool)
s.clientsMu.Unlock()
s.activeNamesMu.Unlock()

log.Println("Server stopped.")
return nil
}

func (s *Server) cleanInactiveConnections() {
	ticker := time.NewTicker(time.Second / 2) // Increase frequency
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			now := time.Now()

			// First pass: identify inactive clients under mutex lock
			s.clientsMu.Lock()
			inactiveClients := make(map[string]*client.Client)
			for name, c := range s.clients {
				if c.State() == protocol.StateActive && now.Sub(c.LastActivity()) > s.cfg.ClientTimeout {
					inactiveClients[name] = c
					// Remove from clients map immediately
					delete(s.clients, name)
				}
			}
			s.clientsMu.Unlock()

			// Second pass: cleanup inactive clients
			for name, c := range inactiveClients {
				s.activeNamesMu.Lock()
				delete(s.activeNames, name)
				s.activeNamesMu.Unlock()

				c.SetState(protocol.StateDisconnecting)
				_ = c.Conn.Close()
				log.Printf("Client %s disconnected: timeout", name)

				// Use non-blocking broadcast for timeout notifications
				select {
				case s.broadcast <- protocol.SystemMessage(fmt.Sprintf("%s has timeout", name)):
				case <-time.After(time.Second):
					log.Printf("Warning: Failed to broadcast timeout message for %s", name)
				case <-s.done:
					return
				}
			}
		}
	}
}

func (s *Server) acceptLoop() {
	for {
		s.lnMu.Lock()
conn, err := s.ln.Accept()
		s.lnMu.Unlock()
if err != nil {
			select {
			case <-s.done:
				log.Println("Stopped accepting new connections.")
				return
			default:
				// Check if the error indicates the listener is closed
				if ne, ok := err.(net.Error); ok {
					if !ne.Temporary() {
						log.Println("Listener closed, stopping accept loop.")
						return
					}
					// For temporary errors, log and continue
					log.Printf("Temporary accept error: %v\n", err)
					continue
				}
				// For non-temporary errors, log and continue
				log.Printf("Accept error: %v\n", err)
				continue
			}
		}

		s.clientsMu.RLock()
		if s.cfg.MaxClients > 0 && len(s.clients) >= s.cfg.MaxClients {
			s.clientsMu.RUnlock()
			conn.Close()
			continue
		}
		s.clientsMu.RUnlock()

		// Use WaitGroup to ensure handler is ready
		var handlerReady sync.WaitGroup
		handlerReady.Add(1)
		go func() {
			defer handlerReady.Done()
			s.handleConnection(conn)
		}()
		handlerReady.Wait()
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	c := client.New(conn)

	// Check if server is shutting down
	s.doneMu.Lock()
	done := s.done
	s.doneMu.Unlock()

	select {
	case <-done:
		return
	default:
	}

	// Authenticate client first
	name, err := client.Authenticate(conn, s.cfg)
	if err != nil {
		log.Printf("Authentication failed: %v", err)
		return
	}

	c.SetState(protocol.StateActive)
	c.ChangeName(name)

	// Register client after successful authentication
	if err := s.registerClient(c, name); err != nil {
		log.Printf("Failed to register client: %v", err)
		return
	}

	// Send message history before starting message handler
	s.sendMessageHistory(c)

	// Create a channel to signal handler completion
	handlerDone := make(chan struct{})

	// Start message handler in goroutine
	go func() {
		s.handleClientMessages(c)
		close(handlerDone)
	}()

	// Wait for either server shutdown or handler completion
	select {
	case <-done:
		// Server is shutting down
		c.SetState(protocol.StateDisconnecting)
		_ = c.Send(protocol.SystemMessage("Server shutting down..."))
		_ = conn.Close()
	case <-handlerDone:
		// Handler finished normally
	}
}

func (s *Server) registerClient(c *client.Client, name string) error {
	s.activeNamesMu.Lock()
	if s.activeNames[name] {
		s.activeNamesMu.Unlock()
		return fmt.Errorf("name already in use: %s", name)
	}
	s.activeNames[name] = true
	s.activeNamesMu.Unlock()

	s.clientsMu.Lock()
	s.clients[name] = c
	s.clientsMu.Unlock()

	log.Printf("New client registered: %s", name)
	return nil
}

func (s *Server) disconnectClient(c *client.Client, reason string) {
	name := c.Name()

	// Set state and close connection first to prevent new messages
	c.SetState(protocol.StateDisconnecting)
	_ = c.Conn.Close()

	// Update maps atomically
	s.clientsMu.Lock()
	s.activeNamesMu.Lock()
	delete(s.clients, name)
	delete(s.activeNames, name)
	s.activeNamesMu.Unlock()
	s.clientsMu.Unlock()

	log.Printf("Client %s disconnected: %s", name, reason)

	// Use non-blocking broadcast
	select {
	case s.broadcast <- protocol.SystemMessage(fmt.Sprintf("%s has %s", name, reason)):
	case <-time.After(time.Second):
		log.Printf("Warning: Failed to broadcast disconnect message for %s", name)
	case <-s.done:
	}
}

func (s *Server) sendMessageHistory(c *client.Client) {
	s.messagesMu.RLock()
	messages := make([]protocol.Message, len(s.messages))
	copy(messages, s.messages)
	s.messagesMu.RUnlock()

	for _, msg := range messages {
		if err := c.Send(msg); err != nil {
			log.Printf("Error sending history to %s: %v", c.Name(), err)
			return
		}
	}
}

func (s *Server) broadcastSystemMessage(text string) {
	s.doneMu.Lock()
	done := s.done
	s.doneMu.Unlock()

	// Use non-blocking send for broadcast with timeout
	select {
	case s.broadcast <- protocol.SystemMessage(text):
	case <-done:
		return
	case <-time.After(time.Second):
		log.Printf("Warning: Failed to broadcast system message: %s", text)
		return
	}
}
