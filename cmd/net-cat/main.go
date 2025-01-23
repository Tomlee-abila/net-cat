package main

import (
"fmt"
"log"
"os"
"strings"

"net-cat/internal/config"
"net-cat/internal/server"
"net-cat/pkg/validation"
)

// Make exit mockable for testing
var osExit = os.Exit

func main() {
if len(os.Args) > 2 {
fmt.Printf("[USAGE]: ./TCPChat $port\n")
osExit(1)
return
}

// Default port
port := "8989"

// Override with command line argument if provided
if len(os.Args) == 2 {
port = os.Args[1]
}

// Validate port
if err := validation.ValidatePort(port); err != nil {
fmt.Printf("[USAGE]: ./TCPChat $port\n")
osExit(1)
}

// Create server configuration
cfg := config.DefaultConfig().
WithListenAddr(":" + port)

// Create and start server
srv := server.New(cfg)

fmt.Printf("Starting TCP Chat server on port %s\n", strings.TrimPrefix(cfg.ListenAddr, ":"))

if err := srv.Start(); err != nil {
log.Printf("Server error: %v\n", err)
osExit(1)
}
}
