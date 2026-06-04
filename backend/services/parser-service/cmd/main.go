package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"inkwords-backend/services/parser-service/app/bootstrap"
	parsedomain "inkwords-backend/services/parser-service/domain/parse"
	"inkwords-backend/shared/kernel/httpx"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default environment variables")
	}
}

func main() {
	r, parseService, taskService, err := bootstrap.BuildRouter()
	if err != nil {
		log.Fatalf("bootstrap parser-service failed: %v", err)
	}

	server := httpx.NewServer(r)
	signalContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	stopConsumer, err := parsedomain.StartParseConsumer(signalContext, taskService, parseService)
	if err != nil {
		log.Printf("RabbitMQ parse consumer initialization skipped: %v", err)
	}
	defer stopConsumer()

	go func() {
		if err := httpx.ShutdownOnContextDone(signalContext, server, 15*time.Second); err != nil {
			log.Printf("Server shutdown failed: %v", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Server startup failed: %v", err)
	}
}
