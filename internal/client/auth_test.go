package client

import (
	"bytes"
	"net"
	"strings"
	"testing"
	"time"

	"net-cat/internal/config"
	"net-cat/internal/errors"
)

// mockNetConn implements net.Conn for testing
type mockNetConn struct {
	readData  bytes.Buffer
	writeData bytes.Buffer
	closed    bool
}

func (m *mockNetConn) Read(b []byte) (n int, err error)  { return m.readData.Read(b) }
func (m *mockNetConn) Write(b []byte) (n int, err error) { return m.writeData.Write(b) }
func (m *mockNetConn) Close() error                      { m.closed = true; return nil }
func (m *mockNetConn) LocalAddr() net.Addr               { return nil }
func (m *mockNetConn) RemoteAddr() net.Addr              { return nil }
func (m *mockNetConn) SetDeadline(t time.Time) error     { return nil }
func (m *mockNetConn) SetReadDeadline(t time.Time) error { return nil }
func (m *mockNetConn) SetWriteDeadline(t time.Time) error { return nil }

func TestAuthenticate(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		cfg         *config.Config
		wantErr     bool
		expectedErr string
	}{
		{
			name:  "Valid username",
			input: "validuser\n",
			cfg: &config.Config{
				MaxNameLength: 20,
			},
			wantErr: false,
		},
		{
			name:  "Empty username",
			input: "\n",
			cfg: &config.Config{
				MaxNameLength: 20,
			},
			wantErr:     true,
			expectedErr: "username cannot be empty",
		},
		{
			name:  "Username too long",
			input: "verylongusernamethatexceedslimit\n",
			cfg: &config.Config{
				MaxNameLength: 20,
			},
			wantErr:     true,
			expectedErr: "username too long",
		},
		{
			name:  "Invalid characters",
			input: "user@name\n",
			cfg: &config.Config{
				MaxNameLength: 20,
			},
			wantErr:     true,
			expectedErr: "username can only contain",
		},
		{
			name:  "Valid username with spaces",
			input: "John Doe\n",
			cfg: &config.Config{
				MaxNameLength: 20,
			},
			wantErr: false,
		},
		{
			name:  "Valid username with underscore",
			input: "john_doe\n",
			cfg: &config.Config{
				MaxNameLength: 20,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &mockNetConn{}
			conn.readData.WriteString(tt.input)

			name, err := Authenticate(conn, tt.cfg)

			// Verify welcome banner was sent
			if !strings.Contains(conn.writeData.String(), "Welcome to Linux TCP-Chat!") {
				t.Error("Welcome banner not sent")
			}
			if !strings.Contains(conn.writeData.String(), "[ENTER YOUR NAME]") {
				t.Error("Name prompt not sent")
			}

			// Verify Linux logo ASCII art
			if !strings.Contains(conn.writeData.String(), "_nnnn_") ||
				!strings.Contains(conn.writeData.String(), "dGGGGMMb") {
				t.Error("Linux logo ASCII art not sent correctly")
			}

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.expectedErr) {
					t.Errorf("Expected error containing %q, got %v", tt.expectedErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			expectedName := strings.TrimSpace(tt.input)
			if name != expectedName {
				t.Errorf("Expected name %q, got %q", expectedName, name)
			}
		})
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		maxLength   int
		wantErr     bool
		expectedErr errors.ErrorType
	}{
		{
			name:      "Valid username",
			username:  "validuser",
			maxLength: 20,
			wantErr:   false,
		},
		{
			name:        "Empty username",
			username:    "",
			maxLength:   20,
			wantErr:     true,
			expectedErr: errors.ErrValidation,
		},
		{
			name:        "Username too long",
			username:    "verylongusernamethatexceedslimit",
			maxLength:   20,
			wantErr:     true,
			expectedErr: errors.ErrValidation,
		},
		{
			name:        "Username with special characters",
			username:    "user@name",
			maxLength:   20,
			wantErr:     true,
			expectedErr: errors.ErrValidation,
		},
		{
			name:      "Username with spaces",
			username:  "John Doe",
			maxLength: 20,
			wantErr:   false,
		},
		{
			name:      "Username with numbers",
			username:  "user123",
			maxLength: 20,
			wantErr:   false,
		},
		{
			name:      "Username with underscore",
			username:  "john_doe",
			maxLength: 20,
			wantErr:   false,
		},
		{
			name:        "Username with leading spaces",
			username:    "   user",
			maxLength:   20,
			wantErr:     true,
			expectedErr: errors.ErrValidation,
		},
		{
			name:        "Username with trailing spaces",
			username:    "user   ",
			maxLength:   20,
			wantErr:     true,
			expectedErr: errors.ErrValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUsername(tt.username, tt.maxLength)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if customErr, ok := err.(*errors.ClientError); !ok || customErr.Type != tt.expectedErr {
					t.Errorf("Expected error type %v, got %v", tt.expectedErr, customErr.Type)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// mockTCPConn implements tcpKeepAliver interface for testing
type mockTCPConn struct {
		    mockNetConn
		    keepAlive     bool
		    keepAlivePeriod time.Duration
	}

func (m *mockTCPConn) SetKeepAlive(enabled bool) error {
		    m.keepAlive = enabled
		    return nil
	}

func (m *mockTCPConn) SetKeepAlivePeriod(d time.Duration) error {
		    m.keepAlivePeriod = d
		    return nil
	}

	func TestSetupTCPConn(t *testing.T) {
    mockKA := &mockTCPConn{}

    cfg := config.Config{
		        ClientTimeout: 30 * time.Second,
	    }

    err := SetupTCPConn(mockKA, cfg)
	    if err != nil {
		        t.Errorf("Unexpected error: %v", err)
	    }

    if !mockKA.keepAlive {
		        t.Error("Expected keepalive to be enabled")
	    }

    if mockKA.keepAlivePeriod != cfg.ClientTimeout {
		        t.Errorf("Expected keepalive period %v, got %v", cfg.ClientTimeout, mockKA.keepAlivePeriod)
	    }
}
