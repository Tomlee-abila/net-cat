package server

import (
"bufio"
"fmt"
"log"
"os"
"strings"
"time"

"net-cat/internal/client"
"net-cat/internal/protocol"
)

// broadcastLoop handles message broadcasting to all clients
func (s *Server) broadcastLoop() {
for {
select {
case <-s.done:
return
case msg := <-s.broadcast:
s.messagesMu.Lock()
s.messages = append(s.messages, msg)
s.messagesMu.Unlock()

// Async file logging
go s.logMessage(msg)

// Broadcast to all clients
s.clientsMu.RLock()
for _, c := range s.clients {
if c.State() == protocol.StateActive &&
c.Name() != msg.From {
// Asynchronous send to prevent blocking
go func(c *client.Client, m protocol.Message) {
if err := c.Send(m); err != nil {
log.Printf("Failed to send message to %s: %v", c.Name(), err)
}
}(c, msg)
}
}
s.clientsMu.RUnlock()
}
}
}

// handleClientMessages processes incoming messages from a client
func (s *Server) handleClientMessages(c *client.Client) {
defer s.disconnectClient(c, "left our chat...")

reader := bufio.NewReader(c.Conn)
c.SetState(protocol.StateActive)

mainLoop:
for {
select {
case <-c.Done():
break mainLoop
default:
// Update last activity time
c.SetDeadline(time.Now().Add(s.cfg.ClientTimeout))

// Write prompt
if err := c.SendPrompt(); err != nil {
log.Printf("Failed to send prompt: %v", err)
return
}

// Read message
line, err := reader.ReadString('\n')
if err != nil {
return
}

// Process message
message := strings.TrimSpace(line)
if message == "" {
continue
}

// Check for name change command
if strings.HasPrefix(message, "/name ") {
newName := strings.TrimSpace(strings.TrimPrefix(message, "/name "))
if err := s.handleNameChange(c, newName); err != nil {
if err := c.Send(protocol.Message{
From:      "SYSTEM",
Content:   fmt.Sprintf("Error changing name: %v", err),
Timestamp: time.Now(),
}); err != nil {
log.Printf("Failed to send error message: %v", err)
}
}
continue
}

// Rate limiting
if time.Since(c.LastActivity()) < s.cfg.MessageRateLimit {
if err := c.Send(protocol.SystemMessage("Message rate limit exceeded. Please wait.")); err != nil {
log.Printf("Failed to send rate limit message: %v", err)
}
continue
}

// Size limit
if len(message) > s.cfg.MaxMessageSize {
if err := c.Send(protocol.SystemMessage(
fmt.Sprintf("Message too long. Maximum length is %d characters.", s.cfg.MaxMessageSize))); err != nil {
log.Printf("Failed to send size limit message: %v", err)
}
continue
}

msg := protocol.NewMessage(c.Name(), message)
s.broadcast <- msg
c.UpdateActivity()
}
}
}

func (s *Server) handleNameChange(c *client.Client, newName string) error {
if !c.CanChangeName() {
return fmt.Errorf("maximum name changes exceeded")
}

if err := client.ValidateUsername(newName, s.cfg.MaxNameLength); err != nil {
return err
}

s.activeNamesMu.Lock()
defer s.activeNamesMu.Unlock()

if s.activeNames[newName] {
return fmt.Errorf("username already taken")
}

oldName := c.Name()
delete(s.activeNames, oldName)
s.activeNames[newName] = true

c.ChangeName(newName)
s.broadcastSystemMessage(fmt.Sprintf("%s has changed their name to %s", oldName, newName))

return nil
}

func (s *Server) logMessage(msg protocol.Message) {
logFile, err := os.OpenFile("server_log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
if err != nil {
log.Printf("Error opening log file: %v", err)
return
}
defer logFile.Close()

logEntry := fmt.Sprintf("[%s][%s]:%s\n",
msg.Timestamp.Format(protocol.TimestampFormat),
msg.From,
msg.Content)

if _, err := logFile.WriteString(logEntry); err != nil {
log.Printf("Error writing to log file: %v", err)
}
}
