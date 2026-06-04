package main

import (
	"log"

	"github.com/joho/godotenv"

	"inkwords-backend/services/review-service/app/bootstrap"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default environment variables")
	}
}

func main() {
	if err := bootstrap.Run(); err != nil {
		log.Fatal(err)
	}
}
