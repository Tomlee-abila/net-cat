# TCP Chat Server

A concurrent TCP chat server implementation in Go with support for multiple clients and real-time message broadcasting.

## Features

- Concurrent client handling with goroutines
- Support for up to 10 simultaneous connections
- Real-time message broadcasting
- Username validation and uniqueness checks
- Message history for new connections
- Rate limiting and message size restrictions
- Graceful shutdown handling
- Persistent chat logging
- Configurable port (default: 8989)

## Requirements

- Go 1.19 or higher

## Building

Use the provided Makefile:

```bash
# Build the project
make build

# Run tests
make test

# Clean build artifacts
make clean
```

## Usage

```bash
# Start server with default port (8989)
./TCPChat

# Start server with custom port
./TCPChat 2525
```

## Client Connection

Connect using netcat or any TCP client:

```bash
nc localhost 8989
```

## Protocol

### Message Format
Messages follow the format: `[YYYY-MM-DD HH:MM:SS][username]: message`

### System Messages
- Join notification: `[SYSTEM] username has joined our chat...`
- Leave notification: `[SYSTEM] username has left our chat...`

## Features Details

### Connection Management
- Maximum 10 concurrent clients
- Automatic timeout after 5 minutes of inactivity
- TCP keepalive enabled
- Graceful connection handling

### Message Controls
- Empty messages are filtered
- Maximum message size: 1024 bytes
- Rate limiting: 1 message per second
- Real-time delivery (<1s latency)

### Username Rules
- Must be non-empty
- Maximum 32 characters
- No newlines or tabs
- Must be unique among active connections

## Testing

```bash
# Run all tests
make test

# Run with race detection
go test -race ./...
```

## Error Handling

- Invalid port numbers
- Connection limits
- Username conflicts
- Network errors
- Resource exhaustion

## Project Structure

- `main.go`: Core server implementation
- `main_test.go`: Test suite
- `Makefile`: Build and test automation
- `server_log.txt`: Message history log

## License

This project is open source and available under the MIT License.
