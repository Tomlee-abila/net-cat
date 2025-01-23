package client

import (
"bufio"
"fmt"
"net"
"strings"

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
if strings.ContainsAny(name, "\n\r\t") {
return errors.New(errors.ErrValidation, "invalid characters in username", nil)
}

if len(strings.TrimSpace(name)) == 0 {
return errors.New(errors.ErrValidation, "username cannot be empty", nil)
}

if len(name) > maxLength {
return errors.New(errors.ErrValidation, fmt.Sprintf("username too long (max %d characters)", maxLength), nil)
}

return nil
}

// SetupTCPConn configures TCP-specific connection settings
func SetupTCPConn(conn net.Conn, timeout config.Config) error {
tcpConn, ok := conn.(*net.TCPConn)
if !ok {
return errors.New(errors.ErrConnection, "connection is not TCP", nil)
}

if err := tcpConn.SetKeepAlive(true); err != nil {
return errors.New(errors.ErrConnection, "failed to set keep-alive", nil)
}

if err := tcpConn.SetKeepAlivePeriod(timeout.ClientTimeout); err != nil {
return errors.New(errors.ErrConnection, "failed to set keep-alive period", nil)
}

return nil
}
