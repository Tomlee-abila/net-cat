package server

import (
    "bufio"
    "fmt"
    "log"
    "os"
    "strings"
    "time"

    "sync"
    "net-cat/internal/client"
    "net-cat/internal/protocol"
)

func (s *Server) isShuttingDown() bool {
    s.doneMu.Lock()
    defer s.doneMu.Unlock()
    select {
    case <-s.done:
        return true
    default:
        return false
    }
}

func (s *Server) broadcastLoop() {
    for {
        // Check done channel first
        s.doneMu.Lock()
        done := s.done
        s.doneMu.Unlock()

        if done == nil {
            return
        }

        select {
        case <-done:
            return // Exit if server is shutting down
        case msg, ok := <-s.broadcast:
            if !ok {
                return // Channel closed
            }
            s.messagesMu.Lock()
            s.messages = append(s.messages, msg)
            s.messagesMu.Unlock()

            // Async file logging
            go s.logMessage(msg)

            // Get copy of client list to avoid holding lock during send
            s.clientsMu.RLock()
            clients := make([]*client.Client, 0, len(s.clients))
            for _, c := range s.clients {
                if c.State() == protocol.StateActive {
                    // Only filter out messages from the same client for non-system messages
                    if msg.From == "SYSTEM" || c.Name() != msg.From {
                        clients = append(clients, c)
                    }
                }
            }
            s.clientsMu.RUnlock()

            // Track failed clients for cleanup
            var failedClients []*client.Client
            var failedClientsMu sync.Mutex
            var wg sync.WaitGroup

            // Broadcast to all clients concurrently
            for _, c := range clients {
                wg.Add(1)
                go func(client *client.Client) {
                    defer wg.Done()

                    // Create done channel for this send operation
                    sendDone := make(chan struct{})
                    go func() {
                        defer close(sendDone)
                        if err := client.Send(msg); err != nil {
                            log.Printf("Failed to send message to %s: %v", client.Name(), err)
                            failedClientsMu.Lock()
                            failedClients = append(failedClients, client)
                            failedClientsMu.Unlock()
                        }
                    }()

                    // Wait with timeout for send to complete
                    select {
                    case <-sendDone:
                    case <-time.After(time.Second):
                        log.Printf("Send timeout for client %s", client.Name())
                        failedClientsMu.Lock()
                        failedClients = append(failedClients, client)
                        failedClientsMu.Unlock()
                    case <-done:
                        return
                    }
                }(c)
            }

            // Wait for all sends to complete or server shutdown
            done := make(chan struct{})
            go func() {
                wg.Wait()
                close(done)
            }()

            select {
            case <-done:
                // All sends completed
            case <-s.done:
                return // Server shutting down
            }

            // Process failed clients after all sends complete
            for _, c := range failedClients {
                s.disconnectClient(c, "connection failure")
            }
        }
    }
}

func (s *Server) handleClientMessages(c *client.Client) {
    reader := bufio.NewReader(c.Conn)
    lastMessageTime := time.Now().Add(-s.cfg.MessageRateLimit)
    c.SetState(protocol.StateActive)

    // Create a cleanup function to handle disconnection
    cleanup := func(reason string) {
        if c.State() != protocol.StateDisconnecting {
            s.disconnectClient(c, reason)
        }
    }
    defer cleanup("left our chat...")

    mainLoop:
    for {
        select {
        case <-s.done:
            return
        case <-c.Done():
            break mainLoop
        default:
        }

        if s.isShuttingDown() {
            // Client disconnected
            break mainLoop
        }

            if err := c.SendPrompt(); err != nil {
                log.Printf("Failed to send prompt: %v", err)
                break mainLoop
            }

            line, err := reader.ReadString('\n')
            if err != nil {
                break mainLoop
            }

            message := strings.TrimSpace(line)
            if message == "" {
                continue
            }

            // Check rate limit first
            now := time.Now()
            if now.Sub(lastMessageTime) < s.cfg.MessageRateLimit {
                remaining := s.cfg.MessageRateLimit - now.Sub(lastMessageTime)
                errMsg := protocol.SystemMessage(fmt.Sprintf("please wait before sending another message (%.1f seconds remaining)", remaining.Seconds()))
                if sendErr := c.Send(errMsg); sendErr != nil {
                    log.Printf("Failed to send error message: %v", sendErr)
                    break mainLoop
                }
                continue
            }

            // Check message size limit
            if len(message) > s.cfg.MaxMessageSize {
                errMsg := protocol.SystemMessage(fmt.Sprintf("message too long (maximum %d characters allowed)", s.cfg.MaxMessageSize))
                if sendErr := c.Send(errMsg); sendErr != nil {
                    log.Printf("Failed to send error message: %v", sendErr)
                    break mainLoop
                }
                continue
            }

            // Update timestamp before processing
            lastMessageTime = now

            // Handle name change command
            if strings.HasPrefix(message, "/name") {
                parts := strings.Fields(message)
                if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
                    errMsg := protocol.SystemMessage("error changing name: invalid name format")
                    if sendErr := c.Send(errMsg); sendErr != nil {
                        log.Printf("Failed to send error message: %v", sendErr)
                        break mainLoop
                    }
                    continue
                }

                newName := strings.TrimSpace(parts[1])
                if strings.ContainsAny(newName, "/\\:*?\"<>|") {
                    errMsg := protocol.SystemMessage("error changing name: invalid characters in name")
                    if sendErr := c.Send(errMsg); sendErr != nil {
                        log.Printf("Failed to send error message: %v", sendErr)
                        break mainLoop
                    }
                    continue
                }

                if err := s.handleNameChange(c, newName); err != nil {
                    errMsg := protocol.SystemMessage(err.Error())
                    if sendErr := c.Send(errMsg); sendErr != nil {
                        log.Printf("Failed to send error message: %v", sendErr)
                        break mainLoop
                    }
                }
                continue
            }

            // Broadcast regular message
            msg := protocol.Message{
                From:      c.Name(),
                Content:   message,
                Timestamp: time.Now(),
            }

            select {
            case s.broadcast <- msg:
            case <-time.After(time.Second): // Use timeout instead of done channel
                break mainLoop
            }
    }
}

func (s *Server) processMessage(c *client.Client, message string, lastMessageTime *time.Time) error {
    // Handle name change command
    if strings.HasPrefix(message, "/name") {
        parts := strings.Fields(message)
        if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
            return fmt.Errorf("error changing name: invalid name format")
        }
        newName := strings.TrimSpace(parts[1])
        if strings.ContainsAny(newName, "/\\:*?\"<>|") {
            return fmt.Errorf("error changing name: invalid characters in name")
        }

        if err := s.handleNameChange(c, newName); err != nil {
            return fmt.Errorf("error changing name: %v", err)
        }
        return nil
    }

    // Check rate limit first
    now := time.Now()
    if now.Sub(*lastMessageTime) < s.cfg.MessageRateLimit {
        remaining := s.cfg.MessageRateLimit - now.Sub(*lastMessageTime)
        return fmt.Errorf("please wait before sending another message (%.1f seconds remaining)", remaining.Seconds())
    }

    // Check message size limit
    if len(message) > s.cfg.MaxMessageSize {
        return fmt.Errorf("message too long (maximum %d characters allowed)", s.cfg.MaxMessageSize)
    }

    // Update last message time before any potential broadcasts
    *lastMessageTime = now
    return nil
}

func (s *Server) handleNameChange(c *client.Client, newName string) error {
    if !c.CanChangeName() {
        return fmt.Errorf("maximum name changes exceeded")
    }

    if err := client.ValidateUsername(newName, s.cfg.MaxNameLength); err != nil {
        return fmt.Errorf("invalid name: %v", err)
    }

    // Take locks in consistent order to prevent deadlocks
    s.clientsMu.Lock()
    s.activeNamesMu.Lock()
    defer s.activeNamesMu.Unlock()
    defer s.clientsMu.Unlock()

    if s.activeNames[newName] {
        return fmt.Errorf("username already taken")
    }

    oldName := c.Name()
    msg := protocol.SystemMessage(fmt.Sprintf("%s changed their name to %s", oldName, newName))

    // Update name mappings
    delete(s.activeNames, oldName)
    s.activeNames[newName] = true

    // Update client state
    delete(s.clients, oldName)
    c.ChangeName(newName)
    s.clients[newName] = c

    // Use non-blocking broadcast with timeout
    select {
    case s.broadcast <- msg:
    case <-time.After(time.Second):
        log.Printf("Warning: Failed to broadcast name change for %s", oldName)
    case <-s.done:
        return fmt.Errorf("server shutting down")
    }

    return nil
}

func (s *Server) logMessage(msg protocol.Message) {
    if s.cfg.LogFile == "" {
        return // Skip logging if no log file configured
    }

    if s.isShuttingDown() {
        return
    }

    // Create a channel to coordinate log write completion
    done := make(chan struct{})
    timer := time.NewTimer(2 * time.Second)
    defer timer.Stop()

    go func() {
        defer close(done)

        logFile, err := os.OpenFile(s.cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
        if err != nil {
            log.Printf("Error opening log file: %v", err)
            select {
            case s.broadcast <- protocol.SystemMessage(fmt.Sprintf("error writing to log: %v", err)):
            case <-timer.C:
            default:
                log.Printf("Failed to broadcast log error message")
            }
            return
        }
        defer logFile.Close()

        logEntry := fmt.Sprintf("[%s][%s]:%s\n",
            msg.Timestamp.Format(protocol.TimestampFormat),
            msg.From,
            msg.Content)

        if _, err := logFile.WriteString(logEntry); err != nil {
            log.Printf("Error writing to log file: %v", err)
            select {
            case s.broadcast <- protocol.SystemMessage(fmt.Sprintf("error writing to log: %v", err)):
            case <-timer.C:
            default:
                log.Printf("Failed to broadcast log error message")
            }
        }
    }()

    // Wait for log write with timeout
    select {
    case <-done:
    case <-timer.C:
        log.Printf("Warning: Log write timed out for message from %s", msg.From)
    }
}
