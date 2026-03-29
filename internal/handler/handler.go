package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/deepansusingh/task-manager-api/internal/db"
	"github.com/deepansusingh/task-manager-api/internal/middleware"
	"github.com/deepansusingh/task-manager-api/internal/model"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func response(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	body, _ := json.Marshal(data)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	w.WriteHeader(status)
	w.Write(body)
}

func generateToken(userID uint) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := os.Getenv("JWT_SECRET")
	return token.SignedString([]byte(secret))
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	response(w, http.StatusOK, map[string]string{"status": "ok"})
}

func Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if input.Name == "" || input.Email == "" || input.Password == "" {
		response(w, http.StatusBadRequest, map[string]string{"error": "name, email and password are required"})
		return
	}
	if len(input.Password) < 6 {
		response(w, http.StatusBadRequest, map[string]string{"error": "password must be at least 6 characters"})
		return
	}
	var existing model.User
	if result := db.DB.Where("email = ?", input.Email).First(&existing); result.Error == nil {
		response(w, http.StatusConflict, map[string]string{"error": "email already registered"})
		return
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		response(w, http.StatusInternalServerError, map[string]string{"error": "could not hash password"})
		return
	}
	user := model.User{
		Name:     input.Name,
		Email:    input.Email,
		Password: string(hashedPassword),
	}
	if result := db.DB.Create(&user); result.Error != nil {
		response(w, http.StatusInternalServerError, map[string]string{"error": "could not create user"})
		return
	}
	response(w, http.StatusCreated, map[string]any{
		"message": "user registered successfully",
		"user":    map[string]any{"id": user.ID, "name": user.Name, "email": user.Email},
	})
}

func Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if input.Email == "" || input.Password == "" {
		response(w, http.StatusBadRequest, map[string]string{"error": "email and password are required"})
		return
	}
	var user model.User
	if result := db.DB.Where("email = ?", input.Email).First(&user); result.Error != nil {
		response(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		response(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		return
	}
	token, err := generateToken(user.ID)
	if err != nil {
		response(w, http.StatusInternalServerError, map[string]string{"error": "could not generate token"})
		return
	}
	response(w, http.StatusOK, map[string]any{
		"message": "login successful",
		"token":   token,
		"user":    map[string]any{"id": user.ID, "name": user.Name, "email": user.Email},
	})
}

// Tasks handles GET /tasks and POST /tasks
// Supports query param filtering: GET /tasks?status=pending
func Tasks(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(uint)

	switch r.Method {

	case http.MethodGet:
		var tasks []model.Task

		// Start building the query — always scope to logged-in user
		query := db.DB.Where("user_id = ?", userID)

		// Check if ?status= query param was provided
		// r.URL.Query() parses the query string into a map
		status := r.URL.Query().Get("status")

		if status != "" {
			// Validate it's one of the allowed values
			allowed := map[string]bool{
				"pending":     true,
				"in_progress": true,
				"done":        true,
			}
			if !allowed[status] {
				response(w, http.StatusBadRequest, map[string]string{
					"error": "invalid status, must be: pending, in_progress, or done",
				})
				return
			}
			// Chain an additional WHERE clause onto the query
			query = query.Where("status = ?", status)
		}

		// Execute the query — either:
		// SELECT * FROM tasks WHERE user_id = ?
		// or
		// SELECT * FROM tasks WHERE user_id = ? AND status = ?
		if result := query.Find(&tasks); result.Error != nil {
			response(w, http.StatusInternalServerError, map[string]string{"error": "could not fetch tasks"})
			return
		}

		// Build response — include the active filter in the response
		// so the client knows what was applied
		resp := map[string]any{
			"tasks": tasks,
			"count": len(tasks),
		}
		if status != "" {
			resp["filter"] = map[string]string{"status": status}
		}

		response(w, http.StatusOK, resp)

	case http.MethodPost:
		var input struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			response(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if strings.TrimSpace(input.Title) == "" {
			response(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
			return
		}
		task := model.Task{
			Title:       input.Title,
			Description: input.Description,
			Status:      "pending",
			UserID:      userID,
		}
		if result := db.DB.Create(&task); result.Error != nil {
			response(w, http.StatusInternalServerError, map[string]string{"error": "could not create task"})
			return
		}
		response(w, http.StatusCreated, map[string]any{
			"message": "task created successfully",
			"task":    task,
		})

	default:
		response(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// TaskByID handles GET, PUT, DELETE /tasks/:id
func TaskByID(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(uint)

	idStr := r.URL.Path[len("/tasks/"):]
	if idStr == "" {
		response(w, http.StatusBadRequest, map[string]string{"error": "missing task id"})
		return
	}
	taskID, err := strconv.Atoi(idStr)
	if err != nil || taskID <= 0 {
		response(w, http.StatusBadRequest, map[string]string{"error": "invalid task id"})
		return
	}

	switch r.Method {

	case http.MethodGet:
		var task model.Task
		if result := db.DB.Where("id = ? AND user_id = ?", taskID, userID).First(&task); result.Error != nil {
			response(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		response(w, http.StatusOK, map[string]any{"task": task})

	case http.MethodPut:
		var task model.Task
		if result := db.DB.Where("id = ? AND user_id = ?", taskID, userID).First(&task); result.Error != nil {
			response(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		var input struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Status      string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			response(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if input.Title != "" {
			task.Title = input.Title
		}
		if input.Description != "" {
			task.Description = input.Description
		}
		if input.Status != "" {
			allowed := map[string]bool{"pending": true, "in_progress": true, "done": true}
			if !allowed[input.Status] {
				response(w, http.StatusBadRequest, map[string]string{"error": "status must be: pending, in_progress, or done"})
				return
			}
			task.Status = input.Status
		}
		db.DB.Save(&task)
		response(w, http.StatusOK, map[string]any{
			"message": "task updated successfully",
			"task":    task,
		})

	case http.MethodDelete:
		var task model.Task
		if result := db.DB.Where("id = ? AND user_id = ?", taskID, userID).First(&task); result.Error != nil {
			response(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		db.DB.Delete(&task)
		response(w, http.StatusOK, map[string]string{"message": "task deleted successfully"})

	default:
		response(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}git remote add origin https://github.com/Deepansusingh/Task-Manger-api.git