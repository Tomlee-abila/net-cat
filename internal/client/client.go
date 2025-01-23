package client

import (
"fmt"
"net"
"sync"
"time"

"net-cat/internal/errors"
"net-cat/internal/protocol"
)

// Client represents a connected chat client
type Client struct {
Conn            net.Conn       // Exposed for server package
IPAddr          string         // Exposed for server package
name            string
state           protocol.ConnectionState
lastActivity    time.Time
msgCount        int
done            chan struct{}
mu              sync.Mutex
closed          bool
nameHistory     []string
nameChangeCount int
}

// New creates a new Client instance
func New(conn net.Conn) *Client {
return &Client{
Conn:            conn,
IPAddr:          conn.RemoteAddr().String(),
state:           protocol.StateConnecting,
lastActivity:    time.Now(),
done:            make(chan struct{}),
mu:              sync.Mutex{},
closed:          false,
nameHistory:     make([]string, 0),
nameChangeCount: 0,
}
}

// Name returns the client's current name
func (c *Client) Name() string {
return c.name
}

// State returns the client's current connection state
func (c *Client) State() protocol.ConnectionState {
return c.state
}

// SetState updates the client's connection state
func (c *Client) SetState(state protocol.ConnectionState) {
c.state = state
}

// LastActivity returns the time of the client's last activity
func (c *Client) LastActivity() time.Time {
return c.lastActivity
}

// UpdateActivity updates the client's last activity timestamp
func (c *Client) UpdateActivity() {
c.lastActivity = time.Now()
}

// Close closes the client connection and marks it as closed
func (c *Client) Close() error {
c.mu.Lock()
defer c.mu.Unlock()

if !c.closed {
close(c.done)
c.closed = true
if c.Conn != nil {
return c.Conn.Close()
}
}
return nil
}

// IsClosed returns whether the client is closed
func (c *Client) IsClosed() bool {
c.mu.Lock()
defer c.mu.Unlock()
return c.closed
}

// Send sends a message to the client
func (c *Client) Send(msg protocol.Message) error {
if c.state != protocol.StateActive {
return errors.New(errors.ErrConnection, "client not active", c)
}

_, err := c.Conn.Write([]byte(msg.String() + "\n"))
if err != nil {
return errors.New(errors.ErrConnection, fmt.Sprintf("failed to send message: %v", err), c)
}

return nil
}

// SendPrompt sends the chat prompt to the client
func (c *Client) SendPrompt() error {
prompt := fmt.Sprintf("[%s][%s]:",
time.Now().Format(protocol.TimestampFormat),
c.name)

_, err := c.Conn.Write([]byte(prompt))
if err != nil {
return errors.New(errors.ErrConnection, fmt.Sprintf("failed to send prompt: %v", err), c)
}

return nil
}

// ChangeName changes the client's name and updates history
func (c *Client) ChangeName(newName string) {
c.nameHistory = append(c.nameHistory, c.name)
c.name = newName
c.nameChangeCount++
}

// CanChangeName checks if the client can change their name
func (c *Client) CanChangeName() bool {
return c.nameChangeCount < 3
}

// Done returns the client's done channel
func (c *Client) Done() <-chan struct{} {
return c.done
}

// SetDeadline sets the read/write deadlines on the connection
func (c *Client) SetDeadline(t time.Time) error {
return c.Conn.SetDeadline(t)
}
