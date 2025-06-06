package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/WillKirkmanM/proxy/internal/config"
	"github.com/WillKirkmanM/proxy/internal/proxy"
)

// main initializes and starts the reverse proxy server
// This function orchestrates the entire application lifecycle including:
// - Configuration loading and validation
// - Server initialisation with graceful shutdown support
// - Signal handling for clean termination
// Time Complexity: O(1) - constant initialisation time
// Space Complexity: O(1) - fixed memory allocation
func main() {
    var configPath = flag.String("config", "config.yaml", "Path to configuration file")
    flag.Parse()

    // Load configuration using singleton pattern
    // This ensures only one configuration instance exists throughout the application

	// Or load from file
	err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	cfg := config.GetInstance()


    // Create proxy server instance using factory pattern
    // The factory handles complex initialisation logic and dependency injection
    server, err := proxy.NewServer(cfg)
    if err != nil {
        log.Fatalf("Failed to create proxy server: %v", err)
    }

    // Setup graceful shutdown using context cancellation
    // This pattern ensures all goroutines are properly terminated
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Channel for OS signals - enables graceful shutdown on SIGINT/SIGTERM
    // Buffer size of 1 prevents blocking on signal delivery
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    // Start server in separate goroutine to prevent blocking main thread
    // This allows concurrent signal handling and server operation
    go func() {
        log.Printf("Starting proxy server on port %s", strconv.Itoa(cfg.Server.Port))
        if err := server.Start(ctx); err != nil {
            log.Fatalf("Server failed to start: %v", err)
        }
    }()

    // Block until termination signal is received
    // This implements the main event loop pattern
    <-sigChan
    log.Println("Received termination signal, shutting down gracefully...")

    // Cancel context to signal all components to shutdown
    cancel()

    // Allow time for graceful shutdown before forced termination
    // 30 second timeout prevents indefinite hanging during shutdown
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer shutdownCancel()

    if err := server.Shutdown(shutdownCtx); err != nil {
        log.Printf("Error during shutdown: %v", err)
    }

    log.Println("Proxy server stopped")
}