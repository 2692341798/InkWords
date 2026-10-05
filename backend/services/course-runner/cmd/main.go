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
	"inkwords-backend/services/course-runner/app/bootstrap"
	textbookverification "inkwords-backend/services/course-runner/domain/textbookverification"
	"inkwords-backend/shared/kernel/httpx"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default environment variables")
	}
}

func main() {
	r, textbookConsumer, err := bootstrap.BuildRouter()
	if err != nil {
		log.Fatalf("bootstrap course-runner failed: %v", err)
	}
	server := httpx.NewServer(r)
	signalContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	stopTextbookConsumer, err := textbookverification.StartConsumer(signalContext, textbookConsumer, "inkwords.textbook-verification")
	if err != nil {
		log.Printf("RabbitMQ textbook verification consumer initialization failed: %v", err)
	}
	defer stopTextbookConsumer()
	go func() {
		if err := httpx.ShutdownOnContextDone(signalContext, server, 15*time.Second); err != nil {
			log.Printf("Server shutdown failed: %v", err)
		}
	}()
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("Server startup failed: %v", err)
	}
}
