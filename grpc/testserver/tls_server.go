package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Create TLS server on port 50052
	server := NewServer()
	server.SetPort("50052")
	server.SetTLS(true, "../../testdata/certs/server.crt", "../../testdata/certs/server.key")

	fmt.Printf("Starting TLS gRPC server on :50052...\n")

	// Start server in background
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("Failed to start TLS server: %v", err)
		}
	}()

	fmt.Printf("TLS gRPC server running at %s\n", server.address)

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("\nShutting down TLS server...")
	server.Stop()
}
