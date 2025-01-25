package client

import (
    "bufio"
    "fmt"
    "net"
    "strings"
    "time"

    "net-cat/internal/config"
    "net-cat/internal/errors"
)

const welcomeBanner = `Welcome to Linux TCP-Chat!
         _nnnn_
        dGGGGMMb
       @p~qp~~qMb
       M|@||@) M|
       @,----.JM|
      JS^\__/  qKL
     dZP        qKRb
    dZP          qKKb
   fZP            SMMb
   HZM            MMMM
   FqM            MMMM
 __| ".        |\dS"qML
 |    '.       | '  \Zq
_)      \.___.,|     .'
\____   )MMMMMP|   .'
     '-'       '--'
[ENTER YOUR NAME]:`

// tcpKeepAliver is an interface for connections that support keepalive
type tcpKeepAliver interface {
    SetKeepAlive(keepalive bool) error
    SetKeepAlivePeriod(d time.Duration) error
}

// Authenticate handles the client authentication process
func Authenticate(conn net.Conn, cfg *config.Config) (string, error) {
    // Send welcome banner
    if _, err := conn.Write([]byte(welcomeBanner + "\n")); err != nil {
        return "", errors.New(errors.ErrConnection, "failed to send welcome banner", nil)
    }

    // Read username
    reader := bufio.NewReader(conn)
    name, err := reader.ReadString('\n')
    if err != nil {
        return "", errors.New(errors.ErrValidation, "failed to read username", nil)
    }

    name = strings.TrimSpace(name)
    if err := ValidateUsername(name, cfg.MaxNameLength); err != nil {
        return "", err
    }

    return name, nil
}

// ValidateUsername checks if a username is valid
func ValidateUsername(name string, maxLength int) error {
    trimmed := strings.TrimSpace(name)

    // Check for empty name
    if len(trimmed) == 0 {
        return errors.New(errors.ErrValidation, "username cannot be empty", nil)
    }

    // Check for leading/trailing spaces
    if trimmed != name {
        return errors.New(errors.ErrValidation, "username cannot have leading or trailing spaces", nil)
    }

    // Check length
    if len(name) > maxLength {
        return errors.New(errors.ErrValidation, fmt.Sprintf("username too long (max %d characters)", maxLength), nil)
    }

    // Check for valid characters (letters, numbers, underscores, and spaces allowed)
    for _, char := range name {
        if !((char >= 'a' && char <= 'z') ||
            (char >= 'A' && char <= 'Z') ||
            (char >= '0' && char <= '9') ||
            char == '_' || char == ' ') {
            return errors.New(errors.ErrValidation, "username can only contain letters, numbers, spaces, and underscores", nil)
        }
    }

    return nil
}

// SetupTCPConn configures TCP-specific connection settings
func SetupTCPConn(conn net.Conn, timeout config.Config) error {
    // Check if connection supports keepalive
    ka, ok := conn.(tcpKeepAliver)
    if !ok {
        return errors.New(errors.ErrConnection, "connection does not support keepalive", nil)
    }

    if err := ka.SetKeepAlive(true); err != nil {
        return errors.New(errors.ErrConnection, "failed to set keep-alive", nil)
    }

    if err := ka.SetKeepAlivePeriod(timeout.ClientTimeout); err != nil {
        return errors.New(errors.ErrConnection, "failed to set keep-alive period", nil)
    }

    return nil
}
