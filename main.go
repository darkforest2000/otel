package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"wapp/handler"
	"wapp/logger"
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
		// Logging might not be available yet, print to stderr
		fmt.Fprintf(os.Stderr, "Failed to set up OpenTelemetry: %v\n", err)
		return
	}
	// Handle shutdown properly so nothing leaks.
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	lg := logger.NewTraceLogger(ctx, "main")

	storage, err := storage.New(ctx)
	if err != nil {
		// Use helper for logging
		lg.Fatal("Failed to connect to database", logger.Err(err))
		panic(fmt.Sprintf("Failed to connect to database: %v", err))
	}
	defer storage.Close()

	usecase := usecase.New(storage)
	handler := handler.New(usecase)
	srv := handler.NewServer(ctx)

	srvErr := make(chan error, 1)
	go func() {
		// Use helper for logging
		lg.Info("Приложение запущено на порту 8080")
		srvErr <- srv.ListenAndServe()
	}()

	// Wait for interruption.
	select {
	case err = <-srvErr:
		// Error when starting HTTP server.
		lg.Error("Error starting server", logger.Err(err))
		return
	case <-ctx.Done():
		// Wait for first CTRL+C.
		lg.Info("Received interrupt signal, shutting down...")
		// Stop receiving signal notifications as soon as possible.
		stop()
	}

	// When Shutdown is called, ListenAndServe immediately returns ErrServerClosed.
	shutdownCtx := context.Background() // Use a background context for shutdown
	err = srv.Shutdown(shutdownCtx)
	if err != nil && !errors.Is(err, context.Canceled) {
		lg.Error("Server shutdown error", logger.Err(err))
	} else {
		lg.Info("Server shutdown completed")
	}

	return
}
