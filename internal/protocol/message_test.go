package protocol

import (
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
if !strings.Contains(str, tc.from) {
t.Errorf("Message.String() = %v, should contain from: %v", str, tc.from)
}
if !strings.Contains(str, tc.content) {
t.Errorf("Message.String() = %v, should contain content: %v", str, tc.content)
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

// Check timestamp is within expected range
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
