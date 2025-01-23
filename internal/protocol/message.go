package protocol

import (
	"fmt"
	"time"
)

// Message represents a chat message with metadata
type Message struct {
	From      string
	Content   string
	Timestamp time.Time
}

// String returns a formatted string representation of the message
func (m Message) String() string {
	return fmt.Sprintf("[%s][%s]:%s",
		m.Timestamp.Format("2006-01-02 15:04:05"),
		m.From,
		m.Content,
	)
}

// NewMessage creates a new message with the current timestamp
func NewMessage(from, content string) Message {
	return Message{
		From:      from,
		Content:   content,
		Timestamp: time.Now(),
	}
}

// SystemMessage creates a new system message
func SystemMessage(content string) Message {
	return NewMessage("SYSTEM", content)
}

const (
	// MaxMessageSize defines the maximum allowed message size in bytes
	MaxMessageSize = 1024

	// MessageRateLimit defines the minimum time between messages
	MessageRateLimit = time.Second

	// TimestampFormat defines the standard timestamp format for messages
	TimestampFormat = "2006-01-02 15:04:05"
)
