package validation

import (
"fmt"
"strconv"
"strings"
)

// ValidatePort checks if a port number is valid
func ValidatePort(port string) error {
// Remove any square brackets if present
port = strings.Trim(port, "[]")

// Convert port to integer
portNum, err := strconv.Atoi(port)
if err != nil {
return fmt.Errorf("invalid port number: %s", port)
}

// Check port range
if portNum < 1 || portNum > 65535 {
return fmt.Errorf("port must be between 1-65535")
}

return nil
}

// IsValidPort returns true if the port number is valid
func IsValidPort(port string) bool {
return ValidatePort(port) == nil
}

// NormalizePort ensures the port string is properly formatted
func NormalizePort(port string) string {
return strings.Trim(port, "[]")
}
