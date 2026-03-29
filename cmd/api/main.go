package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/deepansusingh/task-manager-api/internal/db"
	"github.com/deepansusingh/task-manager-api/internal/handler"
	"github.com/deepansusingh/task-manager-api/internal/middleware"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	db.Connect()

	mux := http.NewServeMux()

	// Public routes — no token needed
	mux.HandleFunc("/health", handler.HealthCheck)
	mux.HandleFunc("/auth/register", handler.Register)
	mux.HandleFunc("/auth/login", handler.Login)

	// Protected routes — token required
	// middleware.AuthMiddleware wraps the handler
	// it verifies the JWT before the handler ever runs
	mux.HandleFunc("/tasks", middleware.AuthMiddleware(handler.Tasks))
	mux.HandleFunc("/tasks/", middleware.AuthMiddleware(handler.TaskByID))

	port := ":" + os.Getenv("PORT")
	fmt.Println("Server running on http://localhost" + port)
	log.Fatal(http.ListenAndServe(port, mux))
}