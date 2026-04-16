package main

import (
	"log"

	"seller-trust-map/backend-go/internal/app"
)

func main() {
	server, err := app.NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := server.Run(); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
