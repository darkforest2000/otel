package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"wapp/handler"
	"wapp/storage"
	"wapp/usecase"
)

// --- Main ---

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Set up OpenTelemetry.
	otelShutdown, err := setupOTelSDK(ctx)
	if err != nil {
		log.Fatalf("Failed to set up OpenTelemetry: %v", err)
		return
	}
	// Handle shutdown properly so nothing leaks.
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	storage, err := storage.NewStorage(ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to database: %v", err))
	}
	defer storage.Close()

	usecase := usecase.New(storage)
	handler := handler.New(usecase)
	srv := handler.NewServer(ctx)

	srvErr := make(chan error, 1)
	go func() {
		log.Printf("Starting server on :8080")
		srvErr <- srv.ListenAndServe()
	}()

	// Wait for interruption.
	select {
	case err = <-srvErr:
		// Error when starting HTTP server.
		log.Printf("Error starting server: %v", err)
		return
	case <-ctx.Done():
		// Wait for first CTRL+C.
		log.Println("Received interrupt signal, shutting down...")
		// Stop receiving signal notifications as soon as possible.
		stop()
	}

	// When Shutdown is called, ListenAndServe immediately returns ErrServerClosed.
	err = srv.Shutdown(context.Background())
	return
}
