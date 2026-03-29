package db

import (
	"fmt"
	"log"
	"os"

	"github.com/deepansusingh/task-manager-api/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		log.Fatal("DB_URL is not set in .env")
	}

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}

	fmt.Println("Database connected successfully")

	// Auto-migrate creates tables if they don't exist
	err = DB.AutoMigrate(&model.User{}, &model.Task{})
	if err != nil {
		log.Fatal("Migration failed: ", err)
	}

	fmt.Println("Database migrated successfully")
}