package server

import (
"fmt"
"log"
"net"
"sync"
"time"

"net-cat/internal/client"
"net-cat/internal/config"
"net-cat/internal/errors"
"net-cat/internal/protocol"
)

// Server represents the chat server
type Server struct {
cfg    *config.Config
ln     net.Listener
quit   chan struct{}
done   chan struct{}

// Separate mutexes for different concerns
clientsMu     sync.RWMutex
messagesMu    sync.RWMutex
activeNamesMu sync.RWMutex

clients     map[string]*client.Client
messages    []protocol.Message
activeNames map[string]bool

// Message broadcasting
broadcast chan protocol.Message
}

// New creates and initializes a new chat server instance
func New(cfg *config.Config) *Server {
if cfg == nil {
cfg = config.DefaultConfig()
}

s := &Server{
cfg:          cfg,
quit:         make(chan struct{}),
done:         make(chan struct{}),
clients:      make(map[string]*client.Client),
messages:     make([]protocol.Message, 0),
activeNames:  make(map[string]bool),
broadcast:    make(chan protocol.Message, 1000),
clientsMu:    sync.RWMutex{},
messagesMu:   sync.RWMutex{},
activeNamesMu: sync.RWMutex{},
}

// Start broadcast loop immediately
go s.broadcastLoop()
return s
}

// Start begins listening for incoming connections and handles them
func (s *Server) Start() error {
ln, err := net.Listen("tcp", s.cfg.ListenAddr)
if err != nil {
return fmt.Errorf("failed to start server: %w", err)
}
s.ln = ln
defer ln.Close()

// Start connection cleaner
go s.cleanInactiveConnections()

log.Printf("Listening on %s\n", s.cfg.ListenAddr)

for {
select {
case <-s.quit:
return nil
default:
conn, err := ln.Accept()
if err != nil {
log.Printf("Failed to accept connection: %v", err)
continue
}

// Set initial timeout
conn.SetDeadline(time.Now().Add(s.cfg.ClientTimeout))

go s.handleConnection(conn)
}
}
}

// Stop gracefully shuts down the server
func (s *Server) Stop() error {
close(s.quit)
close(s.done)

// Close listener
if err := s.ln.Close(); err != nil {
return fmt.Errorf("error closing listener: %w", err)
}

// Disconnect all clients
s.clientsMu.Lock()
for _, client := range s.clients {
s.disconnectClient(client, "server shutdown")
}
s.clientsMu.Unlock()

return nil
}

func (s *Server) handleConnection(conn net.Conn) {
c := client.New(conn)

// Set up TCP connection
if err := client.SetupTCPConn(conn, *s.cfg); err != nil {
log.Printf("Failed to setup TCP connection: %v", err)
conn.Close()
return
}

// Authenticate client
name, err := client.Authenticate(conn, s.cfg)
if err != nil {
log.Printf("Authentication failed: %v", err)
conn.Close()
return
}

if err := s.registerClient(c, name); err != nil {
log.Printf("Failed to register client: %v", err)
conn.Close()
return
}

// Send message history
s.sendMessageHistory(c)

// Broadcast join message
s.broadcastSystemMessage(fmt.Sprintf("%s has joined our chat...", c.Name()))

go s.handleClientMessages(c)
}

func (s *Server) registerClient(c *client.Client, name string) error {
s.clientsMu.Lock()
defer s.clientsMu.Unlock()

if len(s.clients) >= s.cfg.MaxClients {
return errors.New(errors.ErrConnection, "server is full", nil)
}

s.activeNamesMu.Lock()
defer s.activeNamesMu.Unlock()

if s.activeNames[name] {
return errors.New(errors.ErrValidation, "username already taken", nil)
}

c.ChangeName(name)
c.SetState(protocol.StateAuthenticated)
s.clients[c.Name()] = c
s.activeNames[name] = true

return nil
}

func (s *Server) cleanInactiveConnections() {
ticker := time.NewTicker(time.Minute)
defer ticker.Stop()

for {
select {
case <-s.done:
return
case <-ticker.C:
s.clientsMu.Lock()
for _, c := range s.clients {
if time.Since(c.LastActivity()) > s.cfg.ClientTimeout {
s.disconnectClient(c, "timeout")
}
}
s.clientsMu.Unlock()
}
}
}

func (s *Server) disconnectClient(c *client.Client, reason string) {
if c == nil {
return
}

c.SetState(protocol.StateDisconnecting)

s.clientsMu.Lock()
delete(s.clients, c.Name())
s.clientsMu.Unlock()

s.activeNamesMu.Lock()
delete(s.activeNames, c.Name())
s.activeNamesMu.Unlock()

if err := c.Close(); err != nil {
log.Printf("Error closing client connection: %v", err)
}

if reason != "" {
s.broadcastSystemMessage(fmt.Sprintf("%s %s", c.Name(), reason))
}
}

func (s *Server) broadcastSystemMessage(message string) {
s.broadcast <- protocol.SystemMessage(message)
}

func (s *Server) sendMessageHistory(c *client.Client) {
s.messagesMu.RLock()
defer s.messagesMu.RUnlock()

for _, msg := range s.messages {
if err := c.Send(msg); err != nil {
log.Printf("Failed to send message history: %v", err)
return
}
}
}
