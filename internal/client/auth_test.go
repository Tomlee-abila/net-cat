package client

import (
"net"
"strings"
"testing"
"time"

"net-cat/internal/config"
)

// mockAuthConn implements a mock connection for testing authentication
type mockAuthConn struct {
net.Conn
input    string
output   []byte
closed   bool
deadline time.Time
}

func newMockAuthConn(input string) *mockAuthConn {
return &mockAuthConn{
input:  input,
output: make([]byte, 0),
}
}

func (c *mockAuthConn) Read(b []byte) (n int, err error) {
copy(b, c.input)
return len(c.input), nil
}

func (c *mockAuthConn) Write(b []byte) (n int, err error) {
c.output = append(c.output, b...)
return len(b), nil
}

func (c *mockAuthConn) Close() error {
c.closed = true
return nil
}

func (c *mockAuthConn) SetDeadline(t time.Time) error {
c.deadline = t
return nil
}

func (c *mockAuthConn) RemoteAddr() net.Addr {
return &net.TCPAddr{IP: net.ParseIP("127.0.0.1")}
}

func TestAuthenticate(t *testing.T) {
tests := []struct {
name      string
input     string
wantName  string
wantError bool
}{
{
name:      "Valid username",
input:     "testuser\n",
wantName:  "testuser",
wantError: false,
},
{
name:      "Empty username",
input:     "\n",
wantName:  "",
wantError: true,
},
{
name:      "Username with spaces",
input:     "test user\n",
wantName:  "test user",
wantError: false,
},
{
name:      "Username too long",
input:     strings.Repeat("a", 33) + "\n",
wantName:  "",
wantError: true,
},
{
name:      "Username with invalid chars",
input:     "test\tuser\n",
wantName:  "",
wantError: true,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
conn := newMockAuthConn(tt.input)
cfg := config.DefaultConfig()

name, err := Authenticate(conn, cfg)

// Verify welcome banner was sent
if !strings.Contains(string(conn.output), "Welcome to Linux TCP-Chat") {
t.Error("Welcome banner not sent")
}

// Check error
if (err != nil) != tt.wantError {
t.Errorf("Authenticate() error = %v, wantError %v", err, tt.wantError)
return
}

// Check returned name
if name != tt.wantName {
t.Errorf("Authenticate() name = %v, want %v", name, tt.wantName)
}
})
}
}

func TestValidateUsername(t *testing.T) {
tests := []struct {
name      string
username  string
maxLength int
wantError bool
}{
{
name:      "Valid username",
username:  "testuser",
maxLength: 32,
wantError: false,
},
{
name:      "Empty username",
username:  "",
maxLength: 32,
wantError: true,
},
{
name:      "Username too long",
username:  strings.Repeat("a", 33),
maxLength: 32,
wantError: true,
},
{
name:      "Username with tab",
username:  "test\tuser",
maxLength: 32,
wantError: true,
},
{
name:      "Username with newline",
username:  "test\nuser",
maxLength: 32,
wantError: true,
},
{
name:      "Username with carriage return",
username:  "test\ruser",
maxLength: 32,
wantError: true,
},
{
name:      "Username with spaces",
username:  "test user",
maxLength: 32,
wantError: false,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
err := ValidateUsername(tt.username, tt.maxLength)
if (err != nil) != tt.wantError {
t.Errorf("ValidateUsername() error = %v, wantError %v", err, tt.wantError)
}
})
}
}

func TestSetupTCPConn(t *testing.T) {
tests := []struct {
name      string
conn      net.Conn
wantError bool
}{
{
name:      "Valid TCP connection",
conn:      &net.TCPConn{},
wantError: false,
},
{
name:      "Non-TCP connection",
conn:      &mockConn{},
wantError: true,
},
}

cfg := *config.DefaultConfig()

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
err := SetupTCPConn(tt.conn, cfg)
if (err != nil) != tt.wantError {
t.Errorf("SetupTCPConn() error = %v, wantError %v", err, tt.wantError)
}
})
}
}
