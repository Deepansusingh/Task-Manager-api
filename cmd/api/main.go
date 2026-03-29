package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Deepansusingh/Task-Manager-api/internal/db"
	"github.com/Deepansusingh/Task-Manager-api/internal/handler"
	"github.com/Deepansusingh/Task-Manager-api/internal/middleware"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env locally — ignore error in production (Railway injects env vars directly)
	godotenv.Load()

	// Connect to database and run migrations
	db.Connect()

	// Set up routes
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/health", handler.HealthCheck)
	mux.HandleFunc("/auth/register", handler.Register)
	mux.HandleFunc("/auth/login", handler.Login)

	// Protected routes
	mux.HandleFunc("/tasks", middleware.AuthMiddleware(handler.Tasks))
	mux.HandleFunc("/tasks/", middleware.AuthMiddleware(handler.TaskByID))

	// Railway injects PORT automatically — fallback to 9090 for local dev
	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}

	fmt.Println("Server running on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}