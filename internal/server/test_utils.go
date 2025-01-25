package server

import (
	"io"
	"net"
	"sync"
	"time"

	"net-cat/internal/client"
	"net-cat/internal/config"
	"net-cat/internal/protocol"
)

// Mock network types for testing
type mockConn struct {
	writeData  chan []byte
	writeErr   error
	readData   chan []byte
	closed     bool
	remoteAddr net.Addr
	mu         sync.Mutex
}

func newMockConn() *mockConn {
	return &mockConn{
		writeData: make(chan []byte, 1000),
		readData:  make(chan []byte, 1000),
	}
}

func (c *mockConn) Read(b []byte) (n int, err error) {
	select {
	case data := <-c.readData:
		copy(b, data)
		return len(data), nil
	case <-time.After(10 * time.Millisecond):
		return 0, io.EOF
	}
}

func (c *mockConn) Write(b []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.writeErr != nil {
		return 0, c.writeErr
	}

	c.writeData <- b
	return len(b), nil
}

func (c *mockConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

func (c *mockConn) SetDeadline(t time.Time) error      { return nil }
func (c *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func (c *mockConn) RemoteAddr() net.Addr {
	if c.remoteAddr != nil {
		return c.remoteAddr
	}
	return &mockAddr{str: "mock:1234"}
}

func (c *mockConn) LocalAddr() net.Addr {
	return &mockAddr{str: "local:1234"}
}

type mockAddr struct {
	str string
}

func (a *mockAddr) Network() string { return "mock" }
func (a *mockAddr) String() string  { return a.str }

func clearChannelBytes(ch chan []byte) {
	if ch == nil {
		return
	}
	for {
		select {
		case <-ch:
			continue
		default:
			return
		}
	}
}

// createTestServer creates a new server instance for testing
func createTestServer(cfg *config.Config) (*Server, error) {
	if cfg == nil {
		cfg = config.DefaultConfig().WithListenAddr(":0")
	} else {
		cfg = cfg.WithListenAddr(":0")
	}

	srv := &Server{
		cfg:         cfg,
		clients:     make(map[string]*client.Client),
		broadcast:   make(chan protocol.Message, 100),
		messages:    make([]protocol.Message, 0),
		activeNames: make(map[string]bool),
		done:        make(chan struct{}),
	}

	if err := srv.Start(); err != nil {
		return nil, err
	}

	time.Sleep(100 * time.Millisecond)
	return srv, nil
}
