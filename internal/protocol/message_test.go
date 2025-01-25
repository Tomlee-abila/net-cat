package protocol

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestMessage(t *testing.T) {
	testCases := []struct {
		name     string
		from     string
		content  string
		wantFrom string
	}{
		{
			name:     "Normal message",
			from:     "user1",
			content:  "Hello, world!",
			wantFrom: "user1",
		},
		{
			name:     "System message",
			from:     "SYSTEM",
			content:  "Server starting...",
			wantFrom: "SYSTEM",
		},
		{
			name:     "Empty content",
			from:     "user1",
			content:  "",
			wantFrom: "user1",
		},
		{
			name:     "Long content",
			from:     "user1",
			content:  strings.Repeat("a", MaxMessageSize+1),
			wantFrom: "user1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := NewMessage(tc.from, tc.content)

			if msg.From != tc.wantFrom {
				t.Errorf("Message.From = %v, want %v", msg.From, tc.wantFrom)
			}

			if msg.Content != tc.content {
				t.Errorf("Message.Content = %v, want %v", msg.Content, tc.content)
			}

			if msg.Timestamp.IsZero() {
				t.Error("Message.Timestamp should not be zero")
			}

			// Test String() format
			str := msg.String()
if tc.content == "" {
    if str != "" {
        t.Errorf("Empty message should return empty string, got %v", str)
    }
} else {
    if !strings.Contains(str, tc.from) {
        t.Errorf("Message.String() = %v, should contain from: %v", str, tc.from)
    }
    if !strings.Contains(str, tc.content) {
        t.Errorf("Message.String() = %v, should contain content: %v", str, tc.content)
    }
}
		})
	}
}

func TestSystemMessage(t *testing.T) {
	content := "Test system message"
	msg := SystemMessage(content)

	if msg.From != "SYSTEM" {
		t.Errorf("SystemMessage.From = %v, want SYSTEM", msg.From)
	}

	if msg.Content != content {
		t.Errorf("SystemMessage.Content = %v, want %v", msg.Content, content)
	}

	if msg.Timestamp.IsZero() {
		t.Error("SystemMessage.Timestamp should not be zero")
	}
}

func TestNewMessage(t *testing.T) {
	from := "user1"
	content := "test message"
	beforeCreate := time.Now()
	msg := NewMessage(from, content)
	afterCreate := time.Now()

	if msg.From != from {
		t.Errorf("NewMessage.From = %v, want %v", msg.From, from)
	}

	if msg.Content != content {
		t.Errorf("NewMessage.Content = %v, want %v", msg.Content, content)
	}

	if msg.Timestamp.Before(beforeCreate) || msg.Timestamp.After(afterCreate) {
		t.Errorf("NewMessage.Timestamp = %v, want between %v and %v",
			msg.Timestamp, beforeCreate, afterCreate)
	}
}

func TestMessageTimestampFormat(t *testing.T) {
	msg := NewMessage("user1", "test")
	formatted := msg.String()

	expectedFormat := msg.Timestamp.Format(TimestampFormat)
	if !strings.Contains(formatted, expectedFormat) {
		t.Errorf("Message.String() = %v, should contain timestamp format: %v",
			formatted, expectedFormat)
	}
}

func TestMessageConstants(t *testing.T) {
	if MaxMessageSize <= 0 {
		t.Error("MaxMessageSize should be positive")
	}

	if MessageRateLimit <= 0 {
		t.Error("MessageRateLimit should be positive")
	}

	if TimestampFormat == "" {
		t.Error("TimestampFormat should not be empty")
	}
}

func TestMessageFormatCompliance(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		content  string
		wantErr  bool
		validate func(string) bool
	}{
		{
			name:    "Valid message format",
			from:    "user1",
			content: "Hello world",
			validate: func(s string) bool {
				// Format: [YYYY-MM-DD HH:MM:SS][username]:message
				pattern := `^\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\]\[user1\]:Hello world$`
				matched, _ := regexp.MatchString(pattern, s)
				return matched
			},
		},
		{
			name:    "Empty message",
			from:    "user1",
			content: "",
			wantErr: true,
			validate: func(s string) bool {
				return false // Empty messages should not be formatted
			},
		},
		{
			name:    "System notification",
			from:    "SYSTEM",
			content: "User2 has joined our chat...",
			validate: func(s string) bool {
				pattern := `^\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\]\[SYSTEM\]:User2 has joined our chat\.\.\.$`
				matched, _ := regexp.MatchString(pattern, s)
				return matched
			},
		},
		{
			name:    "Message with special characters",
			from:    "user1",
			content: "Hello! @#$%^&*()",
			validate: func(s string) bool {
				pattern := `^\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\]\[user1\]:Hello! @#\$%\^&\*\(\)$`
				matched, _ := regexp.MatchString(pattern, s)
				return matched
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := NewMessage(tt.from, tt.content)
			formatted := msg.String()

			if tt.wantErr {
				if tt.content == "" && formatted != "" {
					t.Error("Empty message should not be formatted")
				}
				return
			}

			if !tt.validate(formatted) {
				t.Errorf("Message format incorrect.\nGot: %s", formatted)
			}

			// Verify timestamp is current
			now := time.Now()
			if msg.Timestamp.Sub(now) > time.Second {
				t.Error("Message timestamp should be close to current time")
			}
		})
	}
}

func TestSystemNotificationFormat(t *testing.T) {
	tests := []struct {
		event    string
		username string
		want     string
	}{
		{"join", "user1", "has joined our chat..."},
		{"leave", "user2", "has left our chat..."},
	}

	for _, tt := range tests {
		t.Run(tt.event, func(t *testing.T) {
			msg := SystemMessage(tt.username + " " + tt.want)
			formatted := msg.String()
			if !strings.Contains(formatted, tt.username) || !strings.Contains(formatted, tt.want) {
				t.Errorf("Incorrect system notification format.\nGot: %s", formatted)
			}
		})
	}
}
