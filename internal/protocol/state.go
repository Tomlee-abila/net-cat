package protocol

// ConnectionState represents the current state of a client connection
type ConnectionState int

const (
	// StateConnecting indicates the client is in the process of connecting
	StateConnecting ConnectionState = iota

	// StateAuthenticated indicates the client has successfully authenticated
	StateAuthenticated

	// StateActive indicates the client is connected and can send/receive messages
	StateActive

	// StateDisconnecting indicates the client is in the process of disconnecting
	StateDisconnecting
)

// String returns a string representation of the ConnectionState
func (s ConnectionState) String() string {
	switch s {
	case StateConnecting:
		return "Connecting"
	case StateAuthenticated:
		return "Authenticated"
	case StateActive:
		return "Active"
	case StateDisconnecting:
		return "Disconnecting"
	default:
		return "Unknown"
	}
}
