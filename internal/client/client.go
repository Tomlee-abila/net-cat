package client

import (
	"fmt"
	"net"
	"sync"
	"time"

	"net-cat/internal/protocol"
)

const (
	maxNameChanges = 2
	MaxNameChanges = 5 // Maximum number of times a client can change their name
)

type Client struct {
	Conn        net.Conn
	state       protocol.ConnectionState
	name        string
	nameHistory []string
	activity    time.Time
	done        chan struct{}
	closed      bool
	mu          sync.Mutex
}

func New(conn net.Conn) *Client {
	return &Client{
		Conn:     conn,
		state:    protocol.StateConnecting,
		activity: time.Now(),
		done:     make(chan struct{}),
	}
}

func (c *Client) Name() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.name
}

func (c *Client) State() protocol.ConnectionState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

func (c *Client) SetState(state protocol.ConnectionState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = state
}

func (c *Client) LastActivity() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.activity
}

func (c *Client) UpdateActivity() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.activity = time.Now()
}

func (c *Client) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

func (c *Client) Done() <-chan struct{} {
	return c.done
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	close(c.done)
	return c.Conn.Close()
}

func (c *Client) Send(msg protocol.Message) error {
	if c.State() != protocol.StateActive {
		return fmt.Errorf("client not in active state")
	}

	_, err := fmt.Fprintf(c.Conn, "%s[%s][%s]:%s\n",
		msg.Timestamp.Format(protocol.TimestampFormat),
		msg.From,
		c.Name(),
		msg.Content)
	return err
}

func (c *Client) SendPrompt() error {
	_, err := fmt.Fprintf(c.Conn, "[%s]: ", c.Name())
	return err
}

func (c *Client) SetDeadline(t time.Time) error {
	return c.Conn.SetDeadline(t)
}

func (c *Client) ChangeName(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.name != "" {
		c.nameHistory = append(c.nameHistory, c.name)
	}
	c.name = name
}

func (c *Client) CanChangeName() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.nameHistory) < maxNameChanges
}
