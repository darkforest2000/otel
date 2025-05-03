package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"
	"wapp/handler"
	"wapp/storage"
	"wapp/usecase"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
)

// --- Main ---

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Set up OpenTelemetry.
	otelShutdown, err := setupOTelSDK(ctx)
	if err != nil {
		fmt.Printf("Failed to set up OpenTelemetry: %v\n", err)
		return
	}
	// Handle shutdown properly so nothing leaks.
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	logger := global.GetLoggerProvider().Logger("main")

	storage, err := storage.New(ctx)
	if err != nil {
		// Log error before panic
		errRecord := log.Record{}
		errRecord.SetTimestamp(time.Now())
		errRecord.SetSeverity(log.SeverityError)
		errRecord.SetSeverityText("ERROR")
		errRecord.SetBody(log.StringValue(fmt.Sprintf("Failed to connect to database: %v", err)))
		logger.Emit(context.Background(), errRecord)

		panic(fmt.Sprintf("Failed to connect to database: %v", err))
	}
	defer storage.Close()

	usecase := usecase.New(storage)
	handler := handler.New(usecase)
	srv := handler.NewServer(ctx)

	srvErr := make(chan error, 1)
	go func() {
		startRecord := log.Record{}
		startRecord.SetTimestamp(time.Now())
		startRecord.SetSeverity(log.SeverityInfo)
		startRecord.SetSeverityText("INFO")
		startRecord.SetBody(log.StringValue("Приложение запущено на порту 8080"))
		logger.Emit(context.Background(), startRecord)

		srvErr <- srv.ListenAndServe()
	}()

	// Wait for interruption.
	select {
	case err = <-srvErr:
		// Error when starting HTTP server.
		errRecord := log.Record{}
		errRecord.SetTimestamp(time.Now())
		errRecord.SetSeverity(log.SeverityError)
		errRecord.SetSeverityText("ERROR")
		errRecord.SetBody(log.StringValue(fmt.Sprintf("Error starting server: %v", err)))
		logger.Emit(context.Background(), errRecord)

		return
	case <-ctx.Done():
		// Wait for first CTRL+C.
		interruptRecord := log.Record{}
		interruptRecord.SetTimestamp(time.Now())
		interruptRecord.SetSeverity(log.SeverityInfo)
		interruptRecord.SetSeverityText("INFO")
		interruptRecord.SetBody(log.StringValue("Received interrupt signal, shutting down..."))
		logger.Emit(context.Background(), interruptRecord)

		// Stop receiving signal notifications as soon as possible.
		stop()
	}

	// When Shutdown is called, ListenAndServe immediately returns ErrServerClosed.
	err = srv.Shutdown(context.Background())
	shutdownRecord := log.Record{}
	shutdownRecord.SetTimestamp(time.Now())
	shutdownRecord.SetSeverity(log.SeverityInfo)
	shutdownRecord.SetSeverityText("INFO")
	shutdownRecord.SetBody(log.StringValue("Server shutdown completed"))
	if err != nil && !errors.Is(err, context.Canceled) {
		shutdownRecord.SetSeverity(log.SeverityError)
		shutdownRecord.SetSeverityText("ERROR")
		shutdownRecord.SetBody(log.StringValue(fmt.Sprintf("Server shutdown error: %v", err)))
	}
	logger.Emit(context.Background(), shutdownRecord)

	return
}
