package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"net-cat/internal/config"
	"net-cat/internal/server"
	"net-cat/pkg/validation"
)

// Make exit mockable for testing
var (
	exitMutex sync.Mutex
	osExit    = os.Exit
)

// getExit returns the current exit function in a thread-safe way
func getExit() func(int) {
	exitMutex.Lock()
	defer exitMutex.Unlock()
	return osExit
}

// setExit sets the exit function in a thread-safe way
func setExit(exit func(int)) {
	exitMutex.Lock()
	defer exitMutex.Unlock()
	osExit = exit
}

func main() {
	// Validate command line arguments
	if len(os.Args) > 2 {
		log.Printf("Error: too many arguments\n")
		fmt.Printf("[USAGE]: ./TCPChat $port\n")
		getExit()(1)
	}

	// Default port
	port := "8989"
	if len(os.Args) == 2 {
		port = os.Args[1]
	}

	// Validate port early
	if err := validation.ValidatePort(port); err != nil {
		log.Printf("Error: invalid port: %v\n", err)
		fmt.Printf("[USAGE]: ./TCPChat $port\n")
		getExit()(1)
	}

	// Create server configuration
	cfg := config.DefaultConfig().WithListenAddr(":" + port)
	srv := server.New(cfg)

	log.Printf("Starting TCP Chat server on port %s\n", strings.TrimPrefix(cfg.ListenAddr, ":"))

	// Start server and handle errors
	errCh := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil {
			errCh <- err
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait for either server error or interrupt signal
	select {
	case err := <-errCh:
		log.Printf("Server error: %v\n", err)
		getExit()(1)
	case <-sigCh:
		if err := srv.Stop(); err != nil {
			log.Printf("Error stopping server: %v\n", err)
		}
	}
}
