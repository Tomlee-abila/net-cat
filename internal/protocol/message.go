package protocol

import (
    "fmt"
    "time"
)

const (
    // TimestampFormat defines how timestamps should be formatted in messages
    TimestampFormat = "2006-01-02 15:04:05"

    // MaxMessageSize is the maximum allowed length of message content
    MaxMessageSize = 1024

    // MessageRateLimit is the minimum time between messages from a client
    MessageRateLimit = time.Second
)

// Message represents a chat message
type Message struct {
    From      string
    Content   string
    Timestamp time.Time
}

// String returns a formatted string representation of the message
func (m Message) String() string {
    if m.Content == "" {
        return ""
    }
    return fmt.Sprintf("[%s][%s]:%s",
        m.Timestamp.Format(TimestampFormat),
        m.From,
        m.Content,
    )
}

// NewMessage creates a new message from the given sender and content
func NewMessage(from, content string) Message {
    return Message{
        From:      from,
        Content:   content,
        Timestamp: time.Now(),
    }
}

// SystemMessage creates a new system message with the given content
func SystemMessage(content string) Message {
    return Message{
        From:      "SYSTEM",
        Content:   content,
        Timestamp: time.Now(),
    }
}
