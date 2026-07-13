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
	verification "inkwords-backend/services/course-runner/domain/verification"
	"inkwords-backend/shared/kernel/httpx"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default environment variables")
	}
}

func main() {
	r, consumer, err := bootstrap.BuildRouter()
	if err != nil {
		log.Fatalf("bootstrap course-runner failed: %v", err)
	}
	server := httpx.NewServer(r)
	signalContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	stopConsumer, err := verification.StartVerificationConsumer(signalContext, consumer, "inkwords.course-verification")
	if err != nil {
		log.Printf("RabbitMQ course verification consumer initialization failed: %v", err)
	}
	defer stopConsumer()
	go func() {
		if err := httpx.ShutdownOnContextDone(signalContext, server, 15*time.Second); err != nil {
			log.Printf("Server shutdown failed: %v", err)
		}
	}()
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("Server startup failed: %v", err)
	}
}
